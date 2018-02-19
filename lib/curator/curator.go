/*
Copyright (c) 2014 Ashley Jeffs

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, sub to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package curator

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Jeffail/leaps/lib/acl"
	"github.com/Jeffail/leaps/lib/audit"
	"github.com/Jeffail/leaps/lib/binder"
	"github.com/Jeffail/leaps/lib/store"
	"github.com/Jeffail/leaps/lib/util/service/log"
	"github.com/Jeffail/leaps/lib/util/service/metrics"
)

//------------------------------------------------------------------------------

// Config - Holds configuration options for a curator.
type Config struct {
	BinderConfig binder.Config `json:"binder" yaml:"binder"`
}

// NewConfig - Returns a fully defined curator configuration with the default
// values for each field.
func NewConfig() Config {
	return Config{
		BinderConfig: binder.NewConfig(),
	}
}

//------------------------------------------------------------------------------

// Errors for the Curator type.
var (
	ErrBinderNotFound = errors.New("binder was not found")
)

// Impl - The underlying implementation of the curator type. Creates and manages
// the entire lifecycle of binders internally.
type Impl struct {
	config   Config
	store    store.Type
	auth     acl.Authenticator
	auditors AuditorContainer

	log   log.Modular
	stats metrics.Type

	// Binders
	openBinders map[string]binder.Type
	binderMutex sync.RWMutex

	// Control channels
	errorChan  chan binder.Error
	closeChan  chan struct{}
	closedChan chan struct{}
}

// New - Creates and returns a new curator, and launches its internal loop.
func New(
	config Config,
	log log.Modular,
	stats metrics.Type,
	auth acl.Authenticator,
	store store.Type,
	auditors AuditorContainer,
) (*Impl, error) {

	curator := Impl{
		config:      config,
		store:       store,
		log:         log.NewModule(":curator"),
		stats:       stats,
		auth:        auth,
		auditors:    auditors,
		openBinders: make(map[string]binder.Type),
		errorChan:   make(chan binder.Error, 10),
		closeChan:   make(chan struct{}),
		closedChan:  make(chan struct{}),
	}
	go curator.loop()

	return &curator, nil
}

// Close - Shut the curator and all subsequent binders down. This call blocks
// until the shut down is finished, and you must ensure that this curator cannot
// be accessed after closing.
func (c *Impl) Close() {
	c.log.Debugln("Close called")
	c.closeChan <- struct{}{}
	<-c.closedChan
}

/*
loop - The main loop of the curator. Two channels are listened to:

- Error channel, used by active binders to request a shut down, either due to
  inactivity or an error having occurred. The curator then calls close on it and
  removes it from the list of binders.

- Close channel, used by the owner of the curator to instigate a clean shut
  down. The curator then forwards to call to all binders and closes itself.
*/
func (c *Impl) loop() {
	c.log.Debugln("Loop called")
	for {
		select {
		case err := <-c.errorChan:
			if err.Err != nil {
				c.stats.Incr("curator.binder_chan.error", 1)
				c.log.Errorf("Binder (%v) %v\n", err.ID, err.Err)
			} else {
				c.log.Infof("Binder (%v) has requested shutdown\n", err.ID)
			}
			c.binderMutex.Lock()
			if b, ok := c.openBinders[err.ID]; ok {
				b.Close()
				delete(c.openBinders, err.ID)
				c.log.Infof("Binder (%v) was closed\n", err.ID)
				c.stats.Incr("curator.binder_shutdown.success", 1)
				c.stats.Decr("curator.open_binders", 1)
			} else {
				c.log.Errorf("Binder (%v) was not located in map\n", err.ID)
				c.stats.Incr("curator.binder_shutdown.error", 1)
			}
			c.binderMutex.Unlock()
		case <-c.closeChan:
			c.log.Infoln("Received call to close, forwarding message to binders")
			c.binderMutex.Lock()
			for _, b := range c.openBinders {
				b.Close()
				c.stats.Decr("curator.open_binders", 1)
			}
			c.binderMutex.Unlock()
			close(c.closedChan)
			return
		}
	}
}

func (c *Impl) newBinder(id string) (binder.Type, error) {
	var auditor audit.Auditor
	var err error
	if c.auditors != nil {
		if auditor, err = c.auditors.Get(id); err != nil {
			return nil, fmt.Errorf("failed to create auditor: %v", err)
		}
	}
	return binder.New(
		id, c.store, c.config.BinderConfig, c.errorChan, c.log, c.stats, auditor,
	)
}

