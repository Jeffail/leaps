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
	"time"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
Model - an interface that represents an internal operation transform model of a particular type.
Initially text is the only supported transform model, however, the plan will eventually be to have
different models for various types of document that should all be supported by our binder.
*/
type Model interface {
	/* PushTransform - Push a single transform to our model, and if successful, return the updated
	 * transform along with the new version of the document.
	 */
	PushTransform(ot interface{}) (interface{}, int, error)

	/* FlushTransforms - apply all unapplied transforms to content, and delete old applied
	 * in accordance with our retention period. Returns a bool indicating whether any changes
	 * were applied, and an error in case a fatal problem was encountered.
	 */
	FlushTransforms(content *interface{}, retention time.Duration) (bool, error)

	/* GetVersion - returns the current version of the document.
	 */
	GetVersion() int
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
	Transform   interface{}
	VersionChan chan<- int
	ErrorChan   chan<- error
}

/*
BinderPortal - A container that holds all data necessary to begin an open portal with the binder,
allowing fresh transforms to be submitted and returned as they come.
*/
type BinderPortal struct {
	Document         *Document
	Version          int
	Error            error
	TransformRcvChan <-chan []interface{}
	RequestSndChan   chan<- BinderRequest
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
