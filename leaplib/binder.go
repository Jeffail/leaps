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
BinderConfig - Holds configuration options for a binder.
*/
type BinderConfig struct {
	FlushPeriod      int64 `json:"flush_period_ms"`
	RetentionPeriod  int64 `json:"retention_period_s"`
	ClientKickPeriod int64 `json:"kick_period_ms"`
}

/*
DefaultBinderConfig - Returns a fully defined Binder configuration with the default values for each
field.
*/
func DefaultBinderConfig() BinderConfig {
	return BinderConfig{
		FlushPeriod:      500,
		RetentionPeriod:  60,
		ClientKickPeriod: 5,
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

	binder.log(LeapInfo, "Bound to existing document, attempting flush")

	if _, err := binder.flush(); err != nil {
		binder.logger.IncrementStat("binder.bind_existing.error")
		return nil, err
	}

	go binder.loop()

	binder.logger.IncrementStat("binder.bind_existing.success")
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

	binder.log(LeapInfo, "Bound to new document, attempting flush")

	if _, err := binder.flush(); err != nil {
		binder.logger.IncrementStat("binder.bind_new.error")
		return nil, err
	}

	go binder.loop()

	binder.logger.IncrementStat("binder.bind_new.success")
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
			b.logger.IncrementStat("binder.process_job.skipped")
		default:
		}
		return
	}

	newOTs := make([]interface{}, 1)
	var err error
	var version int

	b.log(LeapDebug, fmt.Sprintf("Received transform: %v", request.Transform))
	newOTs[0], version, err = b.model.PushTransform(request.Transform)

	if err != nil {
		b.logger.IncrementStat("binder.process_job.error")
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

	b.logger.IncrementStat("binder.process_job.success")

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
		b.logger.IncrementStat("binder.clients_kicked")
		b.log(LeapInfo, fmt.Sprintf("Kicked %v inactive clients", deadClients))
	}

	b.clients = newClients
}

/*
flush - Obtain latest document content, flush current changes to document, and store the updated
version.
*/
func (b *Binder) flush() (*Document, error) {
	var errStore, errFlush error
	var changed bool
	var doc *Document

	doc, errStore = b.block.Fetch(b.ID)
	if errStore != nil {
		b.logger.IncrementStat("binder.block_fetch.error")
		return nil, errStore
	}

	changed, errFlush = b.model.FlushTransforms(&doc.Content, b.config.RetentionPeriod)
	if changed {
		errStore = b.block.Store(b.ID, doc)
	}

	if errStore != nil || errFlush != nil {
		b.logger.IncrementStat("binder.flush.error")
		return nil, fmt.Errorf("%v, %v", errFlush, errStore)
	}
	if changed {
		b.logger.IncrementStat("binder.flush.success")
	}
	return doc, nil
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
					b.log(LeapError, fmt.Sprintf("Flush error: %v, shutting down", err))
					running = false
				} else {
					/* Channel is buffered by one element to be non-blocking, any blocked send will
					 * lead to a rejected client, bad client!
					 */
					sndChan := make(chan []interface{}, 1)
					b.clients = append(b.clients, sndChan)
					b.logger.IncrementStat("binder.subscribed_client")
					b.log(LeapInfo, "Subscribing new client")

					client <- &BinderPortal{
						Version:          b.model.GetVersion(),
						Document:         doc,
						Error:            nil,
						TransformRcvChan: sndChan,
						RequestSndChan:   b.jobs,
					}
					flushTime = time.After(flushPeriod)
				}
			} else {
				b.log(LeapInfo, "Subscribe channel closed, shutting down")
				running = false
			}
		case job, open := <-b.jobs:
			if running && open {
				b.processJob(job)
			} else {
				b.log(LeapInfo, "Jobs channel closed, shutting down")
				running = false
			}
		case <-flushTime:
			if _, err := b.flush(); err != nil {
				b.log(LeapError, fmt.Sprintf("Flush error: %v, shutting down", err))
				b.errorChan <- BinderError{ID: b.ID, Err: err}
				running = false
			}
			flushTime = time.After(flushPeriod)
		}
		if !running {
			b.logger.IncrementStat("binder.closing")
			b.log(LeapInfo, "Closing, shutting down client channels")
			oldClients := b.clients
			b.clients = [](chan<- []interface{}){}
			for _, client := range oldClients {
				close(client)
			}
			b.log(LeapInfo, fmt.Sprintf("Attempting final flush of %v", b.ID))
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
