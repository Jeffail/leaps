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
	"errors"
	"fmt"
	"net/http"

	"code.google.com/p/go.net/websocket"
	"github.com/jeffail/leaps/lib"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
HTTPBinderConfig - Options for individual binders (one for each socket connection)
*/
type HTTPBinderConfig struct {
	BindSendTimeout int `json:"bind_send_timeout_ms"`
}

/*
HTTPServerConfig - Holds configuration options for the HTTPServer.
*/
type HTTPServerConfig struct {
	StaticPath     string           `json:"static_path"`
	Path           string           `json:"socket_path"`
	Address        string           `json:"address"`
	StaticFilePath string           `json:"www_dir"`
	Binder         HTTPBinderConfig `json:"binder"`
}

/*
DefaultHTTPServerConfig - Returns a fully defined HTTPServer configuration with the default values
for each field.
*/
func DefaultHTTPServerConfig() HTTPServerConfig {
	return HTTPServerConfig{
		StaticPath:     "/leaps",
		Path:           "/leaps/socket",
		Address:        "localhost:8080",
		StaticFilePath: "",
		Binder: HTTPBinderConfig{
			BindSendTimeout: 10,
		},
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
LeapClientMessage - A structure that defines a message format to expect from clients. Commands can
be 'create' (init with new document) or 'find' (init with existing document).
*/
type LeapClientMessage struct {
	Command  string        `json:"command"`
	Token    string        `json:"token"`
	DocID    string        `json:"document_id,omitempty"`
	UserID   string        `json:"user_id,omitempty"`
	Document *lib.Document `json:"leap_document,omitempty"`
}

/*
LeapServerMessage - A structure that defines a response message from the server to a client. Type
can be 'document' (init response) or 'error' (an error message to display to the client).
*/
type LeapServerMessage struct {
	Type     string        `json:"response_type"`
	Document *lib.Document `json:"leap_document,omitempty"`
	Version  *int          `json:"version,omitempty"`
	Error    string        `json:"error,omitempty"`
}

/*--------------------------------------------------------------------------------------------------
 */

/*
HTTPServer - A construct designed to take a LeapLocator (a structure for finding and binding to
leap documents) and bind it to http clients.
*/
type HTTPServer struct {
	config    HTTPServerConfig
	logger    *lib.LeapsLogger
	locator   LeapLocator
	closeChan chan bool
}

/*
CreateHTTPServer - Create a new leaps HTTPServer, optionally registers to a custom http.ServeMux, or
set this to nil to use the default http mux (recommended).
*/
func CreateHTTPServer(locator LeapLocator, config HTTPServerConfig, mux *http.ServeMux) (*HTTPServer, error) {
	httpServer := HTTPServer{
		config:    config,
		locator:   locator,
		logger:    locator.GetLogger(),
		closeChan: make(chan bool),
	}
	if len(httpServer.config.Path) == 0 {
		return nil, errors.New("invalid config value for url.socket_path")
	}
	if mux != nil {
		mux.Handle(httpServer.config.Path, websocket.Handler(httpServer.websocketHandler))
	} else {
		http.Handle(httpServer.config.Path, websocket.Handler(httpServer.websocketHandler))
	}
	if len(httpServer.config.StaticFilePath) > 0 {
		if len(httpServer.config.StaticPath) == 0 {
			return nil, errors.New("invalid config value for url.static_path")
		}
		if mux != nil {
			mux.Handle(httpServer.config.StaticPath,
				http.StripPrefix(httpServer.config.StaticPath,
					http.FileServer(http.Dir(httpServer.config.StaticFilePath))))
		} else {
			http.Handle(httpServer.config.StaticPath,
				http.StripPrefix(httpServer.config.StaticPath,
					http.FileServer(http.Dir(httpServer.config.StaticFilePath))))
		}
	}
	return &httpServer, nil
}

/*--------------------------------------------------------------------------------------------------
 */

/*
log - Helper function for logging events, only actually logs when verbose logging is configured.
*/
func (h *HTTPServer) log(level int, message string) {
	h.logger.Log(level, "http", message)
}

/*
processInitMessage - Process an initial message from a client and, if the format is as expected,
return a binder that satisfies the request.
*/
func (h *HTTPServer) processInitMessage(clientMsg *LeapClientMessage) (*lib.BinderPortal, error) {
	switch clientMsg.Command {
	case "create":
		if clientMsg.Document != nil {
			return h.locator.NewDocument(clientMsg.Token, clientMsg.UserID, clientMsg.Document)
		}
		return nil, errors.New("create request must contain a valid document structure")
	case "find":
		if len(clientMsg.DocID) > 0 {
			h.log(lib.LeapInfo, fmt.Sprintf("Attempting to bind to document: %v", clientMsg.DocID))
			return h.locator.FindDocument(clientMsg.Token, clientMsg.DocID)
		}
		return nil, errors.New("find request must contain a valid document ID")
	case "ping":
		return nil, nil
	}
	return nil, fmt.Errorf("first message must be an initializer request, client sent: %v", clientMsg.Command)
}

/*
websocketHandler - The method for creating fresh websocket clients.
*/
func (h *HTTPServer) websocketHandler(ws *websocket.Conn) {
	defer func() {
		if err := ws.Close(); err != nil {
			h.log(lib.LeapError, fmt.Sprintf("Failed to close socket: %v", err))
		}
	}()

	select {
	case <-h.closeChan:
		websocket.JSON.Send(ws, LeapServerMessage{
			Type:  "error",
			Error: "target server node is closing",
		})
		return
	default:
	}

	h.log(lib.LeapInfo, "Fresh client connected via websocket")

	for {
		var launchCmd LeapClientMessage
		websocket.JSON.Receive(ws, &launchCmd)

		if binder, err := h.processInitMessage(&launchCmd); err == nil && binder != nil {
			h.log(lib.LeapInfo, fmt.Sprintf("Client bound to document %v", binder.Document.ID))

			websocket.JSON.Send(ws, LeapServerMessage{
				Type:     "document",
				Document: binder.Document,
				Version:  &binder.Version,
			})

			// TODO: Generic
			hbind := HTTPTextModel{
				config:    h.config.Binder,
				logger:    h.logger,
				closeChan: h.closeChan,
			}
			LaunchWebsocketTextModel(&hbind, ws, binder)
			return
		} else if err != nil {
			h.log(lib.LeapInfo, fmt.Sprintf("Client failed to init: %v", err))
			websocket.JSON.Send(ws, LeapServerMessage{
				Type:  "error",
				Error: fmt.Sprintf("socket initialization failed: %v", err),
			})
			return
		}
	}
}

/*
Listen - Bind to the http endpoint as per configured address, and begin serving requests. This is
simply a helper function that calls http.ListenAndServe
*/
func (h *HTTPServer) Listen() error {
	if len(h.config.Address) == 0 {
		return errors.New("invalid config value for URL.Address")
	}
	h.log(lib.LeapInfo, fmt.Sprintf("Listening for websockets at address: %v",
		fmt.Sprintf("%v%v", h.config.Address, h.config.Path)))
	if len(h.config.StaticPath) > 0 {
		h.log(lib.LeapInfo, fmt.Sprintf("Serving static file requests at address: %v",
			fmt.Sprintf("%v%v", h.config.Address, h.config.StaticPath)))
	}
	err := http.ListenAndServe(h.config.Address, nil)
	return err
}

/*
Stop - Stop serving web requests and close the HTTPServer.
*/
func (h *HTTPServer) Stop() {
	close(h.closeChan)
}

/*--------------------------------------------------------------------------------------------------
 */
