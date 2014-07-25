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

package leapnet

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"github.com/jeffail/leaps/leaplib"
	"time"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
LeapTextClientMessage - A structure that defines a message format to expect from clients connected
to a text model. Commands can currently be 'submit' (submit a transform to a bound document), or
'update' (submit an update to the users cursor position).
*/
type LeapTextClientMessage struct {
	Command   string              `json:"command"`
	Transform *leaplib.OTransform `json:"transform,omitempty"`
	Position  *int64              `json:"position,omitempty"`
	Message   string              `json:"message,omitempty"`
}

/*
LeapTextServerMessage - A structure that defines a response message from a text model to a client.
Type can be 'transforms' (continuous delivery), 'correction' (actual version of a submitted
transform), 'update' (an update to a users status) or 'error' (an error message to display to the
client).
*/
type LeapTextServerMessage struct {
	Type       string               `json:"response_type"`
	Transforms []leaplib.OTransform `json:"transforms,omitempty"`
	Updates    []leaplib.UserUpdate `json:"user_updates,omitempty"`
	Version    int                  `json:"version,omitempty"`
	Error      string               `json:"error,omitempty"`
}

/*--------------------------------------------------------------------------------------------------
 */

/*
HTTPTextModel - an HTTP model that connects a binder of a text document to a client.
*/
type HTTPTextModel struct {
	config    HTTPBinderConfig
	logger    *leaplib.LeapsLogger
	closeChan <-chan bool
}

/*
log - Helper function for logging events, only actually logs when verbose logging is configured.
*/
func (h *HTTPTextModel) log(level int, message string) {
	h.logger.Log(level, "http_text", message)
}

/*
LaunchWebsocketTextModel - Launches a text model that wraps a connected websocket around a
BinderPortal representing a text document.
*/
func LaunchWebsocketTextModel(h *HTTPTextModel, socket *websocket.Conn, binder *leaplib.BinderPortal) {
	bindTOut := time.Duration(h.config.BindSendTimeout) * time.Millisecond

	// TODO: Preserve reference of doc ID?
	binder.Document = nil

	readChan := make(chan LeapTextClientMessage)
	go func() {
		for {
			select {
			case <-h.closeChan:
				h.log(leaplib.LeapInfo, "Closing websocket model")
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

	for {
		select {
		case msg, open := <-readChan:
			if !open {
				if err := binder.SendUpdate(leaplib.UserUpdate{
					Active: false,
					Token:  binder.Token,
				}, bindTOut); err != nil {
					h.log(leaplib.LeapError, fmt.Sprintf("Client update failed %v", err))
				}
				return
			}
			h.log(leaplib.LeapDebug, fmt.Sprintf("Received %v command from client", msg.Command))
			switch msg.Command {
			case "submit":
				if msg.Transform == nil {
					h.log(leaplib.LeapError, "Client submit contained nil transform")
					websocket.JSON.Send(socket, LeapTextServerMessage{
						Type:  "error",
						Error: "submit error: transform was nil",
					})
					return
				}
				if ver, err := binder.SendTransform(*msg.Transform, bindTOut); err == nil {
					h.log(leaplib.LeapDebug, "Sending correction to client")
					websocket.JSON.Send(socket, LeapTextServerMessage{
						Type:    "correction",
						Version: ver,
					})
				} else {
					h.log(leaplib.LeapError, fmt.Sprintf("Transform request failed %v", err))
					websocket.JSON.Send(socket, LeapTextServerMessage{
						Type:  "error",
						Error: fmt.Sprintf("submit error: %v", err),
					})
					return
				}
			case "update":
				if msg.Position != nil || len(msg.Message) > 0 {
					if err := binder.SendUpdate(leaplib.UserUpdate{
						Message:  msg.Message,
						Position: msg.Position,
						Active:   true,
						Token:    binder.Token,
					}, bindTOut); err != nil {
						h.log(leaplib.LeapError, fmt.Sprintf("Client update failed %v", err))
						websocket.JSON.Send(socket, LeapTextServerMessage{
							Type:  "error",
							Error: fmt.Sprintf("update error: %v", err),
						})
						return
					}
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
				return
			}
			if tform != nil {
				if ot, ok := tform.(leaplib.OTransform); ok {
					h.log(leaplib.LeapDebug, "Sending transform to client")
					websocket.JSON.Send(socket, LeapTextServerMessage{
						Type:       "transforms",
						Transforms: []leaplib.OTransform{ot},
					})
				} else if update, ok := tform.(leaplib.UserUpdate); ok {
					h.log(leaplib.LeapDebug, "Sending update to client")
					websocket.JSON.Send(socket, LeapTextServerMessage{
						Type:    "update",
						Updates: []leaplib.UserUpdate{update},
					})
				} else {
					h.log(leaplib.LeapError, fmt.Sprintf("Received unexpected type from RcvChan: %v", tform))
				}
			}
		case <-h.closeChan:
			return
		}
	}
}

/*--------------------------------------------------------------------------------------------------
 */
