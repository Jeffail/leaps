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
to a text model. Commands can currently only be 'submit' (submit a transform to a bound document).
*/
type LeapTextClientMessage struct {
	Command   string             `json:"command"`
	Transform leaplib.OTransform `json:"transform,omitempty"`
}

/*
LeapTextServerMessage - A structure that defines a response message from a text model to a client.
Type can be 'transforms' (continuous delivery), 'correction' (actual version of a submitted
transform), or 'error' (an error message to display to the client).
*/
type LeapTextServerMessage struct {
	Type       string               `json:"response_type"`
	Transforms []leaplib.OTransform `json:"transforms,omitempty"`
	Version    *int                 `json:"version,omitempty"`
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
	defer func() {
		if err := socket.Close(); err != nil {
			h.log(leaplib.LeapError, fmt.Sprintf("Failed to close socket: %v", err))
		}
	}()

	bindTOut := time.Duration(h.config.BindSendTimeout) * time.Millisecond

	// TODO: Preserve reference of doc ID?
	binder.Document = nil

	// This is used to flag client submitted transforms that we shouldn't send back
	ignoreTforms := []int{}

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
				return
			}
			h.log(leaplib.LeapDebug, fmt.Sprintf("Received %v command from client", msg.Command))
			switch msg.Command {
			case "submit":
				if ver, err := binder.SendTransform(msg.Transform, bindTOut); err == nil {
					ignoreTforms = append(ignoreTforms, ver)
					h.log(leaplib.LeapDebug, "Sending correction to client")
					websocket.JSON.Send(socket, LeapTextServerMessage{
						Type:    "correction",
						Version: &ver,
					})
				} else {
					h.log(leaplib.LeapError, fmt.Sprintf("Transform request failed %v", err))
					websocket.JSON.Send(socket, LeapTextServerMessage{
						Type:  "error",
						Error: fmt.Sprintf("submit error: %v", err),
					})
					return
				}
			case "ping":
				// Do nothing
			default:
				websocket.JSON.Send(socket, LeapTextServerMessage{
					Type:  "error",
					Error: "command not recognised",
				})
			}
		case tformsWrap, open := <-binder.TransformRcvChan:
			if !open {
				return
			}
			if len(tformsWrap) == 0 {
				break
			}

			fatal := false

			tforms := make([]leaplib.OTransform, len(tformsWrap))
			for i, tformWrap := range tformsWrap {
				if tform, ok := tformWrap.(leaplib.OTransform); ok {
					tforms[i] = tform
				} else {
					fatal = true
					h.log(leaplib.LeapError, fmt.Sprintf("Received unexpected type from RcvChan: %v", tformWrap))
					break
				}
			}

			if !fatal {
				skip := false
				for i, ignore := range ignoreTforms {
					if ignore == tforms[0].Version {
						skip = true
						ignoreTforms = append(ignoreTforms[:i], ignoreTforms[i+1:]...)
						break
					}
				}

				if !skip {
					h.log(leaplib.LeapDebug, fmt.Sprintf("Sending %v transforms to client", len(tforms)))
					websocket.JSON.Send(socket, LeapTextServerMessage{
						Type:       "transforms",
						Transforms: tforms,
					})
				} else {
					h.log(leaplib.LeapDebug, "Skipping clients own submitted transforms")
				}
			}
		case <-h.closeChan:
			return
		}
	}
}

/*--------------------------------------------------------------------------------------------------
 */
