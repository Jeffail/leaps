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
	"crypto/sha1"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/websocket"

	"github.com/jeffail/util/log"
	"github.com/jeffail/util/path"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
AuthMiddlewareConfig - Holds configuration options for the AuthMiddleware
*/
type AuthMiddlewareConfig struct {
	Enabled        bool   `json:"enabled" yaml:"enabled"`
	PasswdFilePath string `json:"htpasswd_path" yaml:"htpasswd_path"`
}

/*
NewAuthMiddlewareConfig - Returns an AuthMiddleware configuration with the default values
*/
func NewAuthMiddlewareConfig() AuthMiddlewareConfig {
	return AuthMiddlewareConfig{
		Enabled:        false,
		PasswdFilePath: "",
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
AuthMiddleware - A construct designed to take a LeapLocator (a structure for finding and binding to
leap documents) and bind it to http clients.
*/
type AuthMiddleware struct {
	config   AuthMiddlewareConfig
	accounts map[string]string
	logger   *log.Logger
	stats    *log.Stats
}

/*
NewAuthMiddleware - Create a new leaps AuthMiddleware.
*/
func NewAuthMiddleware(
	config AuthMiddlewareConfig,
	logger *log.Logger,
	stats *log.Stats,
) (*AuthMiddleware, error) {
	auth := AuthMiddleware{
		config:   config,
		accounts: map[string]string{},
		logger:   logger.NewModule("[basic_auth]"),
		stats:    stats,
	}
	if config.Enabled {
		if 0 == len(config.PasswdFilePath) {
			return nil, errors.New("HTTP Auth requires a htpasswd file path in the configuration")
		}
		if err := auth.accountsFromFile(config.PasswdFilePath); err != nil {
			return nil, fmt.Errorf("htpasswd file read error: %v", err)
		}
	}
	return &auth, nil
}

/*--------------------------------------------------------------------------------------------------
 */

/*
WrapHandler - Wrap an http request Handler with the AuthMiddleware authentication.
*/
func (a *AuthMiddleware) WrapHandler(handler http.Handler) http.HandlerFunc {
	if !a.config.Enabled {
		return func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.authenticateRequest(r) {
			a.requestAuth(w, r)
		} else {
			handler.ServeHTTP(w, r)
		}
	}
}

/*
WrapHandlerFunc - Wrap an http request HandlerFunc with the AuthMiddleware authentication.
*/
func (a *AuthMiddleware) WrapHandlerFunc(handler http.HandlerFunc) http.HandlerFunc {
	if !a.config.Enabled {
		return handler
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.authenticateRequest(r) {
			a.requestAuth(w, r)
		} else {
			handler(w, r)
		}
	}
}

/*
WrapWSHandler - Wrap a websocket http request handler with the AuthMiddleware authentication.
*/
func (a *AuthMiddleware) WrapWSHandler(handler websocket.Handler) websocket.Handler {
	if !a.config.Enabled {
		return handler
	}
	return func(w *websocket.Conn) {
		if !a.authenticateRequest(w.Request()) {
			w.Close()
		} else {
			handler(w)
		}
	}
}

/*
requestAuth - An HTTP handler that sends back a 401 (request for authentication credentials).
*/
func (a *AuthMiddleware) requestAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", `Basic realm="leaps"`)
	w.WriteHeader(401)
	w.Write([]byte("401 Unauthorized\n"))
}

/*--------------------------------------------------------------------------------------------------
 */

/*
accountsFromFile - Extract a map of username and password hashes from a htpasswd file. MD5 hashes
are not supported, use SHA1 instead.
*/
func (a *AuthMiddleware) accountsFromFile(filePath string) error {
	// If the file path is relative then we use the location of the binary to resolve it.
	if err := path.FromBinaryIfRelative(&filePath); err != nil {
		return err
	}
	r, err := os.Open(filePath)
	if err != nil {
		return err
	}
	CSVReader := csv.NewReader(r)
	CSVReader.Comma = ':'
	CSVReader.Comment = '#'
	CSVReader.FieldsPerRecord = 2

	userHashes, err := CSVReader.ReadAll()
	if err != nil {
		return err
	}
	a.accounts = map[string]string{}
	for _, userHash := range userHashes {
		a.accounts[userHash[0]] = userHash[1]
	}
	return nil
}

/*
authenticateRequest - Attempts to authenticate a request using basic HTTP auth. Returns true or
false, false indicates a failed authentication.
*/
func (a *AuthMiddleware) authenticateRequest(r *http.Request) bool {

	// Expected header format: AUTH_TYPE<SPACE>B64_ENCODED_CREDENTIALS
	authParts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if 2 != len(authParts) {
		a.logger.Warnf("Rejecting due to auth header part count: %v != %v\n", len(authParts), 2)
		return false
	}
	if "Basic" != authParts[0] {
		a.logger.Warnf("Rejecting due to auth type: %v != Basic\n", authParts[0])
		return false
	}
	b64Credentials := authParts[1]
	decodedCredentials, err := base64.StdEncoding.DecodeString(b64Credentials)
	if err != nil {
		a.logger.Errorf("Failed to decode request auth credentials: %v\n", err)
		return false
	}

	// Expected credentials format: USERNAME:PASSWORD
	credentials := strings.SplitN(string(decodedCredentials), ":", 2)
	if 2 != len(credentials) {
		a.logger.Warnf("Rejecting due to credential count: %v != %v\n", len(credentials), 2)
		return false
	}
	passHash, ok := a.accounts[credentials[0]]
	if !ok {
		a.logger.Warnf("Rejecting due to non-existant account: %v\n", credentials[0])
		return false
	}
	if strings.HasPrefix(passHash, "{SHA}") {
		shaGen := sha1.New()
		shaGen.Write([]byte(credentials[1]))
		if passHash[5:] != base64.StdEncoding.EncodeToString(shaGen.Sum(nil)) {
			a.logger.Warnf("Rejecting due to wrong password for account: %v\n", credentials[0])
			return false
		}
	} // Only support SHA1 right now.
	return true
}

/*--------------------------------------------------------------------------------------------------
 */
