/*
Copyright (c) 2017 Ashley Jeffs

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

package api

import "github.com/Jeffail/leaps/lib/api/events"

//------------------------------------------------------------------------------

// RequestHandler - Called when a request has been received, receives the body
// of the request and returns a TypedError indicating whether the handler was
// successful. Errors are sent back to the client.
type RequestHandler func(body []byte) events.TypedError

// ResponseHandler - Called when an outgoing response is about to be sent,
// receives the body of the response and returns a bool indicating whether the
// response should be sent (false == do not send).
type ResponseHandler func(body interface{}) bool

// EventHandler - Called on a connection related event (open, close, etc)
type EventHandler func()

// Emitter - To be instantiated for each connected client. Allows components to
// implement the leaps service API by registering their request and event
// handlers. The emitter then handles networked traffic and brokers incoming
// requests to those registered components. All incoming messages are expected
// to be of the JSON format:
//
//   {
//     "type": "<type_string>",
//     "body": {...}
//   }
//
// Handlers for request types are given the unparsed JSON body of the request.
// It is guaranteed that events will NOT be triggered in parallel, although
// they are not guaranteed to come from the same goroutine.
type Emitter interface {

	// OnReceive - Register a handler for a particular incoming event type.
	OnReceive(reqType string, handler RequestHandler)

	// OnSend - Register a handler for a particular outgoing event type.
	OnSend(resType string, handler ResponseHandler)

	// OnClose - Register an event handler for a close event.
	OnClose(eventHandler EventHandler)

	// Send - Send data out to the client.
	Send(resType string, body interface{}) error
}

//------------------------------------------------------------------------------
