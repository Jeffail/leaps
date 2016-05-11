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

package http

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jeffail/leaps/lib/binder"
	"github.com/jeffail/leaps/lib/curator"
	"github.com/jeffail/leaps/lib/store"
	"github.com/jeffail/leaps/lib/text"
	"github.com/jeffail/util/log"
	"github.com/jeffail/util/metrics"
	"golang.org/x/net/websocket"
)

//--------------------------------------------------------------------------------------------------

/*
leapHTTPClientMessage - A structure that defines a message format to expect from clients. Commands
can be 'create' (init with new document), 'edit' (edit existing document) or 'read' (read only
existing document).
*/
type leapHTTPClientMessage struct {
	Command  string          `json:"command"`
	Token    string          `json:"token"`
	DocID    string          `json:"document_id,omitempty"`
	UserID   string          `json:"user_id"`
	Document *store.Document `json:"leap_document,omitempty"`
}

/*
leapHTTPServerMessage - A structure that defines a response message from the server to a client.
Type can be 'document' (init response) or 'error' (an error message to display to the client).
*/
type leapHTTPServerMessage struct {
	Type     string         `json:"response_type"`
	Document store.Document `json:"leap_document,omitempty"`
	Version  *int           `json:"version,omitempty"`
	Error    string         `json:"error,omitempty"`
}

//--------------------------------------------------------------------------------------------------

// Common errors for the http package.
var (
	ErrInvalidDocument = errors.New("invalid document structure")
	ErrInvalidUserID   = errors.New("invalid user ID")
)

/*
WebsocketHandler - Returns a websocket handler that routes new websockets to a curator. Use this
with an HTTP server with the "golang.org/x/net/websocket" package.
*/
func WebsocketHandler(
	finder curator.Type,
	timeout time.Duration,
	logger log.Modular,
	stats metrics.Aggregator,
) func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		var err error
		var session binder.Portal

		defer func() {
			if err != nil {
				websocket.JSON.Send(ws, leapHTTPServerMessage{
					Type:  "error",
					Error: fmt.Sprintf("socket initialization failed: %v", err),
				})
			}
			if err = ws.Close(); err != nil {
				logger.Errorf("Failed to close socket: %v\n", err)
			}
			stats.Decr("http.open_websockets", 1)
		}()

		stats.Incr("http.websocket.opened", 1)
		stats.Incr("http.open_websockets", 1)

		for session == nil && err == nil {
			var clientMsg leapHTTPClientMessage
			websocket.JSON.Receive(ws, &clientMsg)

			switch clientMsg.Command {
			case "create":
				if clientMsg.Document == nil {
					err = ErrInvalidDocument
				} else {
					session, err = finder.CreateDocument(
						clientMsg.UserID, clientMsg.Token, *clientMsg.Document, timeout)
				}
			case "read":
				if len(clientMsg.DocID) <= 0 {
					err = ErrInvalidDocument
				} else {
					session, err = finder.ReadDocument(
						clientMsg.UserID, clientMsg.Token, clientMsg.DocID, timeout)
				}
			case "edit":
				if len(clientMsg.DocID) <= 0 {
					err = ErrInvalidDocument
				} else {
					session, err = finder.EditDocument(
						clientMsg.UserID, clientMsg.Token, clientMsg.DocID, timeout)
				}
			case "ping":
				// Ignore and continue waiting for init message.
			default:
				err = fmt.Errorf(
					"first command must be init or ping, client sent: %v", clientMsg.Command,
				)
			}
		}

		if session != nil && err == nil {
			version := session.BaseVersion()
			websocket.JSON.Send(ws, leapHTTPServerMessage{
				Type:     "document",
				Document: session.Document(),
				Version:  &version,
			})
			session.ReleaseDocument()

			// Begin serving websocket IO.
			serveWebsocketIO(ws, session, timeout, logger, stats)
		}
	}
}

//--------------------------------------------------------------------------------------------------

