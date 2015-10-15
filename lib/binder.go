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
	"time"

	"github.com/jeffail/leaps/lib/store"
	"github.com/jeffail/leaps/lib/util"
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
	block  store.Store
	log    *log.Logger
	stats  *log.Stats

	// Clients
	clients       []*BinderClient
	subscribeChan chan BinderSubscribeBundle

	// Control channels
	transformChan    chan TransformSubmission
	messageChan      chan MessageSubmission
	usersRequestChan chan usersRequestObj
	exitChan         chan *BinderClient
	kickChan         chan kickRequest
	errorChan        chan<- BinderError
	closedChan       chan struct{}
}

/*
NewBinder - Creates a binder targeting an existing document determined via an ID. Must provide a
store.Store to acquire the document and apply future updates to.
*/
func NewBinder(
	id string,
	block store.Store,
	config BinderConfig,
	errorChan chan<- BinderError,
	log *log.Logger,
	stats *log.Stats,
) (*Binder, error) {

	binder := Binder{
		ID:               id,
		config:           config,
		model:            CreateTextModel(config.ModelConfig),
		block:            block,
		log:              log.NewModule(":binder"),
		stats:            stats,
		clients:          make([]*BinderClient, 0),
		subscribeChan:    make(chan BinderSubscribeBundle),
		transformChan:    make(chan TransformSubmission),
		messageChan:      make(chan MessageSubmission),
		usersRequestChan: make(chan usersRequestObj),
		exitChan:         make(chan *BinderClient),
		kickChan:         make(chan kickRequest),
		errorChan:        errorChan,
		closedChan:       make(chan struct{}),
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
BinderClient - A struct containing information about a connected client and channels used by the
binder to push transforms and user updates out.
*/
type BinderClient struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`

	transformChan chan<- OTransform
	messageChan   chan<- MessageSubmission
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

type usersRequestObj struct {
	responseChan chan<- []string
}

/*
GetUsers - Get a list of user id's connected to this binder.
*/
func (b *Binder) GetUsers(timeout time.Duration) ([]string, error) {
	resChan := make(chan []string)
	b.usersRequestChan <- usersRequestObj{resChan}

	select {
	case result := <-resChan:
		return result, nil
	case <-time.After(timeout):
	}
	return []string{}, ErrTimeout
}

type kickRequest struct {
	userID string
	result chan error
}

/*
KickUser - Signals the binder to remove a particular user. Currently doesn't confirm removal, this
ought to be a blocking call until the removal is validated.
*/
func (b *Binder) KickUser(userID string, timeout time.Duration) error {
	result := make(chan error)
	timer := time.After(timeout)
	select {
	case b.kickChan <- kickRequest{userID: userID, result: result}:
	case <-timer:
		return ErrTimeout
	}
	select {
	case err := <-result:
		return err
	case <-timer:
		return ErrTimeout
	}
}

/*
Subscribe - Returns a BinderPortal, which represents a contract between a client and the binder. If
the subscription was unsuccessful the BinderPortal will contain an error.
*/
func (b *Binder) Subscribe(userID string) BinderPortal {
	retChan := make(chan BinderPortal, 1)
	bundle := BinderSubscribeBundle{
		PortalRcvChan: retChan,
		UserID:        userID,
	}
	b.subscribeChan <- bundle

	return <-retChan
}

/*
SubscribeReadOnly - Returns a BinderPortal, which represents a contract between a client and the
binder. If the subscription was unsuccessful the BinderPortal will contain an error. This is a read
only version of a BinderPortal and means transforms will be received but cannot be submitted.
*/
func (b *Binder) SubscribeReadOnly(userID string) BinderPortal {
	retChan := make(chan BinderPortal, 1)
	bundle := BinderSubscribeBundle{
		PortalRcvChan: retChan,
		UserID:        userID,
	}
	b.subscribeChan <- bundle

	portal := <-retChan
	portal.TransformSndChan = nil

	return portal
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
	transformSndChan := make(chan OTransform, 1)
	messageSndChan := make(chan MessageSubmission, 1)

	// We need to read the full document here anyway, so might as well flush.
	doc, err := b.flush()
	if err != nil {
		return err
	}
	client := BinderClient{
		UserID:        request.UserID,
		SessionID:     util.GenerateStampedUUID(),
		transformChan: transformSndChan,
		messageChan:   messageSndChan,
	}
	portal := BinderPortal{
		Client:           &client,
		Version:          b.model.GetVersion(),
		Document:         doc,
		Error:            nil,
		TransformRcvChan: transformSndChan,
		MessageRcvChan:   messageSndChan,
		TransformSndChan: b.transformChan,
		MessageSndChan:   b.messageChan,
		ExitChan:         b.exitChan,
	}
	select {
	case request.PortalRcvChan <- portal:
		b.stats.Incr("binder.subscribed_clients", 1)
		b.log.Debugf("Subscribed new client %v\n", request.UserID)
		b.clients = append(b.clients, &client)
	case <-time.After(time.Duration(b.config.ClientKickPeriod) * time.Millisecond):
		/* We're not bothered if you suck, you just don't get enrolled, and this isn't
		 * considered an error. Deal with it.
		 */
		b.stats.Incr("binder.rejected_client", 1)
		b.log.Infof("Rejected client request %v\n", request.UserID)
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
processUsersRequest - Processes a request for the list of connected clients.
*/
func (b *Binder) processUsersRequest(request usersRequestObj) {
	var clients []string
	for _, client := range b.clients {
		clients = append(clients, client.UserID)
	}
	select {
	case request.responseChan <- clients:
	case <-time.After(time.Duration(b.config.ClientKickPeriod) * time.Millisecond):
		/* If the receive channel is blocked then we move on, we have more important things to
		 * deal with.
		 */
		b.stats.Incr("binder.rejected_users_request", 1)
		b.log.Warnln("Rejected users request")
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

	for i, c := range b.clients {
		// Skip sends for client from which the message came
		if c == request.Client {
			continue
		}
		select {
		case c.transformChan <- dispatch:
		case <-time.After(clientKickPeriod):
			/* The client may have stopped listening, or is just being slow.
			 * Either way, we have a strict policy here of no time wasters.
			 */
			b.stats.Decr("binder.subscribed_clients", 1)
			b.stats.Incr("binder.clients_kicked", 1)

			b.log.Debugf("Kicking client for user: (%v) for blocked transform send\n", c.UserID)

			b.clients = append(b.clients[:i], b.clients[i+1:]...)
			close(c.transformChan)
			close(c.messageChan)
		}
	}
}

/*
processMessage - Sends a clients message out to other clients.
*/
func (b *Binder) processMessage(request MessageSubmission) {
	clientKickPeriod := (time.Duration(b.config.ClientKickPeriod) * time.Millisecond)

	for i, c := range b.clients {
		// Skip sends for client from which the message came
		if c == request.Client {
			continue
		}
		select {
		case c.messageChan <- request:
		case <-time.After(clientKickPeriod):
			/* The client may have stopped listening, or is just being slow.
			 * Either way, we have a strict policy here of no time wasters.
			 */
			b.stats.Decr("binder.subscribed_clients", 1)
			b.stats.Incr("binder.clients_kicked", 1)

			b.log.Debugf("Kicking client for user: (%v) for blocked transform send\n", c.UserID)

			b.clients = append(b.clients[:i], b.clients[i+1:]...)
			close(c.transformChan)
			close(c.messageChan)
		}
	}
}

/*
flush - Obtain latest document content, flush current changes to document, and store the updated
version.
*/
func (b *Binder) flush() (store.Document, error) {
	var (
		errStore, errFlush error
		changed            bool
		doc                store.Document
	)
	doc, errStore = b.block.Read(b.ID)
	if errStore != nil {
		b.stats.Incr("binder.block_fetch.error", 1)
		return doc, errStore
	}
	changed, errFlush = b.model.FlushTransforms(&doc.Content, b.config.RetentionPeriod)
	if changed {
		errStore = b.block.Update(doc)
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
		case usersRequest, open := <-b.usersRequestChan:
			if running && open {
				b.processUsersRequest(usersRequest)
			} else {
				b.log.Infoln("Users request channel closed, shutting down")
				running = false
			}
		case kickRequest, open := <-b.kickChan:
			if running && open {
				b.log.Debugf("Received kick request for: %v\n", kickRequest.userID)

				// TODO: Refactor and improve kick API
				kicked := 0
				for i, c := range b.clients {
					if c.UserID == kickRequest.userID {
						b.stats.Decr("binder.subscribed_clients", 1)
						b.clients = append(b.clients[:i], b.clients[i+1:]...)
						close(c.transformChan)
						close(c.messageChan)
						kicked++
					}
				}
				if kicked > 0 {
					close(kickRequest.result)
				} else {
					kickRequest.result <- fmt.Errorf("No such userID: %s", kickRequest.userID)
				}
			} else {
				b.log.Infoln("Exit channel closed, shutting down")
				running = false
				close(kickRequest.result)
			}
		case client, open := <-b.exitChan:
			if running && open {
				b.log.Debugf("Received exit request for: %v\n", client.UserID)
				for i, c := range b.clients {
					if c == client {
						b.stats.Decr("binder.subscribed_clients", 1)
						b.clients = append(b.clients[:i], b.clients[i+1:]...)
						close(c.transformChan)
						close(c.messageChan)
					}
				}
			} else {
				b.log.Infoln("Exit channel closed, shutting down")
				running = false
			}
		case <-flushTimer.C:
			if b.model.IsDirty() {
				if _, err := b.flush(); err != nil {
					b.log.Errorf("Flush error: %v, shutting down\n", err)
					b.errorChan <- BinderError{ID: b.ID, Err: err}
					running = false
				}
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
			b.clients = make([]*BinderClient, 0)
			for _, client := range oldClients {
				close(client.transformChan)
				close(client.messageChan)
			}
			b.log.Infof("Attempting final flush of %v\n", b.ID)
			if b.model.IsDirty() {
				if _, err := b.flush(); err != nil {
					b.errorChan <- BinderError{ID: b.ID, Err: err}
				}
			}
			close(b.closedChan)
			return
		}
	}
}

/*--------------------------------------------------------------------------------------------------
 */
