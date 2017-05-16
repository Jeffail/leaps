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

package binder

import (
	"fmt"
	"sync"
	"time"

	"github.com/jeffail/leaps/lib/store"
	"github.com/jeffail/leaps/lib/text"
	"github.com/jeffail/leaps/lib/util"
	"github.com/jeffail/util/log"
	"github.com/jeffail/util/metrics"
)

//------------------------------------------------------------------------------

// Config - Holds configuration options for a binder.
type Config struct {
	FlushPeriod           int64               `json:"flush_period_ms" yaml:"flush_period_ms"`
	RetentionPeriod       int64               `json:"retention_period_s" yaml:"retention_period_s"`
	ClientKickPeriod      int64               `json:"kick_period_ms" yaml:"kick_period_ms"`
	CloseInactivityPeriod int64               `json:"close_inactivity_period_s" yaml:"close_inactivity_period_s"`
	OTBufferConfig        text.OTBufferConfig `json:"transform_buffer" yaml:"transform_buffer"`
}

// NewConfig - Returns a fully defined Binder configuration with the default
// values for each field.
func NewConfig() Config {
	return Config{
		FlushPeriod:           500,
		RetentionPeriod:       60,
		ClientKickPeriod:      200,
		CloseInactivityPeriod: 300,
		OTBufferConfig:        text.NewOTBufferConfig(),
	}
}

//------------------------------------------------------------------------------

// impl - A Type implementation that contains a single document and acts as a
// broker between multiple readers, writers and the storage strategy.
type impl struct {
	id       string
	config   Config
	otBuffer *text.OTBuffer
	block    store.Type

	log   log.Modular
	stats metrics.Aggregator

	// Clients
	clients       []*binderClient
	subscribeChan chan subscribeRequest

	clientMux sync.Mutex

	// Control channels
	transformChan    chan transformSubmission
	messageChan      chan messageSubmission
	usersRequestChan chan usersRequest
	exitChan         chan *binderClient
	kickChan         chan kickRequest
	errorChan        chan<- Error
	closedChan       chan struct{}
}

