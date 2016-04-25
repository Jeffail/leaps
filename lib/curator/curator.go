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

	"github.com/jeffail/leaps/lib/acl"
	"github.com/jeffail/leaps/lib/binder"
	"github.com/jeffail/leaps/lib/store"
	"github.com/jeffail/leaps/lib/util"
	"github.com/jeffail/util/log"
	"github.com/jeffail/util/metrics"
)

//--------------------------------------------------------------------------------------------------

// Config - Holds configuration options for a curator.
type Config struct {
	BinderConfig binder.Config `json:"binder" yaml:"binder"`
}

// NewConfig - Returns a fully defined curator configuration with the default values for each field.
func NewConfig() Config {
	return Config{
		BinderConfig: binder.NewConfig(),
	}
}

//--------------------------------------------------------------------------------------------------

// Errors for the Curator type.
var (
	ErrBinderNotFound = errors.New("binder was not found")
)

/*
impl - The underlying implementation of the curator type. Creates and manages the entire lifecycle
of binders internally.
*/
type impl struct {
	config Config
	store  store.Type
	auth   acl.Authenticator

	log   log.Modular
	stats metrics.Aggregator

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
	stats metrics.Aggregator,
	auth acl.Authenticator,
	store store.Type,
) (Type, error) {

	curator := impl{
		config:      config,
		store:       store,
		log:         log.NewModule(":curator"),
		stats:       stats,
		auth:        auth,
		openBinders: make(map[string]binder.Type),
		errorChan:   make(chan binder.Error, 10),
		closeChan:   make(chan struct{}),
		closedChan:  make(chan struct{}),
	}
	go curator.loop()

	return &curator, nil
}

/*
Close - Shut the curator and all subsequent binders down. This call blocks until the shut down is
finished, and you must ensure that this curator cannot be accessed after closing.
*/
func (c *impl) Close() {
	c.log.Debugln("Close called")
	c.closeChan <- struct{}{}
	<-c.closedChan
}

