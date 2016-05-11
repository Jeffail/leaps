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

import "github.com/jeffail/leaps/lib/text"

//--------------------------------------------------------------------------------------------------

/*
Error - A binder has encountered a problem and needs to close. In order for this to happen it needs
to inform its owner that it should be shut down. Error is a structure used to carry our error
message and our ID over an error channel. A Error with the Err set to nil can be used as a
graceful shutdown request.
*/
type Error struct {
	ID  string
	Err error
}

//--------------------------------------------------------------------------------------------------

// Message - Content, optional cursor position and a flag representing whether the client is active.
type Message struct {
	Content  string `json:"content,omitempty"`
	Position *int64 `json:"position,omitempty"`
	Active   bool   `json:"active"`
}

// ClientInfo - Contains user session credentials (user id and session id).
type ClientInfo struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}

// ClientUpdate - A struct broadcast to all clients containing an update/message.
type ClientUpdate struct {
	ClientInfo ClientInfo `json:"client"`
	Message    Message    `json:"message"`
}

//--------------------------------------------------------------------------------------------------

/*
transformSubmission - A struct used to submit a transform to an active binder, the struct carries
data about the client as well as the transform itself, along with channels used for returning the
response from the binder (error or version adjustment).
*/
type transformSubmission struct {
	client      *binderClient
	transform   text.OTransform
	versionChan chan<- int
	errorChan   chan<- error
}

/*
messageSubmission - A struct used to submit a message to a binder, carrying data about the client as
well as the message contents.
*/
type messageSubmission struct {
	client  *binderClient
	message Message
}

//--------------------------------------------------------------------------------------------------

/*
binderClient - A struct containing information about a connected client and channels used by the
binder to push transforms and user updates out.
*/
type binderClient struct {
	userID    string
	sessionID string

	transformChan chan<- text.OTransform
	updateChan    chan<- ClientUpdate
}

//--------------------------------------------------------------------------------------------------
