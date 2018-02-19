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

	"github.com/Jeffail/leaps/lib/audit"
	"github.com/Jeffail/leaps/lib/store"
	"github.com/Jeffail/leaps/lib/text"
	"github.com/Jeffail/leaps/lib/util/service/log"
	"github.com/Jeffail/leaps/lib/util/service/metrics"
)

//------------------------------------------------------------------------------

// Config - Holds configuration options for a binder.
type Config struct {
	FlushPeriodMS           int64               `json:"flush_period_ms" yaml:"flush_period_ms"`
	RetentionPeriodS        int64               `json:"retention_period_s" yaml:"retention_period_s"`
	ClientKickPeriodMS      int64               `json:"kick_period_ms" yaml:"kick_period_ms"`
	CloseInactivityPeriodMS int64               `json:"close_inactivity_period_ms" yaml:"close_inactivity_period_ms"`
	OTBufferConfig          text.OTBufferConfig `json:"transform_buffer" yaml:"transform_buffer"`
}

// NewConfig - Returns a fully defined Binder configuration with the default
// values for each field.
func NewConfig() Config {
	return Config{
		FlushPeriodMS:           500,
		RetentionPeriodS:        60,
		ClientKickPeriodMS:      200,
		CloseInactivityPeriodMS: 300000,
		OTBufferConfig:          text.NewOTBufferConfig(),
	}
}

//------------------------------------------------------------------------------

// impl - A Type implementation that contains a single document and acts as a
// broker between multiple readers, writers and the storage strategy.
type impl struct {
	id       string
	config   Config
	otBuffer TransformSink
	block    store.Type
	auditor  audit.Auditor

	log   log.Modular
	stats metrics.Type

	// Clients
	clients       []*binderClient
	subscribeChan chan subscribeRequest

	clientMux sync.Mutex

	// Control channels
	transformChan chan transformSubmission
	metadataChan  chan metadataSubmission
	exitChan      chan *binderClient
	errorChan     chan<- Error
	closedChan    chan struct{}
}

// New - Creates a binder targeting an existing document determined via an ID.
func New(
	id string,
	block store.Type,
	config Config,
	errorChan chan<- Error,
	log log.Modular,
	stats metrics.Type,
	auditor audit.Auditor,
) (Type, error) {
	binder := impl{
		id:            id,
		config:        config,
		block:         block,
		auditor:       auditor,
		log:           log.NewModule(":binder"),
		stats:         stats,
		clients:       make([]*binderClient, 0),
		subscribeChan: make(chan subscribeRequest),
		transformChan: make(chan transformSubmission),
		metadataChan:  make(chan metadataSubmission),
		exitChan:      make(chan *binderClient),
		errorChan:     errorChan,
		closedChan:    make(chan struct{}),
	}
	binder.log.Debugln("Attempting to read and bind to new document")

	doc, err := block.Read(id)
	if err != nil {
		binder.stats.Incr("binder.block_fetch.error", 1)
		return nil, err
	}

	binder.otBuffer = text.NewOTBuffer(doc.Content, config.OTBufferConfig)
	go binder.loop()

	stats.Incr("binder.new.success", 1)
	return &binder, nil
}

//------------------------------------------------------------------------------

// ID - Return the binder ID.
func (b *impl) ID() string {
	return b.id
}

type subscribeRequest struct {
	metadata   interface{}
	portalChan chan<- *portalImpl
	errChan    chan<- error
}

