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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"time"

	"github.com/jeffail/util/log"
	binpath "github.com/jeffail/util/path"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
InternalServerConfig - Holds configuration options for the InternalServer.
*/
type InternalServerConfig struct {
	Path           string               `json:"path" yaml:"path"`
	Address        string               `json:"address" yaml:"address"`
	StaticFilePath string               `json:"www_dir" yaml:"www_dir"`
	SSL            SSLConfig            `json:"ssl" yaml:"ssl"`
	HTTPAuth       AuthMiddlewareConfig `json:"basic_auth" yaml:"basic_auth"`
	RequestTimeout int                  `json:"request_timeout_s" yaml:"request_timeout_s"`
}

/*
NewInternalServerConfig - Returns a fully defined InternalServer configuration with the default
values for each field.
*/
func NewInternalServerConfig() InternalServerConfig {
	return InternalServerConfig{
		Path:           "/admin",
		Address:        "",
		StaticFilePath: "",
		SSL:            NewSSLConfig(),
		HTTPAuth:       NewAuthMiddlewareConfig(),
		RequestTimeout: 10,
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
InternalServer - Provides a HTTP API for performing administrative actions and queries with the
leaps service. This server is intended to be inaccessible to outside users of the service.
*/
type InternalServer struct {
	config       InternalServerConfig
	logger       *log.Logger
	stats        *log.Stats
	auth         *AuthMiddleware
	mux          *http.ServeMux
	apiEndpoints []struct{ endpoint, desc string }
	admin        LeapAdmin
}

/*
NewInternalServer - Create a new leaps InternalServer.
*/
func NewInternalServer(
	admin LeapAdmin,
	config InternalServerConfig,
	logger *log.Logger,
	stats *log.Stats,
) (*InternalServer, error) {
	auth, err := NewAuthMiddleware(config.HTTPAuth, logger, stats)
	if err != nil {
		return nil, err
	}
	httpServer := InternalServer{
		config: config,
		admin:  admin,
		logger: logger.NewModule(":http_admin"),
		stats:  stats,
		mux:    http.NewServeMux(),
		auth:   auth,
	}

	// Register handling for static files
	if len(httpServer.config.StaticFilePath) > 0 {
		if len(httpServer.config.Path) == 0 {
			return nil, ErrInvalidStaticPath
		}
		// If the static file path is relative then we use the location of the binary to resolve it.
		if err := binpath.FromBinaryIfRelative(&httpServer.config.StaticFilePath); err != nil {
			return nil, fmt.Errorf("relative path for static files could not be resolved: %v", err)
		}
		httpServer.mux.Handle(httpServer.config.Path,
			httpServer.auth.WrapHandler( // Auth wrap
				http.StripPrefix(httpServer.config.Path, // File strip prefix wrap
					http.FileServer(http.Dir(httpServer.config.StaticFilePath))))) // File serve handler
	}

	httpServer.registerEndpoints()

	return &httpServer, nil
}

/*--------------------------------------------------------------------------------------------------
 */

func (i *InternalServer) registerEndpoints() {
	// Register /endpoints endpoint for printing endpoints
	i.Register("/endpoints", "<GET> the available endpoints of this leaps API",
		func(w http.ResponseWriter, r *http.Request) {
			for _, epoint := range i.apiEndpoints {
				fmt.Fprintf(w, "%v: %v\n", epoint.endpoint, epoint.desc)
			}
			w.Header().Add("Content-Type", "text/plain")
		})

	// Register /kick_user endpoint for kicking users from documents
	i.Register("/kick_user", `<POST> Kick a user from a document {"user_id":"<id>","doc_id":"<id>"}`,
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				i.stats.Incr("http_admin.kick_user.error", 1)
				i.logger.Warnf("/kick_user: Wrong method %v\n", r.Method)
				http.Error(w, "Wrong method", http.StatusMethodNotAllowed)
				return
			}

			bodyBytes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				i.stats.Incr("http_admin.kick_user.error", 1)
				i.logger.Errorf("/kick_user: %v\n", err)
				http.Error(w, "Bad data", http.StatusBadRequest)
				return
			}

			dataObj := struct {
				UserID string `json:"user_id"`
				DocID  string `json:"doc_id"`
			}{}
			if err := json.Unmarshal(bodyBytes, &dataObj); err != nil {
				i.stats.Incr("http_admin.kick_user.error", 1)
				i.logger.Errorf("/kick_user: %v\n", err)
				http.Error(w, "Bad data", http.StatusBadRequest)
				return
			}

			if err := i.admin.KickUser(
				dataObj.DocID,
				dataObj.UserID,
				time.Second*time.Duration(i.config.RequestTimeout),
			); err != nil {
				i.stats.Incr("http_admin.kick_user.error", 1)
				i.logger.Errorf("/kick_user: %v\n", err)
				http.Error(w, "Error kicking user", http.StatusInternalServerError)
				return
			}

			i.stats.Incr("http_admin.kick_user.success", 1)
			i.logger.Infof("/kick_user: Kicked user %v from %v\n", dataObj.UserID, dataObj.DocID)

			fmt.Fprintf(w, "Success")
		})

	// Register /get_users endpoint for listing users connected to all open documents
	i.Register(
		"/get_users",
		`<GET> Get a list of all connected users {"<document_id1>":["<id1>","<id2>"],"<document_id2":["<id3>"]}`,
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				i.stats.Incr("http_admin.get_users.error", 1)
				i.logger.Warnf("/get_users: Wrong method %v\n", r.Method)
				http.Error(w, "Wrong method", http.StatusMethodNotAllowed)
				return
			}

			resultObj, err := i.admin.GetUsers(time.Second * time.Duration(i.config.RequestTimeout))
			if err != nil {
				i.stats.Incr("http_admin.get_users.error", 1)
				i.logger.Errorf("/get_users: %v\n", err)
				http.Error(w, "Error collecting users", http.StatusInternalServerError)
				return
			}

			resultBytes, err := json.Marshal(resultObj)
			if err != nil {
				i.stats.Incr("http_admin.get_users.error", 1)
				i.logger.Errorf("/get_users: %v\n", err)
				http.Error(w, "Error collecting users", http.StatusInternalServerError)
				return
			}

			i.stats.Incr("http_admin.get_users.success", 1)
			i.logger.Debugf("/get_users: sending users for %v documents\n", len(resultObj))

			w.Header().Add("Content-Type", "application/json")
			w.Write(resultBytes)
		})
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Register - Register your handler func to an endpoint of the internal admin API.
*/
func (i *InternalServer) Register(endpoint, description string, handler http.HandlerFunc) {
	fullPath := path.Join(i.config.Path, endpoint)
	i.apiEndpoints = append(i.apiEndpoints, struct{ endpoint, desc string }{
		fullPath,
		description,
	})
	i.mux.HandleFunc(fullPath, handler)
}

