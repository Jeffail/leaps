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
	"errors"
	"fmt"
	"time"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
TransformConfig - Holds configuration options for managing individual transforms stored under a
binder.
*/
type TransformConfig struct {
	RetentionPeriod int64 `json:"retention_period"`
}

/*
BinderConfig - Holds configuration options for a binder.
*/
type BinderConfig struct {
	Transform        TransformConfig `json:"transform"`
	FlushPeriod      int64           `json:"flush_period_ms"`
	ClientKickPeriod int64           `json:"kick_period_ms"`
}

/*
DefaultBinderConfig - Returns a fully defined Binder configuration with the default values for each
field.
*/
func DefaultBinderConfig() BinderConfig {
	return BinderConfig{
		Transform: TransformConfig{
			RetentionPeriod: 60,
		},
		FlushPeriod:      50,
		ClientKickPeriod: 10,
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Binder - Contains a single document and acts as a broker between multiple readers, writers and the
storage strategy.
*/
type Binder struct {
	ID            string
	SubscribeChan chan (chan<- *BinderPortal)

	logger     *LeapsLogger
	model      Model
	block      DocumentStore
	config     BinderConfig
	clients    [](chan<- []interface{})
	jobs       chan BinderRequest
	errorChan  chan<- BinderError
	closedChan chan bool
}

/*
BindExisting - Creates a binder targeting a specific, existing document. Takes the id of the
document along with the DocumentStore to acquire and store the document with.
*/
func BindExisting(
	id string,
	block DocumentStore,
	config BinderConfig,
	errorChan chan<- BinderError,
	logger *LeapsLogger,
) (*Binder, error) {

	binder := Binder{
		ID:            id,
		SubscribeChan: make(chan (chan<- *BinderPortal)),
		logger:        logger,
		model:         CreateTextModel(id), //TODO: Generic
		block:         block,
		config:        config,
		clients:       [](chan<- []interface{}){},
		jobs:          make(chan BinderRequest),
		errorChan:     errorChan,
		closedChan:    make(chan bool),
	}

	binder.log(LeapInfo, "bound to existing document, attempting flush")

	if _, err := binder.flush(); err != nil {
		return nil, err
	}

	go binder.loop()

	return &binder, nil
}

/*
BindNew - Creates a binder around a new document. Requires a DocumentStore to store the document
with. Returns the binder, the ID of the new document, and a potential error.
*/
func BindNew(
	document *Document,
	block DocumentStore,
	config BinderConfig,
	errorChan chan<- BinderError,
	logger *LeapsLogger,
) (*Binder, error) {

	if err := block.Store(document.ID, document); err != nil {
		return nil, err
	}

	binder := Binder{
		ID:            document.ID,
		SubscribeChan: make(chan (chan<- *BinderPortal)),
		logger:        logger,
		model:         CreateTextModel(document.ID), // TODO: Make generic
		block:         block,
		config:        config,
		clients:       [](chan<- []interface{}){},
		jobs:          make(chan BinderRequest),
		errorChan:     errorChan,
		closedChan:    make(chan bool),
	}

	binder.log(LeapInfo, "bound to new document, attempting flush")

	if _, err := binder.flush(); err != nil {
		return nil, err
	}

	go binder.loop()

	return &binder, nil
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Subscribe - Returns a BinderPortal struct that allows a client to bootstrap and sync with the binder
with the document content, current unapplied changes and channels for sending and receiving
transforms.
*/
func (b *Binder) Subscribe() *BinderPortal {
	retChan := make(chan *BinderPortal, 1)
	b.SubscribeChan <- retChan
	return <-retChan
}

/*
Close - Close the binder, before closing the client channels the binder will flush changes and
store the document.
*/
func (b *Binder) Close() {
	close(b.SubscribeChan)
	close(b.jobs)
	<-b.closedChan
}

/*--------------------------------------------------------------------------------------------------
 */

/*
log - Helper function for logging events, only actually logs when verbose logging is configured.
*/
func (b *Binder) log(level int, message string) {
	b.logger.Log(level, "binder", fmt.Sprintf("(%v) %v", b.ID, message))
}

/*
processJob - Processes a clients sent transforms, also returns the conditioned transforms to be
broadcast to other listening clients.
*/
func (b *Binder) processJob(request BinderRequest) {
	if request.Transform == nil {
		select {
		case request.ErrorChan <- errors.New("received job without a transform"):
		default:
		}
		return
	}

	newOTs := make([]interface{}, 1)
	var err error
	var version int

	newOTs[0], version, err = b.model.PushTransform(request.Transform)

	if err != nil {
		select {
		case request.ErrorChan <- err:
		default:
		}
		return
	}

	select {
	case request.VersionChan <- version:
	default:
	}

	clientKickPeriod := (time.Duration(b.config.ClientKickPeriod) * time.Millisecond)

	for i, clientChan := range b.clients {
		select {
		case clientChan <- newOTs:
		case <-time.After(clientKickPeriod):
			/* The client may have stopped listening, or is just being slow.
			 * Either way, we have a strict policy here of no time wasters.
			 */
			close(b.clients[i])
			b.clients[i] = nil
		}
		// Currently also sends to client that submitted it, oops, or no oops?
	}

	deadClients := 0
	newClients := [](chan<- []interface{}){}
	for _, clientChan := range b.clients {
		if clientChan != nil {
			newClients = append(newClients, clientChan)
		} else {
			deadClients++
		}
	}

	if deadClients > 0 {
		b.log(LeapInfo, fmt.Sprintf("Kicked %v inactive clients", deadClients))
	}

	b.clients = newClients
}

/*
flush - Obtain latest document content, flush current changes to document, and store the updated
version.
*/
func (b *Binder) flush() (*Document, error) {
	doc, err := b.block.Fetch(b.ID)
	if err == nil {
		retention := time.Duration(b.config.Transform.RetentionPeriod) * time.Second
		if err = b.model.FlushTransforms(&doc.Content, retention); err != nil {
			return nil, err
		}
		b.block.Store(b.ID, doc)
		return doc, nil
	}
	return nil, err
}

/*--------------------------------------------------------------------------------------------------
 */

/*
loop - The internal loop that performs the broker duties of the binder. The period of intermittent
flushes must be specified.
*/
func (b *Binder) loop() {
	flushPeriod := (time.Duration(b.config.FlushPeriod) * time.Millisecond)

	flushTime := time.After(flushPeriod)
	for {
		running := true
		select {
		case client, open := <-b.SubscribeChan:
			if running && open {
				// We need to read the full document here anyway, so might as well flush.
				doc, err := b.flush()
				if err != nil {
					b.errorChan <- BinderError{ID: b.ID, Err: err}
					b.log(LeapError, fmt.Sprintf("flush error: %v, shutting down", err))
					running = false
				}
				/* Channel is buffered by one element to be non-blocking, any blocked send will
				 * lead to a rejected client, bad client!
				 */
				sndChan := make(chan []interface{}, 1)
				b.clients = append(b.clients, sndChan)
				b.log(LeapInfo, "subscribing new client")

				client <- &BinderPortal{
					Version:          b.model.GetVersion(),
					Document:         doc,
					Error:            nil,
					TransformRcvChan: sndChan,
					RequestSndChan:   b.jobs,
				}
				flushTime = time.After(flushPeriod)
			} else {
				b.log(LeapInfo, "subscribe channel closed, shutting down")
				running = false
			}
		case job, open := <-b.jobs:
			if running && open {
				b.processJob(job)
			} else {
				b.log(LeapInfo, "jobs channel closed, shutting down")
				running = false
			}
		case <-flushTime:
			if _, err := b.flush(); err != nil {
				b.log(LeapError, fmt.Sprintf("flush error: %v, shutting down", err))
				b.errorChan <- BinderError{ID: b.ID, Err: err}
				running = false
			}
			flushTime = time.After(flushPeriod)
		}
		if !running {
			b.log(LeapInfo, "closing, shutting down client channels")
			oldClients := b.clients[:]
			b.clients = [](chan<- []interface{}){}
			for _, client := range oldClients {
				close(client)
			}
			b.log(LeapInfo, fmt.Sprintf("attempting final flush of %v", b.ID))
			if _, err := b.flush(); err != nil {
				b.errorChan <- BinderError{ID: b.ID, Err: err}
			}
			close(b.closedChan)
			return
		}
	}
}

/*--------------------------------------------------------------------------------------------------
 */