// Subscribe - Returns a Portal, which represents a contract between a client
// and the binder. Metadata can be added to the portal in order to identify the
// clients submissions.
func (b *impl) Subscribe(metadata interface{}, timeout time.Duration) (Portal, error) {
	portalChan, errChan := make(chan *portalImpl, 1), make(chan error, 1)
	bundle := subscribeRequest{
		metadata:   metadata,
		portalChan: portalChan,
		errChan:    errChan,
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
// between a client and the binder. Metadata can be added to the portal in order
// to identify the clients submissions.
func (b *impl) SubscribeReadOnly(metadata interface{}, timeout time.Duration) (Portal, error) {
	portalChan, errChan := make(chan *portalImpl, 1), make(chan error, 1)
	bundle := subscribeRequest{
		metadata:   metadata,
		portalChan: portalChan,
		errChan:    errChan,
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
	metadataSndChan := make(chan ClientMetadata, 1)

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
		metadata:      request.metadata,
		transformChan: transformSndChan,
		metadataChan:  metadataSndChan,
	}
	portal := portalImpl{
		client:           &client,
		version:          b.otBuffer.GetVersion(),
		document:         doc,
		transformRcvChan: transformSndChan,
		metadataRcvChan:  metadataSndChan,
		transformSndChan: b.transformChan,
		metadataSndChan:  b.metadataChan,
		exitChan:         b.exitChan,
	}
	select {
	case request.portalChan <- &portal:
		b.stats.Incr("binder.subscribed_clients", 1)
		b.log.Debugf("Subscribed new client %v\n", request.metadata)
		b.clients = append(b.clients, &client)
	case <-time.After(time.Duration(b.config.ClientKickPeriodMS) * time.Millisecond):
		/* We're not bothered if you suck, you just don't get enrolled, and this isn't
		 * considered an error. Deal with it.
		 */
		b.stats.Incr("binder.rejected_client", 1)
		b.log.Infof("Rejected client request %v\n", request.metadata)
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
			close(c.metadataChan)
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
	// If we have an auditor then send it our transforms.
	if b.auditor != nil {
		b.auditor.OnTransform(dispatch)
	}
	select {
	case request.versionChan <- version:
	default:
		b.log.Errorln("Send client version was blocked")
		b.stats.Incr("binder.send_client_version.blocked", 1)
	}
	b.stats.Incr("binder.process_job.success", 1)

	clientKickPeriod := (time.Duration(b.config.ClientKickPeriodMS) * time.Millisecond)

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
				b.log.Debugf("Kicking client for user: (%v) for blocked transform send\n", c.metadata)
				b.removeClient(c)
			}
			wg.Done()
		}(client)
	}

	wg.Wait()
}

// processMetadata - Sends a clients metadata submission out to other clients.
func (b *impl) processMetadata(request metadataSubmission) {
	clientKickPeriod := (time.Duration(b.config.ClientKickPeriodMS) * time.Millisecond)

	b.log.Tracef("Received metadata: %v %v\n", *request.client, request.metadata)

	metadata := ClientMetadata{
		Client:   request.client.metadata,
		Metadata: request.metadata,
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
			case c.metadataChan <- metadata:
				b.stats.Incr("binder.sent_metadata", 1)
			case <-time.After(clientKickPeriod):
				/* The client may have stopped listening, or is just being slow.
				 * Either way, we have a strict policy here of no time wasters.
				 */
				b.stats.Incr("binder.clients_kicked", 1)
				b.log.Debugf("Kicking client for user: (%v) for blocked transform send\n", c.metadata)
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
	changed, errFlush = b.otBuffer.FlushTransforms(&doc.Content, b.config.RetentionPeriodS)
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
- Receiving metadata and transforms from clients
- Dispatching received metadata and transforms to all other enrolled clients
- Intermittently flushing changes to the document storage solution
- Intermittently checking for active clients, and shutting down when unused
*/
func (b *impl) loop() {
	flushPeriod := (time.Duration(b.config.FlushPeriodMS) * time.Millisecond)
	closePeriod := (time.Duration(b.config.CloseInactivityPeriodMS) * time.Millisecond)

	flushTimer := time.NewTimer(flushPeriod)
	closeTimer := time.NewTimer(closePeriod)
	for {
		running := true
		select {
		case clientBundle, open := <-b.subscribeChan:
			if open {
				if err := b.processSubscriber(clientBundle); err != nil {
					b.errorChan <- Error{ID: b.id, Err: err}
					b.log.Errorf("Flush error: %v, shutting down\n", err)
					running = false
				} else {
					flushTimer.Reset(flushPeriod)
				}
			} else {
				b.log.Infoln("Subscribe channel closed, shutting down")
				running = false
			}
		case tform, open := <-b.transformChan:
			if open {
				b.processTransform(tform)
			} else {
				b.log.Infoln("Transforms channel closed, shutting down")
				running = false
			}
		case metadata, open := <-b.metadataChan:
			if open {
				b.processMetadata(metadata)
			} else {
				b.log.Infoln("Metadata channel closed, shutting down")
				running = false
			}
		case client, open := <-b.exitChan:
			if open {
				b.log.Debugf("Received exit request for: %v\n", client.metadata)
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
			} else {
				closeTimer.Reset(closePeriod)
			}
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
				close(client.metadataChan)
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