//------------------------------------------------------------------------------

// EditDocument - Locates or creates a Binder for an existing document and
// returns that Binder for subscribing to. Returns an error if there was a
// problem locating the document.
func (c *Impl) EditDocument(
	userMetadata interface{}, token, documentID string, timeout time.Duration,
) (binder.Portal, error) {
	c.log.Debugf("finding document %v, with userMetadata %v token %v\n", documentID, userMetadata, token)

	if c.auth.Authenticate(userMetadata, token, documentID) < acl.EditAccess {
		c.stats.Incr("curator.edit.rejected_client", 1)
		return nil, fmt.Errorf(
			"failed to authorise join of document id: %v with token: %v", documentID, token,
		)
	}
	c.stats.Incr("curator.edit.accepted_client", 1)

	c.binderMutex.Lock()

	// Check for existing binder
	if openBinder, ok := c.openBinders[documentID]; ok {
		c.binderMutex.Unlock()
		return openBinder.Subscribe(userMetadata, timeout)
	}
	openBinder, err := c.newBinder(documentID)
	if err != nil {
		c.binderMutex.Unlock()

		c.stats.Incr("curator.bind_new.failed", 1)
		c.log.Errorf("Failed to bind to document %v: %v\n", documentID, err)
		return nil, err
	}
	c.openBinders[documentID] = openBinder
	c.binderMutex.Unlock()

	c.stats.Incr("curator.open_binders", 1)
	return openBinder.Subscribe(userMetadata, timeout)
}

// ReadDocument - Locates or creates a Binder for an existing document and
// returns that Binder for subscribing to with read only privileges. Returns an
// error if there was a problem locating the document.
func (c *Impl) ReadDocument(
	userMetadata interface{}, token, documentID string, timeout time.Duration,
) (binder.Portal, error) {
	c.log.Debugf("finding document %v, with userMetadata %v token %v\n", documentID, userMetadata, token)

	if c.auth.Authenticate(userMetadata, token, documentID) < acl.ReadAccess {
		c.stats.Incr("curator.read.rejected_client", 1)
		return nil, fmt.Errorf(
			"failed to authorise read only join of document id: %v with token: %v",
			documentID, token,
		)
	}
	c.stats.Incr("curator.read.accepted_client", 1)

	c.binderMutex.Lock()

	// Check for existing binder
	if openBinder, ok := c.openBinders[documentID]; ok {
		c.binderMutex.Unlock()
		return openBinder.SubscribeReadOnly(userMetadata, timeout)
	}
	openBinder, err := c.newBinder(documentID)
	if err != nil {
		c.binderMutex.Unlock()

		c.stats.Incr("curator.bind_existing.failed", 1)
		c.log.Errorf("Failed to bind to document %v: %v\n", documentID, err)
		return nil, err
	}
	c.openBinders[documentID] = openBinder
	c.binderMutex.Unlock()

	c.stats.Incr("curator.open_binders", 1)
	return openBinder.SubscribeReadOnly(userMetadata, timeout)
}

// CreateDocument - Creates a fresh Binder for a new document, which is
// subsequently stored, returns an error if either the document ID is already
// currently in use, or if there is a problem storing the new document. May
// require authentication, if so a userMetadata is supplied.
func (c *Impl) CreateDocument(
	userMetadata interface{}, token string, doc store.Document, timeout time.Duration,
) (binder.Portal, error) {
	c.log.Debugf("Creating new document with userMetadata %v token %v\n", userMetadata, token)

	if c.auth.Authenticate(userMetadata, token, "") < acl.CreateAccess {
		c.stats.Incr("curator.create.rejected_client", 1)
		return nil, fmt.Errorf("failed to gain permission to create with token: %v", token)
	}
	c.stats.Incr("curator.create.accepted_client", 1)

	if err := c.store.Create(doc); err != nil {
		c.stats.Incr("curator.create_new.failed", 1)
		c.log.Errorf("Failed to create new document: %v\n", err)
		return nil, err
	}
	openBinder, err := c.newBinder(doc.ID)
	if err != nil {
		c.stats.Incr("curator.bind_new.failed", 1)
		c.log.Errorf("Failed to bind to new document: %v\n", err)
		return nil, err
	}
	c.binderMutex.Lock()
	c.openBinders[doc.ID] = openBinder
	c.binderMutex.Unlock()
	c.stats.Incr("curator.open_binders", 1)

	return openBinder.Subscribe(userMetadata, timeout)
}

//------------------------------------------------------------------------------
