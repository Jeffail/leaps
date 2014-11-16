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

	"github.com/jeffail/leaps/util"
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
	ID     string
	config BinderConfig
	model  Model
	block  DocumentStore
	log    *util.Logger
	stats  *util.Stats

	// Clients
	clients       map[string]chan<- interface{}
	SubscribeChan chan BinderSubscribeBundle

	// Control channels
	jobChan    chan BinderRequest
	exitChan   chan string
	errorChan  chan<- BinderError
	closedChan chan struct{}
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
	log *util.Logger,
	stats *util.Stats,
) (*Binder, error) {

	binder := Binder{
		ID:            id,
		config:        config,
		model:         CreateTextModel(config.ModelConfig), //TODO: Generic
		block:         block,
		log:           log.NewModule("[binder]"),
		stats:         stats,
		clients:       make(map[string]chan<- interface{}),
		SubscribeChan: make(chan BinderSubscribeBundle),
		jobChan:       make(chan BinderRequest),
		exitChan:      make(chan string),
		errorChan:     errorChan,
		closedChan:    make(chan struct{}),
	}
	binder.log.Debugln("Bound to existing document, attempting flush")

	if _, err := binder.flush(); err != nil {
		stats.Incr("binder.bind_existing.error", 1)
		return nil, err
	}
	go binder.loop()

	stats.Incr("binder.bind_existing.success", 1)
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
	log *util.Logger,
	stats *util.Stats,
) (*Binder, error) {

	if err := block.Create(document.ID, document); err != nil {
		return nil, err
	}
	binder := Binder{
		ID:            document.ID,
		log:           log.NewModule("[binder]"),
		stats:         stats,
		model:         CreateTextModel(config.ModelConfig), // TODO: Make generic
		block:         block,
		config:        config,
		clients:       make(map[string]chan<- interface{}),
		SubscribeChan: make(chan BinderSubscribeBundle),
		jobChan:       make(chan BinderRequest),
		exitChan:      make(chan string),
		errorChan:     errorChan,
		closedChan:    make(chan struct{}),
	}
	binder.log.Debugln("Bound to new document, attempting flush")

	if _, err := binder.flush(); err != nil {
		stats.Incr("binder.bind_new.error", 1)
		return nil, err
	}
	go binder.loop()

	stats.Incr("binder.bind_new.success", 1)
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
	<-b.closedChan
}

/*--------------------------------------------------------------------------------------------------
 */

/*
processSubscriber - Processes a prospective client wishing to subscribe to this binder. This
involves flushing the model in order to obtain a clean version of the document, if this fails
we return false to flag the binder loop that we should shut down.
*/
func (b *Binder) processSubscriber(request BinderSubscribeBundle) error {
	if _, ok := b.clients[request.Token]; ok {
		b.stats.Incr("binder.rejected_client", 1)
		b.log.Warnf("Rejected client due to duplicate token: %v\n", request.Token)
		return errors.New("rejected due to duplicate token")
	}

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
		RequestSndChan:   b.jobChan,
		ExitChan:         b.exitChan,
	}:
		b.stats.Incr("binder.subscribed_clients", 1)
		b.log.Debugf("Subscribed new client %v\n", request.Token)
		b.clients[request.Token] = sndChan
	case <-time.After(time.Duration(b.config.ClientKickPeriod) * time.Millisecond):
		// We're not bothered if you suck, you just don't get enrolled. Deal with it.
		b.stats.Incr("binder.rejected_client", 1)
		b.log.Infof("Rejected client request %v\n", request.Token)
		return nil
	}

	return nil
}

/*
sendClientError - Sends an error to a channel, the channel should be non-blocking (buffered by at
least one and kept empty). In the event where the channel is blocked a log entry is made.
*/
func (b *Binder) sendClientError(errChan chan<- error, err error) {
	select {
	case errChan <- err:
	default:
		b.log.Errorln("Send client error was blocked")
		b.stats.Incr("binder.send_client_error.blocked", 1)
	}
}

