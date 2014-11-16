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
	"time"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
UserUpdate - A struct containing an update for a clients' status.
*/
type UserUpdate struct {
	Message  string `json:"message,omitempty"`
	Position *int64 `json:"position,omitempty"`
	Active   bool   `json:"active"`
	Token    string `json:"user_id"`
}

/*--------------------------------------------------------------------------------------------------
 */

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

/*
BinderRequest - A container used to communicate with a binder, it holds a transform to be
submitted to the document model. Two channels are used for return values from the request.
VersionChan is used to send back the actual version of the transform submitted. ErrorChan is used to
send errors that occur. Both channels must be non-blocking, so a buffer of 1 is recommended.
*/
type BinderRequest struct {
	Token       string
	Transform   interface{}
	VersionChan chan<- int
	ErrorChan   chan<- error
}

/*
BinderSubscribeBundle - A container that holds all data necessary to provide a binder that you
wish to subscribe to. Contains a user token for identifying the client and a channel for
receiving the resultant BinderPortal.
*/
type BinderSubscribeBundle struct {
	Token         string
	PortalRcvChan chan<- *BinderPortal
}

/*--------------------------------------------------------------------------------------------------
 */

/*
BinderPortal - A container that holds all data necessary to begin an open portal with the binder,
allowing fresh transforms to be submitted and returned as they come. Also carries the token of the
client.
*/
type BinderPortal struct {
	Token            string
	Document         *Document
	Version          int
	Error            error
	TransformRcvChan <-chan interface{}
	RequestSndChan   chan<- BinderRequest
	ExitChan         chan<- string
}

/*
SendTransform - A helper function for submitting a transform to the binder. The binder responds
with either an error or a corrected version number for the document at the time of your submission.
*/
func (p *BinderPortal) SendTransform(ot interface{}, timeout time.Duration) (int, error) {
	// Buffered channels because the server skips blocked sends
	errChan := make(chan error, 1)
	verChan := make(chan int, 1)
	p.RequestSndChan <- BinderRequest{
		Token:       p.Token,
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

/*
SendUpdate - A helper function for submitting an update to the binder. The binder will return an
error in the event of one.
*/
func (p *BinderPortal) SendUpdate(update interface{}, timeout time.Duration) error {
	// Buffered channels because the server skips blocked sends
	errChan := make(chan error, 1)
	p.RequestSndChan <- BinderRequest{
		Token:       p.Token,
		Transform:   update,
		VersionChan: nil,
		ErrorChan:   errChan,
	}
	select {
	case err := <-errChan:
		return err
	case <-time.After(timeout):
	}
	return errors.New("timeout occured waiting for binder response")
}

/*
Exit - Inform the binder that this client is shutting down.
*/
func (p *BinderPortal) Exit(timeout time.Duration) {
	select {
	case p.ExitChan <- p.Token:
	case <-time.After(timeout):
	}
}

/*--------------------------------------------------------------------------------------------------
 */