// New - Creates a binder targeting an existing document determined via an ID.
func New(
	id string,
	block store.Type,
	config Config,
	errorChan chan<- Error,
	log log.Modular,
	stats metrics.Aggregator,
) (Type, error) {
	binder := impl{
		id:               id,
		config:           config,
		otBuffer:         text.NewOTBuffer(config.OTBufferConfig),
		block:            block,
		log:              log.NewModule(":binder"),
		stats:            stats,
		clients:          make([]*binderClient, 0),
		subscribeChan:    make(chan subscribeRequest),
		transformChan:    make(chan transformSubmission),
		messageChan:      make(chan messageSubmission),
		usersRequestChan: make(chan usersRequest),
		exitChan:         make(chan *binderClient),
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

//------------------------------------------------------------------------------

// ID - Return the binder ID.
func (b *impl) ID() string {
	return b.id
}

type usersRequest struct {
	responseChan chan<- []string
}

// GetUsers - Get a list of user id's connected to this binder.
func (b *impl) GetUsers(timeout time.Duration) ([]string, error) {
	resChan := make(chan []string)
	b.usersRequestChan <- usersRequest{resChan}

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

// KickUser - Signals the binder to remove a particular user. Currently doesn't
// confirm removal, this ought to be a blocking call until the removal is
// validated.
func (b *impl) KickUser(userID string, timeout time.Duration) error {
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

type subscribeRequest struct {
	userID     string
	portalChan chan<- *portalImpl
	errChan    chan<- error
}

// Subscribe - Returns a Portal, which represents a contract between a client
// and the binder.
func (b *impl) Subscribe(userID string, timeout time.Duration) (Portal, error) {
	portalChan, errChan := make(chan *portalImpl, 1), make(chan error, 1)
	bundle := subscribeRequest{
		portalChan: portalChan,
		errChan:    errChan,
		userID:     userID,
	}

	select {
	case b.subscribeChan <- bundle:
	case <-time.After(timeout):
		return nil, ErrTimeout
	}

	select {
	case portal := <-portalChan:
		return portal, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(timeout):
	}
	return nil, ErrTimeout
}

// SubscribeReadOnly - Returns a read-only Portal, which represents a contract
// between a client and the binder.
func (b *impl) SubscribeReadOnly(userID string, timeout time.Duration) (Portal, error) {
	portalChan, errChan := make(chan *portalImpl, 1), make(chan error, 1)
	bundle := subscribeRequest{
		portalChan: portalChan,
		errChan:    errChan,
		userID:     userID,
	}

	select {
	case b.subscribeChan <- bundle:
	case <-time.After(timeout):
		return nil, ErrTimeout
	}

	select {
	case portal := <-portalChan:
		portal.transformSndChan = nil
		return portal, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(timeout):
	}
	return nil, ErrTimeout
}

// Close - Close the binder, before closing the client channels the binder will
// flush changes and store the document.
func (b *impl) Close() {
	close(b.subscribeChan)
	<-b.closedChan
}

//------------------------------------------------------------------------------

// processSubscriber - Processes a prospective client wishing to subscribe to
// this binder. This involves flushing the OTBuffer in order to obtain a clean
// version of the document, if this fails we return false to flag the binder
// loop that we should shut down.
func (b *impl) processSubscriber(request subscribeRequest) error {
	transformSndChan := make(chan text.OTransform, 1)
	updateSndChan := make(chan ClientUpdate, 1)

	var err error
	var doc store.Document

	// We need to read the full document here anyway, so might as well flush.
	doc, err = b.flush()
	if err != nil {
		select {
		case request.errChan <- err:
		default:
		}
		return err
	}

	client := binderClient{
		userID:        request.userID,
		sessionID:     util.GenerateStampedUUID(),
		transformChan: transformSndChan,
		updateChan:    updateSndChan,
	}
	portal := portalImpl{
		client:           &client,
		version:          b.otBuffer.GetVersion(),
		document:         doc,
		transformRcvChan: transformSndChan,
		updateRcvChan:    updateSndChan,
		transformSndChan: b.transformChan,
		messageSndChan:   b.messageChan,
		exitChan:         b.exitChan,
	}
	select {
	case request.portalChan <- &portal:
		b.stats.Incr("binder.subscribed_clients", 1)
		b.log.Debugf("Subscribed new client %v\n", request.userID)
		b.clients = append(b.clients, &client)
	case <-time.After(time.Duration(b.config.ClientKickPeriod) * time.Millisecond):
		/* We're not bothered if you suck, you just don't get enrolled, and this isn't
		 * considered an error. Deal with it.
		 */
		b.stats.Incr("binder.rejected_client", 1)
		b.log.Infof("Rejected client request %v\n", request.userID)
	}
	return nil
}

// removeClient - Closes a client and removes it from the binder, uses a mutex
// and is therefore safe to call asynchronously.
func (b *impl) removeClient(client *binderClient) {
	b.clientMux.Lock()
	defer b.clientMux.Unlock()

	for i := 0; i < len(b.clients); i++ {
		c := b.clients[i]
		if c == client {
			b.stats.Decr("binder.subscribed_clients", 1)
			b.clients = append(b.clients[:i], b.clients[i+1:]...)
			i--

			close(c.transformChan)
			close(c.updateChan)
		}
	}
}

// sendClientError - Sends an error to a channel, the channel should be
// non-blocking (buffered by at least one and kept empty). In the event where
// the channel is blocked a log entry is made.
func (b *impl) sendClientError(errChan chan<- error, err error) {
	select {
	case errChan <- err:
	default:
		b.log.Errorln("Send client error was blocked")
		b.stats.Incr("binder.send_client_error.blocked", 1)
	}
}

// processUsersRequest - Processes a request for the list of connected clients.
func (b *impl) processUsersRequest(request usersRequest) {
	var clients []string
	for _, client := range b.clients {
		clients = append(clients, client.userID)
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

// processTransform - Processes a clients transform submission, and broadcasts
// the transform out to other clients.
func (b *impl) processTransform(request transformSubmission) {
	var dispatch text.OTransform
	var err error
	var version int

	b.log.Debugf("Received transform: %q\n", fmt.Sprintf("%v", request.transform))
	dispatch, version, err = b.otBuffer.PushTransform(request.transform)

	if err != nil {
		b.stats.Incr("binder.process_job.error", 1)
		b.sendClientError(request.errorChan, err)
		return
	}
	select {
	case request.versionChan <- version:
	default:
		b.log.Errorln("Send client version was blocked")
		b.stats.Incr("binder.send_client_version.blocked", 1)
	}
	b.stats.Incr("binder.process_job.success", 1)

	clientKickPeriod := (time.Duration(b.config.ClientKickPeriod) * time.Millisecond)

	wg := sync.WaitGroup{}

	clients := b.clients
	wg.Add(len(clients) - 1)

	for i := 0; i < len(clients); i++ {
		client := clients[i]

		// Skip sends for client from which the message came
		if client == request.client {
			continue
		}
		go func(c *binderClient) {
			select {
			case c.transformChan <- dispatch:
			case <-time.After(clientKickPeriod):
				/* The client may have stopped listening, or is just being slow.
				 * Either way, we have a strict policy here of no time wasters.
				 */
				b.stats.Incr("binder.clients_kicked", 1)
				b.log.Debugf("Kicking client for user: (%v) for blocked transform send\n", c.userID)
				b.removeClient(c)
			}
			wg.Done()
		}(client)
	}

	wg.Wait()
}

// processMessage - Sends a clients message out to other clients.
func (b *impl) processMessage(request messageSubmission) {
	clientKickPeriod := (time.Duration(b.config.ClientKickPeriod) * time.Millisecond)

	b.log.Tracef("Received message: %v %v\n", *request.client, request.message)

	clientUpdate := ClientUpdate{
		ClientInfo: ClientInfo{
			UserID:    request.client.userID,
			SessionID: request.client.sessionID,
		},
		Message: request.message,
	}

	wg := sync.WaitGroup{}

	clients := b.clients
	wg.Add(len(clients) - 1)

	for i := 0; i < len(clients); i++ {
		client := clients[i]

		// Skip sends for client from which the message came
		if client == request.client {
			continue
		}
		go func(c *binderClient) {
			select {
			case c.updateChan <- clientUpdate:
				b.stats.Incr("binder.sent_message", 1)
			case <-time.After(clientKickPeriod):
				/* The client may have stopped listening, or is just being slow.
				 * Either way, we have a strict policy here of no time wasters.
				 */
				b.stats.Incr("binder.clients_kicked", 1)
				b.log.Debugf("Kicking client for user: (%v) for blocked transform send\n", c.userID)
				b.removeClient(c)
			}
			wg.Done()
		}(client)
	}

	wg.Wait()
}

// flush - Obtain latest document content, flush current changes to document,
// and store the updated version.
func (b *impl) flush() (store.Document, error) {
	var (
		errStore, errFlush error
		changed            bool
		doc                store.Document
	)
	doc, errStore = b.block.Read(b.id)
	if errStore != nil {
		b.stats.Incr("binder.block_fetch.error", 1)
		return doc, errStore
	}
	changed, errFlush = b.otBuffer.FlushTransforms(&doc.Content, b.config.RetentionPeriod)
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

//------------------------------------------------------------------------------

/*
loop - The internal loop that performs the broker duties of the binder. Which
includes the following:

- Enrolling new clients by dispatching a fresh copy of the document
- Receiving messages and transforms from clients
- Dispatching received messages and transforms to all other enrolled clients
- Intermittently flushing changes to the document storage solution
- Intermittently checking for active clients, and shutting down when unused
*/
func (b *impl) loop() {
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
					b.errorChan <- Error{ID: b.id, Err: err}
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
				clients := b.clients
				kicked := 0
				for i := 0; i < len(clients); i++ {
					c := clients[i]
					if c.userID == kickRequest.userID {
						b.removeClient(clients[i])
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
				b.log.Debugf("Received exit request for: %v\n", client.userID)
				b.removeClient(client)
			} else {
				b.log.Infoln("Exit channel closed, shutting down")
				running = false
			}
		case <-flushTimer.C:
			if b.otBuffer.IsDirty() {
				if _, err := b.flush(); err != nil {
					b.log.Errorf("Flush error: %v, shutting down\n", err)
					b.errorChan <- Error{ID: b.id, Err: err}
					running = false
				}
			}
			flushTimer.Reset(flushPeriod)
		case <-closeTimer.C:
			if 0 == len(b.clients) {
				b.log.Infoln("Binder inactive, requesting shutdown")
				// Send graceful close request
				b.errorChan <- Error{ID: b.id, Err: nil}
			}
			closeTimer.Reset(closePeriod)
		}
		if !running {
			flushTimer.Stop()
			closeTimer.Stop()

			b.stats.Incr("binder.closing", 1)
			b.log.Infoln("Closing, shutting down client channels")
			oldClients := b.clients
			b.clients = make([]*binderClient, 0)
			for _, client := range oldClients {
				close(client.transformChan)
				close(client.updateChan)
			}
			b.log.Infof("Attempting final flush of %v\n", b.id)
			if b.otBuffer.IsDirty() {
				if _, err := b.flush(); err != nil {
					b.errorChan <- Error{ID: b.id, Err: err}
				}
			}
			close(b.closedChan)
			return
		}
	}
}

//------------------------------------------------------------------------------
