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
	"github.com/jeffail/leaps/util"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
LeapTextClientMessage - A structure that defines a message format to expect from clients connected
to a text model. Commands can currently be 'submit' (submit a transform to a bound document), or
'update' (submit an update to the users cursor position).
*/
type LeapTextClientMessage struct {
	Command   string          `json:"command"`
	Transform *lib.OTransform `json:"transform,omitempty"`
	Position  *int64          `json:"position,omitempty"`
	Message   string          `json:"message,omitempty"`
}

/*
LeapTextServerMessage - A structure that defines a response message from a text model to a client.
Type can be 'transforms' (continuous delivery), 'correction' (actual version of a submitted
transform), 'update' (an update to a users status) or 'error' (an error message to display to the
client).
*/
type LeapTextServerMessage struct {
	Type       string              `json:"response_type"`
	Transforms []lib.OTransform    `json:"transforms,omitempty"`
	Updates    []lib.ClientMessage `json:"user_updates,omitempty"`
	Version    int                 `json:"version,omitempty"`
	Error      string              `json:"error,omitempty"`
}

/*--------------------------------------------------------------------------------------------------
 */

/*
HTTPTextModel - an HTTP model that connects a binder of a text document to a client.
*/
type HTTPTextModel struct {
	config    HTTPBinderConfig
	logger    *util.Logger
	stats     *util.Stats
	closeChan <-chan bool
}

/*
LaunchWebsocketTextModel - Launches a text model that wraps a connected websocket around a
BinderPortal representing a text document.
*/
func LaunchWebsocketTextModel(h *HTTPTextModel, socket *websocket.Conn, binder lib.BinderPortal) {
	bindTOut := time.Duration(h.config.BindSendTimeout) * time.Millisecond

	defer func() {
		binder.Exit(bindTOut)
	}()

	// TODO: Preserve reference of doc ID?
	binder.Document = nil

	readChan := make(chan LeapTextClientMessage)
	go func() {
		for {
			select {
			case <-h.closeChan:
				h.logger.Debugln("Closing websocket model")
				close(readChan)
				return
			default:
			}
			var clientMsg LeapTextClientMessage
			if err := websocket.JSON.Receive(socket, &clientMsg); err == nil {
				readChan <- clientMsg
			} else {
				close(readChan)
				return
			}
		}
	}()

	h.stats.Incr("http.client.connected", 1)

	for {
		select {
		case msg, open := <-readChan:
			if !open {
				binder.SendMessage(lib.ClientMessage{
					Active: false,
					Token:  binder.Token,
				})
				h.logger.Debugln("Closing websocket due to closed read channel")
				return
			}
			h.logger.Tracef("Received %v command from client\n", msg.Command)
			switch msg.Command {
			case "submit":
				if msg.Transform == nil {
					h.logger.Errorln("Client submit contained nil transform")
					websocket.JSON.Send(socket, LeapTextServerMessage{
						Type:  "error",
						Error: "submit error: transform was nil",
					})
					h.logger.Debugln("Closing websocket due to nil transform")
					binder.SendMessage(lib.ClientMessage{
						Active: false,
						Token:  binder.Token,
					})
					return
				}
				if ver, err := binder.SendTransform(*msg.Transform, bindTOut); err == nil {
					h.logger.Traceln("Sending correction to client")
					websocket.JSON.Send(socket, LeapTextServerMessage{
						Type:    "correction",
						Version: ver,
					})
				} else {
					h.logger.Errorf("Transform request failed %v\n", err)
					websocket.JSON.Send(socket, LeapTextServerMessage{
						Type:  "error",
						Error: fmt.Sprintf("submit error: %v", err),
					})
					h.logger.Debugln("Closing websocket due to failed transform send")
					binder.SendMessage(lib.ClientMessage{
						Active: false,
						Token:  binder.Token,
					})
					return
				}
			case "update":
				if msg.Position != nil || len(msg.Message) > 0 {
					binder.SendMessage(lib.ClientMessage{
						Message:  msg.Message,
						Position: msg.Position,
						Active:   true,
						Token:    binder.Token,
					})
				}
			case "ping":
				// Do nothing
			default:
				websocket.JSON.Send(socket, LeapTextServerMessage{
					Type:  "error",
					Error: "command not recognised",
				})
			}
		case tform, open := <-binder.TransformRcvChan:
			if !open {
				h.logger.Debugln("Closing websocket due to closed transform channel")
				return
			}
			h.logger.Traceln("Sending transform to client")
			websocket.JSON.Send(socket, LeapTextServerMessage{
				Type:       "transforms",
				Transforms: []lib.OTransform{tform},
			})
		case msg, open := <-binder.MessageRcvChan:
			if !open {
				h.logger.Debugln("Closing websocket due to closed message channel")
				return
			}
			h.logger.Traceln("Sending update to client")
			websocket.JSON.Send(socket, LeapTextServerMessage{
				Type:    "update",
				Updates: []lib.ClientMessage{msg},
			})
		case <-h.closeChan:
			h.stats.Decr("http.client.connected", 1)
			h.logger.Debugln("Closing websocket due to closed channel signal")
			return
		}
	}
}

/*--------------------------------------------------------------------------------------------------
 */
