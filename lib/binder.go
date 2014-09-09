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
	FlushPeriod           int64       `json:"flush_period_ms"`
	RetentionPeriod       int64       `json:"retention_period_s"`
	ClientKickPeriod      int64       `json:"kick_period_ms"`
	CloseInactivityPeriod int64       `json:"close_inactivity_period_s"`
	ModelConfig           ModelConfig `json:"transform_model"`
}

/*
DefaultBinderConfig - Returns a fully defined Binder configuration with the default values for each
field.
*/
func DefaultBinderConfig() BinderConfig {
	return BinderConfig{
		FlushPeriod:           500,
		RetentionPeriod:       60,
		ClientKickPeriod:      5,
		CloseInactivityPeriod: 300,
		ModelConfig:           DefaultModelConfig(),
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
	SubscribeChan chan BinderSubscribeBundle

	logger     *LeapsLogger
	model      Model
	block      DocumentStore
	config     BinderConfig
	clients    []BinderClient
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
		SubscribeChan: make(chan BinderSubscribeBundle),
		logger:        logger,
		model:         CreateTextModel(config.ModelConfig), //TODO: Generic
		block:         block,
		config:        config,
		clients:       []BinderClient{},
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

	if err := block.Create(document.ID, document); err != nil {
		return nil, err
	}

	binder := Binder{
		ID:            document.ID,
		SubscribeChan: make(chan BinderSubscribeBundle),
		logger:        logger,
		model:         CreateTextModel(config.ModelConfig), // TODO: Make generic
		block:         block,
		config:        config,
		clients:       []BinderClient{},
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
transforms. Accepts a string as a token for identifying the user, if left empty the token is
generated.

Multiple clients can connect with the same token, however, these users will be treated
as if the same client, therefore not receiving each others messages, cursor positions being
overwritten and being kicked in unison.
*/
func (b *Binder) Subscribe(token string) *BinderPortal {
	if len(token) == 0 {
		token = GenerateID("client salt")
	}

	retChan := make(chan *BinderPortal, 1)
	bundle := BinderSubscribeBundle{
		PortalRcvChan: retChan,
		Token:         token,
	}
	b.SubscribeChan <- bundle

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
processSubscriber - Processes a prospective client wishing to subscribe to this binder. This
involves flushing the model in order to obtain a clean version of the document, if this fails
we return false to flag the binder loop that we should shut down.
*/
func (b *Binder) processSubscriber(request BinderSubscribeBundle) error {
	/* Channel is buffered by one element to be non-blocking, any blocked send will
	 * lead to a rejected client, bad client!
	 */
	sndChan := make(chan interface{}, 1)

	// We need to read the full document here anyway, so might as well flush.
	doc, err := b.flush()
	if err != nil {
		return err
	}

	select {
	case request.PortalRcvChan <- &BinderPortal{
		Token:            request.Token,
		Version:          b.model.GetVersion(),
		Document:         doc,
		Error:            nil,
		TransformRcvChan: sndChan,
		RequestSndChan:   b.jobs,
	}:
		b.clients = append(b.clients, BinderClient{
			Token:            request.Token,
			TransformSndChan: sndChan,
		})
	case <-time.After(time.Duration(b.config.ClientKickPeriod) * time.Millisecond):
		// We're not bothered if you suck, you just don't get enrolled. Deal with it.
		b.logger.IncrementStat("binder.rejected_client")
		b.log(LeapInfo, fmt.Sprintf("Rejected client request %v", request.Token))
		return nil
	}

	b.logger.IncrementStat("binder.subscribed_client")
	b.log(LeapInfo, fmt.Sprintf("Subscribed new client %v", request.Token))

	return nil
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

	dispatch := request.Transform

	if request.VersionChan != nil {
		var err error
		var version int

		b.log(LeapDebug, fmt.Sprintf("Received transform: %v", request.Transform))
		dispatch, version, err = b.model.PushTransform(request.Transform)

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
	} else {
		request.ErrorChan <- nil
		b.logger.IncrementStat("binder.process_update.success")
	}

	clientKickPeriod := (time.Duration(b.config.ClientKickPeriod) * time.Millisecond)

	for i, c := range b.clients {
		// Skip sends for clients with matching tokens
		if c.Token == request.Token {
			continue
		}
		select {
		case c.TransformSndChan <- dispatch:
		case <-time.After(clientKickPeriod):
			/* The client may have stopped listening, or is just being slow.
			 * Either way, we have a strict policy here of no time wasters.
			 */
			close(b.clients[i].TransformSndChan)
			b.clients[i].TransformSndChan = nil
		}
		// Currently also sends to client that submitted it, oops, or no oops?
	}

	deadClients := 0
	newClients := []BinderClient{}
	for _, c := range b.clients {
		if c.TransformSndChan != nil {
			newClients = append(newClients, c)
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

/*
checkActive - Scans any remaining clients to purge the inactive, returns true if there are remaining
active clients connected.
*/
func (b *Binder) checkActive() bool {
	if len(b.clients) > 0 {
		clientKickPeriod := (time.Duration(b.config.ClientKickPeriod) * time.Millisecond)

		for _, c := range b.clients {
			select {
			case c.TransformSndChan <- nil:
				return true
			case <-time.After(clientKickPeriod):
			}
		}
	}
	return false
}

/*--------------------------------------------------------------------------------------------------
 */

/*
loop - The internal loop that performs the broker duties of the binder. The period of intermittent
flushes must be specified.
*/
func (b *Binder) loop() {
	flushPeriod := (time.Duration(b.config.FlushPeriod) * time.Millisecond)
	closePeriod := (time.Duration(b.config.CloseInactivityPeriod) * time.Second)

	flushTimer := time.NewTimer(flushPeriod)
	closeTimer := time.NewTimer(closePeriod)
	for {
		running := true
		select {
		case clientBundle, open := <-b.SubscribeChan:
			if running && open {
				if err := b.processSubscriber(clientBundle); err != nil {
					b.errorChan <- BinderError{ID: b.ID, Err: err}
					b.log(LeapError, fmt.Sprintf("Flush error: %v, shutting down", err))
					running = false
				} else {
					flushTimer.Reset(flushPeriod)
					closeTimer.Reset(closePeriod)
				}
			} else {
				b.log(LeapInfo, "Subscribe channel closed, shutting down")
				running = false
			}
		case job, open := <-b.jobs:
			if running && open {
				b.processJob(job)
				closeTimer.Reset(closePeriod)
			} else {
				b.log(LeapInfo, "Jobs channel closed, shutting down")
				running = false
			}
		case <-flushTimer.C:
			if _, err := b.flush(); err != nil {
				b.log(LeapError, fmt.Sprintf("Flush error: %v, shutting down", err))
				b.errorChan <- BinderError{ID: b.ID, Err: err}
				running = false
			}
			flushTimer.Reset(flushPeriod)
		case <-closeTimer.C:
			active := b.checkActive()

			if !active {
				b.log(LeapInfo, "Binder inactive, requesting shutdown")
				// Send graceful close request
				b.errorChan <- BinderError{ID: b.ID, Err: nil}
			}
			closeTimer.Reset(closePeriod)
		}
		if !running {
			flushTimer.Stop()
			closeTimer.Stop()

			b.logger.IncrementStat("binder.closing")
			b.log(LeapInfo, "Closing, shutting down client channels")
			oldClients := b.clients
			b.clients = []BinderClient{}
			for _, client := range oldClients {
				close(client.TransformSndChan)
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
