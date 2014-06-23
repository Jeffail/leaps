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
	"log"
	"os"
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
	LogVerbose       bool            `json:"verbose_logging"`
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
		LogVerbose:       false,
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
BinderError - A binder has encountered a problem and needs to close. In order for this to happen it
needs to inform its owner that it should be shut down. BinderError is a structure used to carry
our error message and our ID over an error channel.
*/
type BinderError struct {
	ID  string
	Err error
}

/*
BinderRequest - A container used to communicate with a binder, it holds a transform to be
submitted to the document model. Two channels are used for return values from the request.
VersionChan is used to send back the actual version of the transform submitted. ErrorChan is used to
send errors that occur. Both channels must be non-blocking, so a buffer of 1 is recommended.
*/
type BinderRequest struct {
	Transform   *OTransform
	VersionChan chan<- int
	ErrorChan   chan<- error
}

/*
BinderPortal - A container that holds all data necessary to begin an open portal with the binder,
allowing fresh transforms to be submitted and returned as they come.
*/
type BinderPortal struct {
	Transforms       []*OTransform
	Document         *Document
	Version          int
	Error            error
	TransformRcvChan <-chan []*OTransform
	RequestSndChan   chan<- BinderRequest
}

/*
SendTransform - A helper function for submitting a transform to the binder. The binder responds
with either an error or a corrected version number for the document at the time of your submission.
*/
func (p *BinderPortal) SendTransform(ot *OTransform, timeout time.Duration) (int, error) {
	// Buffered channels because the server skips blocked sends
	errChan := make(chan error, 1)
	verChan := make(chan int, 1)
	p.RequestSndChan <- BinderRequest{
		Transform:   ot,
		VersionChan: verChan,
		ErrorChan:   errChan,
	}
	select {
	case err := <-errChan:
		return 0, err
	case ver := <-verChan:
		return ver, nil
	case <-time.After(timeout):
	}
	return 0, errors.New("timeout occured waiting for binder response")
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

	logger     *log.Logger
	model      *OModel
	block      DocumentStore
	config     BinderConfig
	clients    [](chan<- []*OTransform)
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
) (*Binder, error) {

	binder := Binder{
		ID:            id,
		SubscribeChan: make(chan (chan<- *BinderPortal)),
		logger:        log.New(os.Stdout, "[leaps.binder] ", log.LstdFlags),
		model:         CreateModel(id),
		block:         block,
		config:        config,
		clients:       [](chan<- []*OTransform){},
		jobs:          make(chan BinderRequest),
		errorChan:     errorChan,
		closedChan:    make(chan bool),
	}

	binder.log("info", "bound to existing document, attempting flush")

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
) (*Binder, error) {

	if err := block.Store(document.ID, document); err != nil {
		return nil, err
	}

	binder := Binder{
		ID:            document.ID,
		SubscribeChan: make(chan (chan<- *BinderPortal)),
		logger:        log.New(os.Stdout, "[leaps.binder] ", log.LstdFlags),
		model:         CreateModel(document.ID),
		block:         block,
		config:        config,
		clients:       [](chan<- []*OTransform){},
		jobs:          make(chan BinderRequest),
		errorChan:     errorChan,
		closedChan:    make(chan bool),
	}

	binder.log("info", "bound to new document, attempting flush")

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
func (b *Binder) log(level, message string) {
	if b.config.LogVerbose {
		b.logger.Printf("| %v -> (%v) %v\n", level, b.ID, message)
	}
}

/*
processJob - Processes a clients sent transforms, also returns the conditioned transforms to be
broadcast to other listening clients.
*/
func (b *Binder) processJob(request BinderRequest) {
	version := b.model.Version

	if request.Transform == nil {
		select {
		case request.ErrorChan <- errors.New("received job of zero transforms"):
		default:
		}
		return
	}

	newOTs := make([]*OTransform, 1)
	var err error

	newOTs[0], err = b.model.PushTransform(*request.Transform)
	if err != nil {
		select {
		case request.ErrorChan <- err:
		default:
		}
		return
	}

	select {
	case request.VersionChan <- (version + 1):
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
	newClients := [](chan<- []*OTransform){}
	for _, clientChan := range b.clients {
		if clientChan != nil {
			newClients = append(newClients, clientChan)
		} else {
			deadClients++
		}
	}

	if deadClients > 0 {
		b.log("info", fmt.Sprintf("Kicked %v inactive clients", deadClients))
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
					b.log("error", fmt.Sprintf("flush error: %v, shutting down", err))
					running = false
				}
				/* Channel is buffered by one element to be non-blocking, any blocked send will
				 * lead to a rejected client, bad client!
				 */
				sndChan := make(chan []*OTransform, 1)
				b.clients = append(b.clients, sndChan)
				b.log("info", "subscribing new client")

				client <- &BinderPortal{
					Version:          b.model.Version,
					Document:         doc,
					Transforms:       b.model.Unapplied,
					Error:            nil,
					TransformRcvChan: sndChan,
					RequestSndChan:   b.jobs,
				}
				flushTime = time.After(flushPeriod)
			} else {
				b.log("info", "subscribe channel closed, shutting down")
				running = false
			}
		case job, open := <-b.jobs:
			if running && open {
				b.processJob(job)
			} else {
				b.log("info", "jobs channel closed, shutting down")
				running = false
			}
		case <-flushTime:
			if _, err := b.flush(); err != nil {
				b.log("error", fmt.Sprintf("flush error: %v, shutting down", err))
				b.errorChan <- BinderError{ID: b.ID, Err: err}
				running = false
			}
			flushTime = time.After(flushPeriod)
		}
		if !running {
			b.log("info", "closing, shutting down client channels")
			oldClients := b.clients[:]
			b.clients = [](chan<- []*OTransform){}
			for _, client := range oldClients {
				close(client)
			}
			b.log("info", fmt.Sprintf("attempting final flush of %v", b.ID))
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
