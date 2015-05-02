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

	"github.com/jeffail/leaps/lib"
	"github.com/jeffail/util/log"
	"github.com/jeffail/util/path"
	"golang.org/x/net/websocket"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
SSLConfig - Options for setting an SSL certificate
*/
type SSLConfig struct {
	Enabled         bool   `json:"enabled" yaml:"enabled"`
	CertificatePath string `json:"certificate_path" yaml:"certificate_path"`
	PrivateKeyPath  string `json:"private_key_path" yaml:"private_key_path"`
}

/*
NewSSLConfig - Creates a new SSLConfig object with default values
*/
func NewSSLConfig() SSLConfig {
	return SSLConfig{
		Enabled:         false,
		CertificatePath: "",
		PrivateKeyPath:  "",
	}
}

/*
HTTPBinderConfig - Options for individual binders (one for each socket connection)
*/
type HTTPBinderConfig struct {
	BindSendTimeout int `json:"bind_send_timeout_ms" yaml:"bind_send_timeout_ms"`
}

/*
HTTPServerConfig - Holds configuration options for the HTTPServer.
*/
type HTTPServerConfig struct {
	StaticPath     string               `json:"static_path" yaml:"static_path"`
	Path           string               `json:"socket_path" yaml:"socket_path"`
	Address        string               `json:"address" yaml:"address"`
	StaticFilePath string               `json:"www_dir" yaml:"www_dir"`
	Binder         HTTPBinderConfig     `json:"binder" yaml:"binder"`
	SSL            SSLConfig            `json:"ssl" yaml:"ssl"`
	HTTPAuth       AuthMiddlewareConfig `json:"basic_auth" yaml:"basic_auth"`
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
		SSL:      NewSSLConfig(),
		HTTPAuth: NewAuthMiddlewareConfig(),
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
LeapClientMessage - A structure that defines a message format to expect from clients. Commands can
be 'create' (init with new document) or 'find' (init with existing document).
*/
type LeapClientMessage struct {
	Command  string        `json:"command" yaml:"command"`
	Token    string        `json:"token" yaml:"token"`
	DocID    string        `json:"document_id,omitempty" yaml:"document_id,omitempty"`
	UserID   string        `json:"user_id,omitempty" yaml:"user_id,omitempty"`
	Document *lib.Document `json:"leap_document,omitempty" yaml:"leap_document,omitempty"`
}

/*
LeapServerMessage - A structure that defines a response message from the server to a client. Type
can be 'document' (init response) or 'error' (an error message to display to the client).
*/
type LeapServerMessage struct {
	Type     string        `json:"response_type" yaml:"response_type"`
	Document *lib.Document `json:"leap_document,omitempty" yaml:"leap_document,omitempty"`
	Version  *int          `json:"version,omitempty" yaml:"version,omitempty"`
	Error    string        `json:"error,omitempty" yaml:"error,omitempty"`
}

/*--------------------------------------------------------------------------------------------------
 */

// Errors for the HTTPServer type.
var (
	ErrInvalidSocketPath = errors.New("invalid config value for socket path")
	ErrInvalidStaticPath = errors.New("invalid config value for static path")
	ErrInvalidDocument   = errors.New("invalid document structure")
	ErrInvalidURLAddr    = errors.New("invalid config value for URL.Address")
	ErrInvalidSSLConfig  = errors.New("invalid config value for certificate path and/or private key path")
)

/*
HTTPServer - A construct designed to take a LeapLocator (a structure for finding and binding to
leap documents) and bind it to http clients.
*/
type HTTPServer struct {
	config    HTTPServerConfig
	logger    *log.Logger
	stats     *log.Stats
	auth      *AuthMiddleware
	locator   LeapLocator
	closeChan chan bool
}

/*
CreateHTTPServer - Create a new leaps HTTPServer.
*/
func CreateHTTPServer(
	locator LeapLocator,
	config HTTPServerConfig,
	logger *log.Logger,
	stats *log.Stats,
) (*HTTPServer, error) {
	auth, err := NewAuthMiddleware(config.HTTPAuth, logger, stats)
	if err != nil {
		return nil, err
	}
	httpServer := HTTPServer{
		config:    config,
		locator:   locator,
		logger:    logger.NewModule(":http"),
		stats:     stats,
		auth:      auth,
		closeChan: make(chan bool),
	}
	if len(httpServer.config.Path) == 0 {
		return nil, ErrInvalidSocketPath
	}
	http.Handle(
		httpServer.config.Path,
		httpServer.auth.WrapWSHandler(websocket.Handler(httpServer.websocketHandler)),
	)
	if len(httpServer.config.StaticFilePath) > 0 {
		if len(httpServer.config.StaticPath) == 0 {
			return nil, ErrInvalidStaticPath
		}
		// If the static file path is relative then we use the location of the binary to resolve it.
		if err := path.FromBinaryIfRelative(&httpServer.config.StaticFilePath); err != nil {
			return nil, fmt.Errorf("relative path for static files could not be resolved: %v", err)
		}
		http.Handle(httpServer.config.StaticPath,
			httpServer.auth.WrapHandler( // Auth wrap
				http.StripPrefix(httpServer.config.StaticPath, // File strip prefix wrap
					http.FileServer(http.Dir(httpServer.config.StaticFilePath))))) // File serve handler
	}
	return &httpServer, nil
}

/*--------------------------------------------------------------------------------------------------
 */

/*
websocketHandler - The method for creating fresh websocket clients.
*/
func (h *HTTPServer) websocketHandler(ws *websocket.Conn) {
	defer func() {
		if err := ws.Close(); err != nil {
			h.logger.Errorf("Failed to close socket: %v\n", err)
		}
		h.stats.Decr("http.open_websockets", 1)
	}()

	h.stats.Incr("http.websocket.opened", 1)
	h.stats.Incr("http.open_websockets", 1)

	select {
	case <-h.closeChan:
		websocket.JSON.Send(ws, LeapServerMessage{
			Type:  "error",
			Error: "target server node is closing",
		})
		return
	default:
	}

	h.logger.Infoln("Fresh client connected via websocket")

	handleInitError := func(err error) {
		h.logger.Infof("Client failed to init: %v\n", err)
		websocket.JSON.Send(ws, LeapServerMessage{
			Type:  "error",
			Error: fmt.Sprintf("socket initialization failed: %v", err),
		})
	}

	for {
		var clientMsg LeapClientMessage
		websocket.JSON.Receive(ws, &clientMsg)

		switch clientMsg.Command {
		case "create":
			if clientMsg.Document == nil {
				handleInitError(ErrInvalidDocument)
				return
			}
			h.logger.Infoln("Attempting to create document")
			if binder, err := h.locator.CreateDocument(
				clientMsg.Token, clientMsg.UserID, *clientMsg.Document); err == nil {
				h.logger.Infof("Client bound to document %v\n", binder.Document.ID)

				websocket.JSON.Send(ws, LeapServerMessage{
					Type:     "document",
					Document: &binder.Document,
					Version:  &binder.Version,
				})
				socketRouter := NewWebsocketServer(h.config.Binder, ws, binder, h.closeChan, h.logger, h.stats)
				socketRouter.Launch()
			} else {
				handleInitError(err)
			}
			return
		case "find":
			if len(clientMsg.DocID) <= 0 {
				handleInitError(ErrInvalidDocument)
				return
			}
			h.logger.Infof("Attempting to bind to document: %v\n", clientMsg.DocID)
			if binder, err := h.locator.FindDocument(clientMsg.Token, clientMsg.DocID); err == nil {
				h.logger.Infof("Client bound to document %v\n", binder.Document.ID)

				websocket.JSON.Send(ws, LeapServerMessage{
					Type:     "document",
					Document: &binder.Document,
					Version:  &binder.Version,
				})
				socketRouter := NewWebsocketServer(h.config.Binder, ws, binder, h.closeChan, h.logger, h.stats)
				socketRouter.Launch()
			} else {
				handleInitError(err)
			}
			return
		case "ping":
			// Ignore
		default:
			handleInitError(fmt.Errorf("first message must be init, client sent: %v", clientMsg.Command))
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
		return ErrInvalidURLAddr
	}
	if h.config.SSL.Enabled {
		if len(h.config.SSL.CertificatePath) == 0 || len(h.config.SSL.PrivateKeyPath) == 0 {
			return ErrInvalidSSLConfig
		}
		// If the static paths are relative then we use the location of the binary to resolve it.
		if err := path.FromBinaryIfRelative(&h.config.SSL.CertificatePath); err != nil {
			return fmt.Errorf("relative path for certificate could not be resolved: %v", err)
		}
		if err := path.FromBinaryIfRelative(&h.config.SSL.PrivateKeyPath); err != nil {
			return fmt.Errorf("relative path for private key could not be resolved: %v", err)
		}
	}
	h.logger.Infof("Listening for websockets at address: %v%v\n", h.config.Address, h.config.Path)
	if len(h.config.StaticPath) > 0 {
		h.logger.Infof("Serving static file requests at address: %v%v\n", h.config.Address, h.config.StaticPath)
	}
	var err error
	if h.config.SSL.Enabled {
		err = http.ListenAndServeTLS(
			h.config.Address,
			h.config.SSL.CertificatePath,
			h.config.SSL.PrivateKeyPath,
			nil,
		)
	} else {
		err = http.ListenAndServe(h.config.Address, nil)
	}
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
