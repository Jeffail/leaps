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
)

/*--------------------------------------------------------------------------------------------------
 */

/*
CuratorConfig - Holds configuration options for a curator.
*/
type CuratorConfig struct {
	StoreConfig  DocumentStoreConfig `json:"storage"`
	BinderConfig BinderConfig        `json:"binder"`
	LoggerConfig LoggerConfig        `json:"logger"`
}

/*
DefaultCuratorConfig - Returns a fully defined curator configuration with the default values for
each field.
*/
func DefaultCuratorConfig() CuratorConfig {
	return CuratorConfig{
		StoreConfig:  DefaultDocumentStoreConfig(),
		BinderConfig: DefaultBinderConfig(),
		LoggerConfig: DefaultLoggerConfig(),
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Curator - A structure designed to keep track of a live collection of Binders. Assists prospective
clients in locating their target Binders, and when necessary creates new Binders.
*/
type Curator struct {
	config      CuratorConfig
	store       DocumentStore
	logger      *LeapsLogger
	openBinders map[string]*Binder
	errorChan   chan BinderError
	closeChan   chan bool
	closedChan  chan bool
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

	curator := Curator{
		config:      config,
		store:       store,
		logger:      CreateLogger(config.LoggerConfig),
		openBinders: make(map[string]*Binder),
		errorChan:   make(chan BinderError),
		closeChan:   make(chan bool),
		closedChan:  make(chan bool),
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
			c.log(LeapInfo, "received call to close, forwarding message to binders")
			for _, b := range c.openBinders {
				b.Close()
			}
			close(c.closedChan)
			return
		case err := <-c.errorChan:
			c.log(LeapError, fmt.Sprintf("Binder (%v) %v\n", err.ID, err.Err))
			if b, ok := c.openBinders[err.ID]; ok {
				b.Close()
				delete(c.openBinders, err.ID)
				c.log(LeapInfo, fmt.Sprintf("Binder (%v) was closed\n", err.ID))
			} else {
				c.log(LeapError, fmt.Sprintf("Binder (%v) was not located in map\n", err.ID))
			}
		}
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
FindDocument - Locates an existing, or creates a fresh Binder for an existing document and returns
that Binder for subscribing to. Returns an error if there was a problem locating the document.
*/
func (c *Curator) FindDocument(id string) (*BinderPortal, error) {
	if binder, ok := c.openBinders[id]; ok {
		return binder.Subscribe(), nil
	}

	binder, err := BindExisting(id, c.store, c.config.BinderConfig, c.errorChan, c.logger)
	if err != nil {
		return nil, err
	}

	c.openBinders[id] = binder

	return binder.Subscribe(), nil
}

/*
NewDocument - Creates a fresh Binder for a new document, which is subsequently stored, returns an
error if either the document ID is already currently in use, or if there is a problem storing the
new document.
*/
func (c *Curator) NewDocument(doc *Document) (*BinderPortal, error) {
	// Always generate a fresh ID
	doc.ID = GenerateID(doc.Title, doc.Description)

	if err := ValidateDocument(doc); err != nil {
		return nil, err
	}

	binder, err := BindNew(doc, c.store, c.config.BinderConfig, c.errorChan, c.logger)
	if err != nil {
		return nil, err
	}

	c.openBinders[doc.ID] = binder

	return binder.Subscribe(), nil
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
