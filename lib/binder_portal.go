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
TransformSubmission - A struct used to submit a transform to a binder. The submission must contain
the token of the client, as well as two channels for returning either the corrected version of the
transform if successful, or an error if the submit was unsuccessful.
*/
type TransformSubmission struct {
	Token       string
	Transform   OTransform
	VersionChan chan<- int
	ErrorChan   chan<- error
}

/*
MessageSubmission - A struct used to submit a message to a binder. The submission must contain the
token of the client in order to avoid the message being sent back to the same client.
*/
type MessageSubmission struct {
	Token   string
	Message ClientMessage
}

/*
BinderSubscribeBundle - A container that holds all data necessary to provide a binder that you
wish to subscribe to. Contains a user token for identifying the client and a channel for
receiving the resultant BinderPortal.
*/
type BinderSubscribeBundle struct {
	Token         string
	PortalRcvChan chan<- BinderPortal
}

/*--------------------------------------------------------------------------------------------------
 */

// Errors for the BinderPortal type.
var (
	ErrTimeout = errors.New("timed out")
)

/*
BinderPortal - A container that holds all data necessary to begin an open portal with the binder,
allowing fresh transforms to be submitted and returned as they come. Also carries the token of the
client.
*/
type BinderPortal struct {
	Token            string
	Document         Document
	Version          int
	Error            error
	TransformRcvChan <-chan OTransform
	MessageRcvChan   <-chan ClientMessage
	TransformSndChan chan<- TransformSubmission
	MessageSndChan   chan<- MessageSubmission
	ExitChan         chan<- string
}

/*
SendTransform - Submits a transform to the binder. The binder responds with either an error or a
corrected version number for the transform. This is safe to call from any goroutine.
*/
func (p *BinderPortal) SendTransform(ot OTransform, timeout time.Duration) (int, error) {
	// Buffered channels because the server skips blocked sends
	errChan := make(chan error, 1)
	verChan := make(chan int, 1)
	p.TransformSndChan <- TransformSubmission{
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
	return 0, ErrTimeout
}

/*
SendMessage - Sends a message to the binder, which is subsequently sent out to all other clients.
This is safe to call from any goroutine.
*/
func (p *BinderPortal) SendMessage(message ClientMessage) {
	p.MessageSndChan <- MessageSubmission{
		Token:   p.Token,
		Message: message,
	}
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