/*
leapSocketClientMessage - A structure that defines a message format to expect from clients connected
to a text model. Commands can currently be 'submit' (submit a transform to a bound document), or
'update' (submit an update to the users cursor position).
*/
type leapSocketClientMessage struct {
	Command   string           `json:"command"`
	Transform *text.OTransform `json:"transform,omitempty"`
	Position  *int64           `json:"position,omitempty"`
	Message   string           `json:"message,omitempty"`
}

/*
leapSocketServerMessage - A structure that defines a response message from a text model to a client.
Type can be 'transforms' (continuous delivery), 'correction' (actual version of a submitted
transform), 'update' (an update to a users status) or 'error' (an error message to display to the
client).
*/
type leapSocketServerMessage struct {
	Type       string                `json:"response_type"`
	Transforms []text.OTransform     `json:"transforms,omitempty"`
	Updates    []binder.ClientUpdate `json:"user_updates,omitempty"`
	Version    int                   `json:"version,omitempty"`
	Error      string                `json:"error,omitempty"`
}

//--------------------------------------------------------------------------------------------------

func serveWebsocketIO(
	ws *websocket.Conn,
	portal binder.Portal,
	timeout time.Duration,
	logger log.Modular,
	stats metrics.Aggregator,
) {
	defer portal.Exit(timeout)

	// Signal to close
	var incomingCloseInt uint32
	outgoingCloseChan := make(chan struct{})

	// Signals that goroutine is closing
	incomingClosedChan := make(chan struct{})
	outgoingClosedChan := make(chan struct{})

	// Loop incoming messages.
	go func() {
		var err error
		defer func() {
			if err != nil {
				websocket.JSON.Send(ws, leapSocketServerMessage{
					Type:  "error",
					Error: err.Error(),
				})
			}
			close(incomingClosedChan)
		}()

		for atomic.LoadUint32(&incomingCloseInt) == 0 {
			var msg leapSocketClientMessage
			if socketErr := websocket.JSON.Receive(ws, &msg); socketErr != nil {
				return
			}
			switch msg.Command {
			case "submit":
				if msg.Transform == nil {
					err = errors.New("submit error: transform was nil")
					return
				}
				var ver int
				if ver, err = portal.SendTransform(*msg.Transform, timeout); err != nil {
					return
				}
				websocket.JSON.Send(ws, leapSocketServerMessage{
					Type:    "correction",
					Version: ver,
				})
			case "update":
				if msg.Position != nil || len(msg.Message) > 0 {
					portal.SendMessage(binder.Message{
						Content:  msg.Message,
						Position: msg.Position,
						Active:   true,
					})
				}
			case "ping":
				// Do nothing
			default:
				err = errors.New("command not recognised")
			}
		}
	}()

	// Loop outgoing messages.
	go func() {
		defer close(outgoingClosedChan)
		for {
			select {
			case <-outgoingCloseChan:
				return
			case tform, open := <-portal.TransformReadChan():
				if !open {
					return
				}
				websocket.JSON.Send(ws, leapSocketServerMessage{
					Type:       "transforms",
					Transforms: []text.OTransform{tform},
				})
			case msg, open := <-portal.UpdateReadChan():
				if !open {
					return
				}
				websocket.JSON.Send(ws, leapSocketServerMessage{
					Type:    "update",
					Updates: []binder.ClientUpdate{msg},
				})
			}
		}
	}()

	// If one channel closes, close the other, if the socket is being closed then close both.
	select {
	case <-incomingClosedChan:
		close(outgoingCloseChan)
		<-outgoingClosedChan
		portal.SendMessage(binder.Message{
			Active: false,
		})
	case <-outgoingClosedChan:
		atomic.StoreUint32(&incomingCloseInt, 1)
		<-incomingClosedChan
		portal.SendMessage(binder.Message{
			Active: false,
		})
	}
}

//--------------------------------------------------------------------------------------------------
