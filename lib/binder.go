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

	"github.com/jeffail/util/log"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
BinderConfig - Holds configuration options for a binder.
*/
type BinderConfig struct {
	FlushPeriod           int64       `json:"flush_period_ms" yaml:"flush_period_ms"`
	RetentionPeriod       int64       `json:"retention_period_s" yaml:"retention_period_s"`
	ClientKickPeriod      int64       `json:"kick_period_ms" yaml:"kick_period_ms"`
	CloseInactivityPeriod int64       `json:"close_inactivity_period_s" yaml:"close_inactivity_period_s"`
	ModelConfig           ModelConfig `json:"transform_model" yaml:"transform_model"`
}

/*
DefaultBinderConfig - Returns a fully defined Binder configuration with the default values for each
field.
*/
func DefaultBinderConfig() BinderConfig {
	return BinderConfig{
		FlushPeriod:           500,
		RetentionPeriod:       60,
		ClientKickPeriod:      200,
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
	log    *log.Logger
	stats  *log.Stats

	// Clients
	clients       map[string]BinderClient
	subscribeChan chan BinderSubscribeBundle

	// Control channels
	transformChan chan TransformSubmission
	messageChan   chan MessageSubmission
	exitChan      chan string
	errorChan     chan<- BinderError
	closedChan    chan struct{}
}

/*
NewBinder - Creates a binder targeting an existing document determined via an ID. Must provide a
DocumentStore to acquire the document and apply future updates to.
*/
func NewBinder(
	id string,
	block DocumentStore,
	config BinderConfig,
	errorChan chan<- BinderError,
	log *log.Logger,
	stats *log.Stats,
) (*Binder, error) {

	binder := Binder{
		ID:            id,
		config:        config,
		model:         CreateTextModel(config.ModelConfig),
		block:         block,
		log:           log.NewModule("[binder]"),
		stats:         stats,
		clients:       make(map[string]BinderClient),
		subscribeChan: make(chan BinderSubscribeBundle),
		transformChan: make(chan TransformSubmission),
		messageChan:   make(chan MessageSubmission),
		exitChan:      make(chan string),
		errorChan:     errorChan,
		closedChan:    make(chan struct{}),
	}
	binder.log.Debugln("Bound to document, attempting flush")

	if _, err := binder.flush(); err != nil {
		stats.Incr("binder.new.error", 1)
		return nil, err
	}
	go binder.loop()

	stats.Incr("binder.new.success", 1)
	return &binder, nil
}

/*--------------------------------------------------------------------------------------------------
 */

/*
ClientMessage - A struct containing various updates to a clients' state and an optional message to
be distributed out to all other clients of a binder.
*/
type ClientMessage struct {
	Message  string `json:"message,omitempty" yaml:"message,omitempty"`
	Position *int64 `json:"position,omitempty" yaml:"position,omitempty"`
	Active   bool   `json:"active" yaml:"active"`
	Token    string `json:"user_id" yaml:"user_id"`
}

/*
BinderClient - A struct containing information about a connected client and channels used by the
binder to push transforms and user updates out.
*/
type BinderClient struct {
	Token         string
	TransformChan chan<- OTransform
	MessageChan   chan<- ClientMessage
}

/*
BinderError - A binder has encountered a problem and needs to close. In order for this to happen it
needs to inform its owner that it should be shut down. BinderError is a structure used to carry
our error message and our ID over an error channel. A BinderError with the Err set to nil can be
used as a graceful shutdown request.
*/
type BinderError struct {
	ID  string
	Err error
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Subscribe - Returns a BinderPortal, which represents a contract between a client and the binder. If
the subscription was unsuccessful the BinderPortal will contain an error.
*/
func (b *Binder) Subscribe(token string) BinderPortal {
	if len(token) == 0 {
		token = GenerateStampedUUID()
	}
	retChan := make(chan BinderPortal, 1)
	bundle := BinderSubscribeBundle{
		PortalRcvChan: retChan,
		Token:         token,
	}
	b.subscribeChan <- bundle

	return <-retChan
}

/*
Close - Close the binder, before closing the client channels the binder will flush changes and
store the document.
*/
func (b *Binder) Close() {
	close(b.subscribeChan)
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

	transformSndChan := make(chan OTransform, 1)
	messageSndChan := make(chan ClientMessage, 1)

	// We need to read the full document here anyway, so might as well flush.
	doc, err := b.flush()
	if err != nil {
		return err
	}
	select {
	case request.PortalRcvChan <- BinderPortal{
		Token:            request.Token,
		Version:          b.model.GetVersion(),
		Document:         doc,
		Error:            nil,
		TransformRcvChan: transformSndChan,
		MessageRcvChan:   messageSndChan,
		TransformSndChan: b.transformChan,
		MessageSndChan:   b.messageChan,
		ExitChan:         b.exitChan,
	}:
		b.stats.Incr("binder.subscribed_clients", 1)
		b.log.Debugf("Subscribed new client %v\n", request.Token)
		b.clients[request.Token] = BinderClient{
			Token:         request.Token,
			TransformChan: transformSndChan,
			MessageChan:   messageSndChan,
		}
	case <-time.After(time.Duration(b.config.ClientKickPeriod) * time.Millisecond):
		/* We're not bothered if you suck, you just don't get enrolled, and this isn't
		 * considered an error. Deal with it.
		 */
		b.stats.Incr("binder.rejected_client", 1)
		b.log.Infof("Rejected client request %v\n", request.Token)
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
processTransform - Processes a clients transform submission, and broadcasts the transform out to
other clients.
*/
func (b *Binder) processTransform(request TransformSubmission) {
	var dispatch OTransform
	var err error
	var version int

	b.log.Debugf("Received transform: %q\n", fmt.Sprintf("%v", request.Transform))
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

	clientKickPeriod := (time.Duration(b.config.ClientKickPeriod) * time.Millisecond)

	for key, c := range b.clients {
		// Skip sends for clients with matching tokens
		if key == request.Token {
			continue
		}
		select {
		case c.TransformChan <- dispatch:
		case <-time.After(clientKickPeriod):
			/* The client may have stopped listening, or is just being slow.
			 * Either way, we have a strict policy here of no time wasters.
			 */
			b.stats.Decr("binder.subscribed_clients", 1)
			b.stats.Incr("binder.clients_kicked", 1)

			b.log.Debugf("Kicking client (%v) for blocked transform send\n", key)

			delete(b.clients, key)
			close(c.TransformChan)
			close(c.MessageChan)
		}
	}
}

/*
processMessage - Sends a clients message out to other clients.
*/
func (b *Binder) processMessage(request MessageSubmission) {
	clientKickPeriod := (time.Duration(b.config.ClientKickPeriod) * time.Millisecond)

	for key, c := range b.clients {
		// Skip sends for clients with matching tokens
		if key == request.Token {
			continue
		}
		select {
		case c.MessageChan <- request.Message:
		case <-time.After(clientKickPeriod):
			/* The client may have stopped listening, or is just being slow.
			 * Either way, we have a strict policy here of no time wasters.
			 */
			b.stats.Decr("binder.subscribed_clients", 1)
			b.stats.Incr("binder.clients_kicked", 1)

			b.log.Debugf("Kicking client (%v) for blocked message send\n", key)

			delete(b.clients, key)
			close(c.TransformChan)
			close(c.MessageChan)
		}
	}
}

/*
flush - Obtain latest document content, flush current changes to document, and store the updated
version.
*/
func (b *Binder) flush() (Document, error) {
	var (
		errStore, errFlush error
		changed            bool
		doc                Document
	)
	doc, errStore = b.block.Fetch(b.ID)
	if errStore != nil {
		b.stats.Incr("binder.block_fetch.error", 1)
		return doc, errStore
	}
	changed, errFlush = b.model.FlushTransforms(&doc.Content, b.config.RetentionPeriod)
	if changed {
		errStore = b.block.Store(b.ID, doc)
	}
	if errStore != nil || errFlush != nil {
		b.stats.Incr("binder.flush.error", 1)
		return doc, fmt.Errorf("%v, %v", errFlush, errStore)
	}
	if changed {
		b.stats.Incr("binder.flush.success", 1)
	}
	return doc, nil
}

/*--------------------------------------------------------------------------------------------------
 */

/*
loop - The internal loop that performs the broker duties of the binder. Which includes the
following:

- Enrolling new clients by dispatching a fresh copy of the document
- Receiving messages and transforms from clients
- Dispatching received messages and transforms to all other enrolled clients
- Intermittently flushing changes to the document storage solution
- Intermittently checking for active clients, and shutting down when unused
*/
func (b *Binder) loop() {
	flushPeriod := (time.Duration(b.config.FlushPeriod) * time.Millisecond)
	closePeriod := (time.Duration(b.config.CloseInactivityPeriod) * time.Second)

	flushTimer := time.NewTimer(flushPeriod)
	closeTimer := time.NewTimer(closePeriod)
	for {
		running := true
		select {
		case clientBundle, open := <-b.subscribeChan:
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
		case tform, open := <-b.transformChan:
			if running && open {
				b.processTransform(tform)
				closeTimer.Reset(closePeriod)
			} else {
				b.log.Infoln("Transforms channel closed, shutting down")
				running = false
			}
		case message, open := <-b.messageChan:
			if running && open {
				b.processMessage(message)
				closeTimer.Reset(closePeriod)
			} else {
				b.log.Infoln("Messages channel closed, shutting down")
				running = false
			}
		case exitKey, open := <-b.exitChan:
			if running && open {
				b.log.Debugf("Received exit request from: %v\n", exitKey)
				if c, ok := b.clients[exitKey]; ok {
					delete(b.clients, exitKey)
					close(c.TransformChan)
					close(c.MessageChan)
				}
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
			b.clients = make(map[string]BinderClient)
			for _, client := range oldClients {
				close(client.TransformChan)
				close(client.MessageChan)
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
