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

package leaplib

import (
	"fmt"
	"sync"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
CuratorConfig - Holds configuration options for a curator.
*/
type CuratorConfig struct {
	StoreConfig         DocumentStoreConfig      `json:"storage"`
	BinderConfig        BinderConfig             `json:"binder"`
	LoggerConfig        LoggerConfig             `json:"logger"`
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
		LoggerConfig:        DefaultLoggerConfig(),
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
	logger        *LeapsLogger
	authenticator TokenAuthenticator
	openBinders   map[string]*Binder
	binderMutex   sync.RWMutex
	errorChan     chan BinderError
	closeChan     chan bool
	closedChan    chan bool
}

/*
CreateNewCurator - Creates and returns a fresh curator, and launches its internal loop for
monitoring Binder errors.
*/
func CreateNewCurator(config CuratorConfig) (*Curator, error) {
	store, err := DocumentStoreFactory(config.StoreConfig)
	if err != nil {
		return nil, err
	}
	auth, err := TokenAuthenticatorFactory(config.AuthenticatorConfig)
	if err != nil {
		return nil, err
	}

	curator := Curator{
		config:        config,
		store:         store,
		logger:        CreateLogger(config.LoggerConfig),
		authenticator: auth,
		openBinders:   make(map[string]*Binder),
		errorChan:     make(chan BinderError, 10),
		closeChan:     make(chan bool),
		closedChan:    make(chan bool),
	}

	go curator.loop()

	return &curator, nil
}

/*
Close - Shut the curator down, you must ensure that this library cannot be accessed after closing.
*/
func (c *Curator) Close() {
	c.closeChan <- true
	<-c.closedChan
}

/*
log - Helper function for logging events, only actually logs when verbose logging is configured.
*/
func (c *Curator) log(level int, message string) {
	c.logger.Log(level, "curator", message)
}

/*
loop - The main loop of the curator, this loop simply listens to the error and close channels.
*/
func (c *Curator) loop() {
	for {
		select {
		case <-c.closeChan:
			c.log(LeapInfo, "Received call to close, forwarding message to binders")
			c.binderMutex.Lock()
			for _, b := range c.openBinders {
				b.Close()
			}
			c.binderMutex.Unlock()
			close(c.closedChan)
			return
		case err := <-c.errorChan:
			if err.Err != nil {
				c.logger.IncrementStat("curator.binder_chan.error")
				c.log(LeapError, fmt.Sprintf("Binder (%v) %v\n", err.ID, err.Err))
			} else {
				c.log(LeapInfo, fmt.Sprintf("Binder (%v) has requested shutdown\n", err.ID))
			}
			c.binderMutex.Lock()
			if b, ok := c.openBinders[err.ID]; ok {
				b.Close()
				delete(c.openBinders, err.ID)
				c.log(LeapInfo, fmt.Sprintf("Binder (%v) was closed\n", err.ID))
				c.logger.IncrementStat("curator.binder_shutdown.success")
			} else {
				c.log(LeapError, fmt.Sprintf("Binder (%v) was not located in map\n", err.ID))
				c.logger.IncrementStat("curator.binder_shutdown.error")
			}
			c.binderMutex.Unlock()
		}
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
FindDocument - Locates an existing, or creates a fresh Binder for an existing document and returns
that Binder for subscribing to. Returns an error if there was a problem locating the document.
*/
func (c *Curator) FindDocument(token string, id string) (*BinderPortal, error) {
	if !c.authenticator.AuthoriseJoin(token, id) {
		return nil, fmt.Errorf("failed to authorise join of document id: %v with token: %v", id, token)
	}

	c.binderMutex.Lock()
	defer c.binderMutex.Unlock()

	if binder, ok := c.openBinders[id]; ok {
		c.logger.IncrementStat("curator.subscribed_client")
		return binder.Subscribe(token), nil
	}

	binder, err := BindExisting(id, c.store, c.config.BinderConfig, c.errorChan, c.logger)
	if err != nil {
		return nil, err
	}

	c.openBinders[id] = binder

	c.logger.IncrementStat("curator.subscribed_client")
	return binder.Subscribe(token), nil
}

/*
NewDocument - Creates a fresh Binder for a new document, which is subsequently stored, returns an
error if either the document ID is already currently in use, or if there is a problem storing the
new document.
*/
func (c *Curator) NewDocument(token string, doc *Document) (*BinderPortal, error) {
	if !c.authenticator.AuthoriseCreate(token) {
		return nil, fmt.Errorf("failed to gain permission to create with token: %v", token)
	}

	// Always generate a fresh ID
	doc.ID = GenerateID(fmt.Sprintf("%v%v", doc.Title, doc.Description))

	if err := ValidateDocument(doc); err != nil {
		c.logger.IncrementStat("curator.validate_new_document.error")
		return nil, err
	}
	c.logger.IncrementStat("curator.validate_new_document.success")

	binder, err := BindNew(doc, c.store, c.config.BinderConfig, c.errorChan, c.logger)
	if err != nil {
		return nil, err
	}

	c.binderMutex.Lock()
	c.openBinders[doc.ID] = binder
	c.binderMutex.Unlock()

	c.logger.IncrementStat("curator.subscribed_client")
	return binder.Subscribe(token), nil
}

/*
GetLogger - The curator generates a LeapsLogger that all other leaps components should write to.
GetLogger returns a reference to the logger for components not generated by the curator itself.
*/
func (c *Curator) GetLogger() *LeapsLogger {
	return c.logger
}

/*--------------------------------------------------------------------------------------------------
 */
