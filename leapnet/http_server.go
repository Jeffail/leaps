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
	"errors"
	"fmt"
	"github.com/jeffail/leaps/leaplib"
	"net/http"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
URLConfig - Holds configuration options for the HTTPServer URL.
*/
type URLConfig struct {
	Path    string `json:"path"`
	Address string `json:"address"`
}

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
	URL    URLConfig        `json:"url"`
	Binder HTTPBinderConfig `json:"binder"`
}

/*
DefaultHTTPServerConfig - Returns a fully defined HTTPServer configuration with the default values
for each field.
*/
func DefaultHTTPServerConfig() HTTPServerConfig {
	return HTTPServerConfig{
		URL: URLConfig{
			Path:    "/leapsocket",
			Address: ":8080",
		},
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
	Command  string            `json:"command"`
	ID       string            `json:"document_id,omitempty"`
	Document *leaplib.Document `json:"leap_document,omitempty"`
}

/*
LeapServerMessage - A structure that defines a response message from the server to a client. Type
can be 'document' (init response) or 'error' (an error message to display to the client).
*/
type LeapServerMessage struct {
	Type     string            `json:"response_type"`
	Document *leaplib.Document `json:"leap_document,omitempty"`
	Version  *int              `json:"version,omitempty"`
	Error    string            `json:"error,omitempty"`
}

/*--------------------------------------------------------------------------------------------------
 */

/*
HTTPServer - A construct designed to take a LeapLocator (a structure for finding and binding to
leap documents) and bind it to http clients.
*/
type HTTPServer struct {
	config    HTTPServerConfig
	logger    *leaplib.LeapsLogger
	locator   LeapLocator
	closeChan chan bool
}

/*
CreateHTTPServer - Create a new leaps HTTPServer.
*/
func CreateHTTPServer(locator LeapLocator, config HTTPServerConfig) (*HTTPServer, error) {
	httpServer := HTTPServer{
		config:    config,
		locator:   locator,
		logger:    locator.GetLogger(),
		closeChan: make(chan bool),
	}
	if len(httpServer.config.URL.Path) == 0 {
		return nil, errors.New("invalid config value for URL.Path")
	}
	http.Handle(httpServer.config.URL.Path, websocket.Handler(httpServer.websocketHandler))
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
func (h *HTTPServer) processInitMessage(clientMsg *LeapClientMessage) (*leaplib.BinderPortal, error) {
	switch clientMsg.Command {
	case "create":
		if clientMsg.Document != nil {
			return h.locator.NewDocument(clientMsg.Document)
		}
		return nil, errors.New("create request must contain a valid document structure")
	case "find":
		if len(clientMsg.ID) > 0 {
			h.log(leaplib.LeapInfo, fmt.Sprintf("Attempting to bind to document: %v", clientMsg.ID))
			return h.locator.FindDocument(clientMsg.ID)
		}
		return nil, errors.New("find request must contain a valid document ID")
	}
	return nil, fmt.Errorf("first message must be an initializer request, client sent: %v", clientMsg.Command)
}

/*
websocketHandler - The method for creating fresh websocket clients.
*/
func (h *HTTPServer) websocketHandler(ws *websocket.Conn) {
	select {
	case <-h.closeChan:
		websocket.JSON.Send(ws, LeapServerMessage{
			Type:  "error",
			Error: "target server node is closing",
		})
		return
	default:
	}

	h.log(leaplib.LeapInfo, "Fresh client connected via websocket")

	var launchCmd LeapClientMessage
	websocket.JSON.Receive(ws, &launchCmd)

	if binder, err := h.processInitMessage(&launchCmd); err == nil {
		h.log(leaplib.LeapInfo, fmt.Sprintf("Client bound to document %v", binder.Document.ID))

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
	} else {
		h.log(leaplib.LeapInfo, fmt.Sprintf("Client failed to init: %v", err))
		websocket.JSON.Send(ws, LeapServerMessage{
			Type:  "error",
			Error: fmt.Sprintf("socket initialization failed: %v", err),
		})
	}
}

/*
Listen - Bind to the http endpoint and begin serving requests.
*/
func (h *HTTPServer) Listen() error {
	if len(h.config.URL.Address) == 0 {
		return errors.New("invalid config value for URL.Address")
	}
	h.log(leaplib.LeapInfo, fmt.Sprintf("Listening at address: %v", h.config.URL.Address))
	err := http.ListenAndServe(h.config.URL.Address, nil)
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