/*
processJob - Processes a clients sent transforms, also returns the conditioned transforms to be
broadcast to other listening clients.
*/
func (b *Binder) processJob(request BinderRequest) {
	if request.Transform == nil {
		b.sendClientError(request.ErrorChan, errors.New("received job without a transform"))
		b.stats.Incr("binder.process_job.skipped", 1)
		return
	}
	dispatch := request.Transform

	// When the version chan is nil we assume the transform is a status update.
	if request.VersionChan != nil {
		var err error
		var version int

		b.log.Debugf("Received transform: %v\n", request.Transform)
		dispatch, version, err = b.model.PushTransform(request.Transform)

		if err != nil {
			b.stats.Incr("binder.process_job.error", 1)
			b.sendClientError(request.ErrorChan, err)
			return
		}
		select {
		case request.VersionChan <- version:
		default:
			b.log.Errorln("Send client version was blocked")
			b.stats.Incr("binder.send_client_version.blocked", 1)
		}
		b.stats.Incr("binder.process_job.success", 1)
	} else {
		b.sendClientError(request.ErrorChan, nil)
		b.stats.Incr("binder.process_update.success", 1)
	}

	clientKickPeriod := (time.Duration(b.config.ClientKickPeriod) * time.Millisecond)

	for key, c := range b.clients {
		// Skip sends for clients with matching tokens
		if key == request.Token {
			continue
		}
		select {
		case c <- dispatch:
		case <-time.After(clientKickPeriod):
			/* The client may have stopped listening, or is just being slow.
			 * Either way, we have a strict policy here of no time wasters.
			 */
			b.stats.Decr("binder.subscribed_clients", 1)
			b.stats.Incr("binder.clients_kicked", 1)
			delete(b.clients, key)
			close(c)
		}
	}
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
		b.stats.Incr("binder.block_fetch.error", 1)
		return nil, errStore
	}
	changed, errFlush = b.model.FlushTransforms(&doc.Content, b.config.RetentionPeriod)
	if changed {
		errStore = b.block.Store(b.ID, doc)
	}
	if errStore != nil || errFlush != nil {
		b.stats.Incr("binder.flush.error", 1)
		return nil, fmt.Errorf("%v, %v", errFlush, errStore)
	}
	if changed {
		b.stats.Incr("binder.flush.success", 1)
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
					b.log.Errorf("Flush error: %v, shutting down\n", err)
					running = false
				} else {
					flushTimer.Reset(flushPeriod)
					closeTimer.Reset(closePeriod)
				}
			} else {
				b.log.Infoln("Subscribe channel closed, shutting down")
				running = false
			}
		case job, open := <-b.jobChan:
			if running && open {
				b.processJob(job)
				closeTimer.Reset(closePeriod)
			} else {
				b.log.Infoln("Jobs channel closed, shutting down")
				running = false
			}
		case exitKey, open := <-b.exitChan:
			if running && open {
				b.log.Debugf("Received exit request from: %v\n", exitKey)
				delete(b.clients, exitKey)
				b.stats.Decr("binder.subscribed_clients", 1)
			} else {
				b.log.Infoln("Exit channel closed, shutting down")
				running = false
			}
		case <-flushTimer.C:
			if _, err := b.flush(); err != nil {
				b.log.Errorf("Flush error: %v, shutting down\n", err)
				b.errorChan <- BinderError{ID: b.ID, Err: err}
				running = false
			}
			flushTimer.Reset(flushPeriod)
		case <-closeTimer.C:
			if 0 == len(b.clients) {
				b.log.Infoln("Binder inactive, requesting shutdown")
				// Send graceful close request
				b.errorChan <- BinderError{ID: b.ID, Err: nil}
			}
			closeTimer.Reset(closePeriod)
		}
		if !running {
			flushTimer.Stop()
			closeTimer.Stop()

			b.stats.Incr("binder.closing", 1)
			b.log.Infoln("Closing, shutting down client channels")
			oldClients := b.clients
			b.clients = make(map[string]chan<- interface{})
			for _, client := range oldClients {
				close(client)
			}
			b.log.Infof("Attempting final flush of %v\n", b.ID)
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
