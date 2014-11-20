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

package lib

import (
	"fmt"
	"sync"

	"github.com/jeffail/leaps/util"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
CuratorConfig - Holds configuration options for a curator.
*/
type CuratorConfig struct {
	StoreConfig         DocumentStoreConfig      `json:"storage"`
	BinderConfig        BinderConfig             `json:"binder"`
	AuthenticatorConfig TokenAuthenticatorConfig `json:"authenticator"`
}

/*
DefaultCuratorConfig - Returns a fully defined curator configuration with the default values for
each field.
*/
func DefaultCuratorConfig() CuratorConfig {
	return CuratorConfig{
		StoreConfig:         DefaultDocumentStoreConfig(),
		BinderConfig:        DefaultBinderConfig(),
		AuthenticatorConfig: DefaultTokenAuthenticatorConfig(),
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Curator - A structure designed to keep track of a live collection of Binders. Assists prospective
clients in locating their target Binders, and when necessary creates new Binders.
*/
type Curator struct {
	config        CuratorConfig
	store         DocumentStore
	log           *util.Logger
	stats         *util.Stats
	authenticator TokenAuthenticator

	// Binders
	openBinders map[string]*Binder
	binderMutex sync.RWMutex

	// Control channels
	errorChan  chan BinderError
	closeChan  chan struct{}
	closedChan chan struct{}
}

/*
NewCurator - Creates and returns a fresh curator, and launches its internal loop for monitoring
Binder errors.
*/
func NewCurator(config CuratorConfig, log *util.Logger, stats *util.Stats) (*Curator, error) {
	store, err := DocumentStoreFactory(config.StoreConfig)
	if err != nil {
		return nil, err
	}
	auth, err := TokenAuthenticatorFactory(config.AuthenticatorConfig, log)
	if err != nil {
		return nil, err
	}

	log.Debugln("Creating curator")
	curator := Curator{
		config:        config,
		store:         store,
		log:           log.NewModule("[curator]"),
		stats:         stats,
		authenticator: auth,
		openBinders:   make(map[string]*Binder),
		errorChan:     make(chan BinderError, 10),
		closeChan:     make(chan struct{}),
		closedChan:    make(chan struct{}),
	}
	go curator.loop()

	return &curator, nil
}

/*
Close - Shut the curator and all subsequent binders down. This call blocks until the shut down is
finished, and you must ensure that this library cannot be accessed after closing.
*/
func (c *Curator) Close() {
	c.log.Debugln("Close called")
	c.closeChan <- struct{}{}
	<-c.closedChan
}

/*
loop - The main loop of the curator, this loop simply listens to the error and close channels.
*/
func (c *Curator) loop() {
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
			}
			c.binderMutex.Unlock()
			close(c.closedChan)
			return
		}
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
FindDocument - Locates or creates a Binder for an existing document and returns that Binder for
subscribing to. Returns an error if there was a problem locating the document.
*/
func (c *Curator) FindDocument(token, id string) (BinderPortal, error) {
	c.log.Debugf("finding document %v, with token %v\n", id, token)

	if !c.authenticator.AuthoriseJoin(token, id) {
		c.stats.Incr("curator.find.rejected_client", 1)
		return BinderPortal{}, fmt.Errorf("failed to authorise join of document id: %v with token: %v\n", id, token)
	}
	c.stats.Incr("curator.find.accepted_client", 1)

	c.binderMutex.Lock()
	defer c.binderMutex.Unlock()

	// Check for existing binder
	if binder, ok := c.openBinders[id]; ok {
		return binder.Subscribe(token), nil
	}
	binder, err := NewBinder(id, c.store, c.config.BinderConfig, c.errorChan, c.log, c.stats)
	if err != nil {
		c.stats.Incr("curator.bind_existing.failed", 1)
		c.log.Errorf("Failed to bind to document %v: %v\n", id, err)
		return BinderPortal{}, err
	}
	c.openBinders[id] = binder

	return binder.Subscribe(token), nil
}

/*
NewDocument - Creates a fresh Binder for a new document, which is subsequently stored, returns an
error if either the document ID is already currently in use, or if there is a problem storing the
new document. May require authentication, if so a userID is supplied.
*/
func (c *Curator) NewDocument(token string, userID string, doc *Document) (BinderPortal, error) {
	c.log.Debugf("Creating new document with token %v\n", token)

	if !c.authenticator.AuthoriseCreate(token, userID) {
		c.stats.Incr("curator.create.rejected_client", 1)
		return BinderPortal{}, fmt.Errorf("failed to gain permission to create with token: %v\n", token)
	}
	c.stats.Incr("curator.create.accepted_client", 1)

	// Always generate a fresh ID
	doc.ID = GenerateID()

	if err := c.store.Create(doc.ID, doc); err != nil {
		c.stats.Incr("curator.create_new.failed", 1)
		c.log.Errorf("Failed to create new document: %v\n", err)
		return BinderPortal{}, err
	}
	binder, err := NewBinder(doc.ID, c.store, c.config.BinderConfig, c.errorChan, c.log, c.stats)
	if err != nil {
		c.stats.Incr("curator.bind_new.failed", 1)
		c.log.Errorf("Failed to bind to new document: %v\n", err)
		return BinderPortal{}, err
	}
	c.binderMutex.Lock()
	c.openBinders[doc.ID] = binder
	c.binderMutex.Unlock()

	return binder.Subscribe(token), nil
}

/*--------------------------------------------------------------------------------------------------
 */
