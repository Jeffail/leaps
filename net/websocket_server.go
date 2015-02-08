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

package net

import (
	"fmt"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/jeffail/leaps/lib"
	"github.com/jeffail/util/log"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
LeapSocketClientMessage - A structure that defines a message format to expect from clients connected
to a text model. Commands can currently be 'submit' (submit a transform to a bound document), or
'update' (submit an update to the users cursor position).
*/
type LeapSocketClientMessage struct {
	Command   string          `json:"command" yaml:"command"`
	Transform *lib.OTransform `json:"transform,omitempty" yaml:"transform,omitempty"`
	Position  *int64          `json:"position,omitempty" yaml:"position,omitempty"`
	Message   string          `json:"message,omitempty" yaml:"message,omitempty"`
}

/*
LeapSocketServerMessage - A structure that defines a response message from a text model to a client.
Type can be 'transforms' (continuous delivery), 'correction' (actual version of a submitted
transform), 'update' (an update to a users status) or 'error' (an error message to display to the
client).
*/
type LeapSocketServerMessage struct {
	Type       string              `json:"response_type" yaml:"response_type"`
	Transforms []lib.OTransform    `json:"transforms,omitempty" yaml:"transforms,omitempty"`
	Updates    []lib.ClientMessage `json:"user_updates,omitempty" yaml:"user_updates,omitempty"`
	Version    int                 `json:"version,omitempty" yaml:"version,omitempty"`
	Error      string              `json:"error,omitempty" yaml:"error,omitempty"`
}

/*--------------------------------------------------------------------------------------------------
 */

/*
WebsocketServer - A websocket client that connects a binder of a document to a websocket client.
*/
type WebsocketServer struct {
	config    HTTPBinderConfig
	logger    *log.Logger
	stats     *log.Stats
	socket    *websocket.Conn
	binder    lib.BinderPortal
	closeChan <-chan bool
}

/*
NewWebsocketServer - Creates a new HTTP websocket client.
*/
func NewWebsocketServer(
	config HTTPBinderConfig,
	socket *websocket.Conn,
	binder lib.BinderPortal,
	closeChan <-chan bool,
	logger *log.Logger,
	stats *log.Stats,
) *WebsocketServer {
	return &WebsocketServer{
		config:    config,
		socket:    socket,
		binder:    binder,
		closeChan: closeChan,
		logger:    logger.NewModule("[socket]"),
		stats:     stats,
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Launch - Launches the client, wrapping two goroutines around a connected websocket and a
BinderPortal. This call spawns two goroutines and blocks until both are closed. One goroutine
manages incoming messages routing through to the binder, the other manages outgoing messages routing
back through the websocket.
*/
func (w *WebsocketServer) Launch() {
	bindTOut := time.Duration(w.config.BindSendTimeout) * time.Millisecond

	// TODO: Preserve reference of doc ID?
	w.binder.Document = lib.Document{}

	defer func() {
		w.binder.Exit(bindTOut)
	}()

	// Signal to close
	incomingCloseChan := make(chan struct{})
	outgoingCloseChan := make(chan struct{})

	// Signals that goroutine is closing
	incomingClosedChan := make(chan struct{})
	outgoingClosedChan := make(chan struct{})

	go w.loopIncoming(incomingClosedChan, incomingCloseChan)
	go w.loopOutgoing(outgoingClosedChan, outgoingCloseChan)

	// If one channel closes, close the other, if the socket is being closed then close both.
	select {
	case <-incomingClosedChan:
		close(outgoingCloseChan)
		<-outgoingClosedChan
		w.binder.SendMessage(lib.ClientMessage{
			Active: false,
			Token:  w.binder.Token,
		})
	case <-outgoingClosedChan:
		close(incomingCloseChan)
		<-incomingClosedChan
		w.binder.SendMessage(lib.ClientMessage{
			Active: false,
			Token:  w.binder.Token,
		})
	case <-w.closeChan:
		close(incomingCloseChan)
		close(outgoingCloseChan)
		<-incomingClosedChan
		<-outgoingClosedChan
	}
}

func (w *WebsocketServer) loopIncoming(closeSignalChan chan<- struct{}, closeCmdChan <-chan struct{}) {
	bindTOut := time.Duration(w.config.BindSendTimeout) * time.Millisecond

	for {
		select {
		case <-closeCmdChan:
			w.logger.Debugln("Closing websocket incoming router")
			closeSignalChan <- struct{}{}
			return
		default:
		}

		var msg LeapSocketClientMessage
		if err := websocket.JSON.Receive(w.socket, &msg); err == nil {
			w.logger.Tracef("Received %v command from client\n", msg.Command)

			switch msg.Command {
			case "submit":
				if msg.Transform == nil {
					w.logger.Errorln("Client submit contained nil transform")
					websocket.JSON.Send(w.socket, LeapSocketServerMessage{
						Type:  "error",
						Error: "submit error: transform was nil",
					})
					w.logger.Debugln("Closing websocket due to nil transform")
					closeSignalChan <- struct{}{}
					return
				}
				if ver, err := w.binder.SendTransform(*msg.Transform, bindTOut); err == nil {
					w.logger.Traceln("Sending correction to client")
					websocket.JSON.Send(w.socket, LeapSocketServerMessage{
						Type:    "correction",
						Version: ver,
					})
				} else {
					w.logger.Errorf("Transform request failed %v\n", err)
					websocket.JSON.Send(w.socket, LeapSocketServerMessage{
						Type:  "error",
						Error: fmt.Sprintf("submit error: %v", err),
					})
					w.logger.Debugln("Closing websocket due to failed transform send")
					closeSignalChan <- struct{}{}
					return
				}
			case "update":
				if msg.Position != nil || len(msg.Message) > 0 {
					w.binder.SendMessage(lib.ClientMessage{
						Message:  msg.Message,
						Position: msg.Position,
						Active:   true,
						Token:    w.binder.Token,
					})
				}
			case "ping":
				// Do nothing
			default:
				websocket.JSON.Send(w.socket, LeapSocketServerMessage{
					Type:  "error",
					Error: "command not recognised",
				})
			}
		} else {
			w.logger.Traceln("Websocket closed, closing client")
			closeSignalChan <- struct{}{}
			return
		}
	}
}

func (w *WebsocketServer) loopOutgoing(closeSignalChan chan<- struct{}, closeCmdChan <-chan struct{}) {
	for {
		select {
		case <-closeCmdChan:
			w.logger.Debugln("Closing websocket outgoing router")
			closeSignalChan <- struct{}{}
			return
		case tform, open := <-w.binder.TransformRcvChan:
			if !open {
				w.logger.Debugln("Closing websocket due to closed transform channel")
				closeSignalChan <- struct{}{}
				return
			}
			w.logger.Traceln("Sending transform to client")
			websocket.JSON.Send(w.socket, LeapSocketServerMessage{
				Type:       "transforms",
				Transforms: []lib.OTransform{tform},
			})
		case msg, open := <-w.binder.MessageRcvChan:
			if !open {
				w.logger.Debugln("Closing websocket due to closed message channel")
				closeSignalChan <- struct{}{}
				return
			}
			w.logger.Traceln("Sending update to client")
			websocket.JSON.Send(w.socket, LeapSocketServerMessage{
				Type:    "update",
				Updates: []lib.ClientMessage{msg},
			})
		}
	}
}

/*--------------------------------------------------------------------------------------------------
 */
