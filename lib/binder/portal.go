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
	"errors"
	"time"

	"github.com/jeffail/leaps/lib/store"
	"github.com/jeffail/leaps/lib/text"
)

//--------------------------------------------------------------------------------------------------

// Errors for the binder portal type.
var (
	ErrReadOnlyPortal = errors.New("attempting to send transforms through a READ ONLY portal")
)

//--------------------------------------------------------------------------------------------------

/*
portalImpl - A container that holds all data necessary to begin an open portal with the binder,
allowing fresh transforms to be submitted and returned as they come. Also carries the BinderClient
of the client.
*/
type portalImpl struct {
	client   *binderClient
	document store.Document
	version  int

	transformRcvChan <-chan text.OTransform
	updateRcvChan    <-chan ClientUpdate

	transformSndChan chan<- transformSubmission
	messageSndChan   chan<- messageSubmission
	exitChan         chan<- *binderClient
}

// UserID - Returns the user ID of this portal.
func (p *portalImpl) UserID() string {
	return p.client.userID
}

// SessionID - Returns the session ID of this portal.
func (p *portalImpl) SessionID() string {
	return p.client.sessionID
}

// BaseVersion - Returns the version of the binder when this session opened.
func (p *portalImpl) BaseVersion() int {
	return p.version
}

// Document - Returns the document contents as it was when the session was opened.
func (p *portalImpl) Document() store.Document {
	return p.document
}

// ReleaseDocument - Releases the cached document.
func (p *portalImpl) ReleaseDocument() {
	p.document = store.Document{}
}

// TransformReadChan - Returns a channel for receiving live transforms from the binder.
func (p *portalImpl) TransformReadChan() <-chan text.OTransform {
	return p.transformRcvChan
}

// UpdateReadChan - Returns a channel for receiving meta updates from the binder.
func (p *portalImpl) UpdateReadChan() <-chan ClientUpdate {
	return p.updateRcvChan
}

/*
SendTransform - Submits a transform to the binder. The binder responds with either an error or a
corrected version number for the transform. This is safe to call from any goroutine.
*/
func (p *portalImpl) SendTransform(ot text.OTransform, timeout time.Duration) (int, error) {
	// Check if we are READ ONLY
	if nil == p.transformSndChan {
		return 0, ErrReadOnlyPortal
	}
	// Buffered channels because the server skips blocked sends
	errChan := make(chan error, 1)
	verChan := make(chan int, 1)
	p.transformSndChan <- transformSubmission{
		client:      p.client,
		transform:   ot,
		versionChan: verChan,
		errorChan:   errChan,
	}
	select {
	case err := <-errChan:
		return 0, err
	case ver := <-verChan:
		return ver, nil
	case <-time.After(timeout):
	}
	return 0, ErrTimeout
}

/*
SendMessage - Sends a message to the binder, which is subsequently sent out to all other clients.
This is safe to call from any goroutine.
*/
func (p *portalImpl) SendMessage(message Message) {
	p.messageSndChan <- messageSubmission{
		client:  p.client,
		message: message,
	}
}

// Exit - Inform the binder that this client is shutting down.
func (p *portalImpl) Exit(timeout time.Duration) {
	select {
	case p.exitChan <- p.client:
	case <-time.After(timeout):
	}
}

//--------------------------------------------------------------------------------------------------