/*
Listen - Bind to the http endpoint as per configured address, and begin serving requests. This is
simply a helper function that calls http.ListenAndServe
*/
func (i *InternalServer) Listen() error {
	if len(i.config.Address) == 0 {
		return ErrInvalidURLAddr
	}
	if i.config.SSL.Enabled {
		if len(i.config.SSL.CertificatePath) == 0 || len(i.config.SSL.PrivateKeyPath) == 0 {
			return ErrInvalidSSLConfig
		}
		// If the static paths are relative then we use the location of the binary to resolve it.
		if err := binpath.FromBinaryIfRelative(&i.config.SSL.CertificatePath); err != nil {
			return fmt.Errorf("relative path for certificate could not be resolved: %v", err)
		}
		if err := binpath.FromBinaryIfRelative(&i.config.SSL.PrivateKeyPath); err != nil {
			return fmt.Errorf("relative path for private key could not be resolved: %v", err)
		}
	}
	i.logger.Infof("Serving internal admin requests at address: %v%v\n", i.config.Address, i.config.Path)
	var err error
	if i.config.SSL.Enabled {
		err = http.ListenAndServeTLS(
			i.config.Address,
			i.config.SSL.CertificatePath,
			i.config.SSL.PrivateKeyPath,
			i.mux,
		)
	} else {
		err = http.ListenAndServe(i.config.Address, i.mux)
	}
	return err
}

/*--------------------------------------------------------------------------------------------------
 */