/*
loop - The main loop of the curator. Two channels are listened to:

- Error channel, used by active binders to request a shut down, either due to inactivity or an error
having occurred. The curator then calls close on it and removes it from the list of binders.

- Close channel, used by the owner of the curator to instigate a clean shut down. The curator then
forwards to call to all binders and closes itself.
*/
func (c *impl) loop() {
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

/*--------------------------------------------------------------------------------------------------
 */

// KickUser - Remove a user from a document, requires the respective user and document IDs.
func (c *impl) KickUser(documentID, userID string, timeout time.Duration) error {
	c.log.Debugf("attempting to kick user %v from document %v\n", documentID, userID)

	c.binderMutex.Lock()

	// Check for existing binder
	binder, ok := c.openBinders[documentID]

	c.binderMutex.Unlock()

	if !ok {
		c.stats.Incr("curator.kick_user.error", 1)
		c.log.Errorf("Failed to kick user %v from %v: Document was not open\n", userID, documentID)
		return ErrBinderNotFound
	}

	if err := binder.KickUser(userID, timeout); err != nil {
		c.stats.Incr("curator.kick_user.error", 1)
		return err
	}

	c.stats.Incr("curator.kick_user.success", 1)
	return nil
}

// GetUsers - Return a full list of all connected users of all open documents.
func (c *impl) GetUsers(timeout time.Duration) (map[string][]string, error) {
	openBinders := []binder.Type{}

	c.binderMutex.Lock()
	for _, binder := range c.openBinders {
		openBinders = append(openBinders, binder)
	}
	c.binderMutex.Unlock()

	started := time.Now()

	list, listMutex := map[string][]string{}, sync.Mutex{}

	wg := sync.WaitGroup{}
	wg.Add(len(openBinders))

	for _, b := range openBinders {
		go func(tmpBinder binder.Type) {
			defer wg.Done()

			users, err := tmpBinder.GetUsers(timeout - time.Since(started))
			if err != nil {
				c.stats.Incr("curator.get_users.error", 1)
				c.log.Errorf("Failed to get users list from %v\n", tmpBinder.ID())
			} else if len(users) > 0 {
				listMutex.Lock()
				list[tmpBinder.ID()] = users
				listMutex.Unlock()
			}
		}(b)
	}

	wg.Wait()

	c.stats.Incr("curator.get_users.success", 1)
	return list, nil
}

/*
EditDocument - Locates or creates a Binder for an existing document and returns that Binder for
subscribing to. Returns an error if there was a problem locating the document.
*/
func (c *impl) EditDocument(
	userID, token, documentID string, timeout time.Duration,
) (binder.Portal, error) {
	c.log.Debugf("finding document %v, with userID %v token %v\n", documentID, userID, token)

	if c.auth.Authenticate(userID, token, documentID) < acl.EditAccess {
		c.stats.Incr("curator.edit.rejected_client", 1)
		return nil, fmt.Errorf(
			"failed to authorise join of document id: %v with token: %v\n", documentID, token,
		)
	}
	c.stats.Incr("curator.edit.accepted_client", 1)

	c.binderMutex.Lock()

	// Check for existing binder
	if openBinder, ok := c.openBinders[documentID]; ok {
		c.binderMutex.Unlock()
		return openBinder.Subscribe(userID, timeout)
	}
	openBinder, err := binder.New(
		documentID, c.store, c.config.BinderConfig, c.errorChan, c.log, c.stats,
	)
	if err != nil {
		c.binderMutex.Unlock()

		c.stats.Incr("curator.bind_existing.failed", 1)
		c.log.Errorf("Failed to bind to document %v: %v\n", documentID, err)
		return nil, err
	}
	c.openBinders[documentID] = openBinder
	c.binderMutex.Unlock()

	c.stats.Incr("curator.open_binders", 1)
	return openBinder.Subscribe(userID, timeout)
}

/*
ReadDocument - Locates or creates a Binder for an existing document and returns that Binder for
subscribing to with read only privileges. Returns an error if there was a problem locating the
document.
*/
func (c *impl) ReadDocument(
	userID, token, documentID string, timeout time.Duration,
) (binder.Portal, error) {
	c.log.Debugf("finding document %v, with userID %v token %v\n", documentID, userID, token)

	if c.auth.Authenticate(userID, token, documentID) < acl.ReadAccess {
		c.stats.Incr("curator.read.rejected_client", 1)
		return nil, fmt.Errorf(
			"failed to authorise read only join of document id: %v with token: %v\n",
			documentID, token,
		)
	}
	c.stats.Incr("curator.read.accepted_client", 1)

	c.binderMutex.Lock()

	// Check for existing binder
	if openBinder, ok := c.openBinders[documentID]; ok {
		c.binderMutex.Unlock()
		return openBinder.SubscribeReadOnly(userID, timeout)
	}
	openBinder, err := binder.New(
		documentID, c.store, c.config.BinderConfig, c.errorChan, c.log, c.stats,
	)
	if err != nil {
		c.binderMutex.Unlock()

		c.stats.Incr("curator.bind_existing.failed", 1)
		c.log.Errorf("Failed to bind to document %v: %v\n", documentID, err)
		return nil, err
	}
	c.openBinders[documentID] = openBinder
	c.binderMutex.Unlock()

	c.stats.Incr("curator.open_binders", 1)
	return openBinder.SubscribeReadOnly(userID, timeout)
}

/*
CreateDocument - Creates a fresh Binder for a new document, which is subsequently stored, returns an
error if either the document ID is already currently in use, or if there is a problem storing the
new document. May require authentication, if so a userID is supplied.
*/
func (c *impl) CreateDocument(
	userID, token string, doc store.Document, timeout time.Duration,
) (binder.Portal, error) {
	c.log.Debugf("Creating new document with userID %v token %v\n", userID, token)

	if c.auth.Authenticate(userID, token, "") < acl.CreateAccess {
		c.stats.Incr("curator.create.rejected_client", 1)
		return nil, fmt.Errorf("failed to gain permission to create with token: %v\n", token)
	}
	c.stats.Incr("curator.create.accepted_client", 1)

	// Always generate a fresh ID
	doc.ID = util.GenerateStampedUUID()

	if err := c.store.Create(doc); err != nil {
		c.stats.Incr("curator.create_new.failed", 1)
		c.log.Errorf("Failed to create new document: %v\n", err)
		return nil, err
	}
	openBinder, err := binder.New(
		doc.ID, c.store, c.config.BinderConfig, c.errorChan, c.log, c.stats,
	)
	if err != nil {
		c.stats.Incr("curator.bind_new.failed", 1)
		c.log.Errorf("Failed to bind to new document: %v\n", err)
		return nil, err
	}
	c.binderMutex.Lock()
	c.openBinders[doc.ID] = openBinder
	c.binderMutex.Unlock()
	c.stats.Incr("curator.open_binders", 1)

	return openBinder.Subscribe(userID, timeout)
}

//--------------------------------------------------------------------------------------------------
