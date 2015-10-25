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

package auth

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/jeffail/leaps/lib/register"
	"github.com/jeffail/leaps/lib/util"
	"github.com/jeffail/util/log"
	"github.com/jeffail/util/metrics"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
HTTPConfig - A config object for the HTTP API authentication object.
*/
type HTTPConfig struct {
	Path         string `json:"path" yaml:"path"`
	ExpiryPeriod int64  `json:"expiry_period_s" yaml:"expiry_period_s"`
}

/*
NewHTTPConfig - Returns a default config object for a HTTP.
*/
func NewHTTPConfig() HTTPConfig {
	return HTTPConfig{
		Path:         "auth",
		ExpiryPeriod: 60,
	}
}

/*--------------------------------------------------------------------------------------------------
 */

func (h *HTTP) createGenerateTokenHandler(tokens tokensMap) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST endpoint only", http.StatusMethodNotAllowed)
			return
		}

		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			h.logger.Errorf("Failed to read request body: %v\n", err)
			http.Error(w, "Bad request: could not read body", http.StatusBadRequest)
			return
		}

		var bodyObj struct {
			Key string `json:"key_value"`
		}
		if err = json.Unmarshal(bytes, &bodyObj); err != nil {
			h.logger.Errorf("Failed to parse request body: %v\n", err)
			http.Error(w, "Bad request: could not parse body", http.StatusBadRequest)
			return
		}

		if 0 == len(bodyObj.Key) {
			h.logger.Errorln("User ID not found in request body")
			http.Error(w, "Bad request: no user id found", http.StatusBadRequest)
			return
		}

		token := util.GenerateStampedUUID()

		h.mutex.Lock()

		tokens[token] = tokenMapValue{
			value:   bodyObj.Key,
			expires: time.Now().Add(time.Second * time.Duration(h.config.HTTPConfig.ExpiryPeriod)),
		}
		h.mutex.Unlock()

		resBytes, err := json.Marshal(struct {
			Token string `json:"token"`
		}{
			Token: token,
		})
		if err != nil {
			h.logger.Errorf("Failed to generate JSON response: %v\n", err)
			http.Error(w, "Failed to generate response", http.StatusInternalServerError)
			return
		}

		w.Write(resBytes)
		w.Header().Add("Content-Type", "application/json")

		h.clearExpiredTokens(tokens)
	}
}

/*--------------------------------------------------------------------------------------------------
 */

type tokenMapValue struct {
	value   string
	expires time.Time
}

type tokensMap map[string]tokenMapValue

/*
HTTP - Uses the admin HTTP server to expose an endpoint for submitting authentication
tokens.
*/
type HTTP struct {
	logger *log.Logger
	stats  metrics.Aggregator
	config Config

	// Lock for token reading/writing
	mutex sync.RWMutex

	// Stored tokens for various actions
	tokensCreate   tokensMap
	tokensJoin     tokensMap
	tokensReadOnly tokensMap

	// HTTP handlers for various actions
	createHandler   http.HandlerFunc
	joinHandler     http.HandlerFunc
	readOnlyHandler http.HandlerFunc
}

/*
NewHTTP - Creates an HTTP using the provided configuration.
*/
func NewHTTP(config Config, logger *log.Logger, stats metrics.Aggregator) *HTTP {
	authorizer := HTTP{
		logger: logger.NewModule(":http_auth"),
		stats:  stats,
		config: config,
		mutex:  sync.RWMutex{},

		tokensCreate:   tokensMap{},
		tokensJoin:     tokensMap{},
		tokensReadOnly: tokensMap{},
	}

	authorizer.createHandler = authorizer.createGenerateTokenHandler(authorizer.tokensCreate)
	authorizer.joinHandler = authorizer.createGenerateTokenHandler(authorizer.tokensJoin)
	authorizer.readOnlyHandler = authorizer.createGenerateTokenHandler(authorizer.tokensReadOnly)

	return &authorizer
}

/*--------------------------------------------------------------------------------------------------
 */

/*
clearExpiredTokens - Purges our expired tokens from a map.
*/
func (h *HTTP) clearExpiredTokens(tokens tokensMap) {
	expiredTokens := []string{}

	h.mutex.RLock()
	for token, val := range tokens {
		if val.expires.Before(time.Now()) {
			expiredTokens = append(expiredTokens, token)
		}
	}
	h.mutex.RUnlock()

	if len(expiredTokens) > 0 {
		h.mutex.Lock()
		for _, token := range expiredTokens {
			delete(tokens, token)
		}
		h.mutex.Unlock()
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
AuthoriseCreate - Checks whether a specific token has been generated for a user through the HTTP
authentication endpoint for creating a new document.
*/
func (h *HTTP) AuthoriseCreate(token, userID string) bool {
	if !h.config.AllowCreate {
		return false
	}

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if tObj, ok := h.tokensCreate[token]; ok {
		if tObj.value == userID {
			delete(h.tokensCreate, token)
			return true
		}
	}
	return false
}

/*
AuthoriseJoin - Checks whether a specific token has been generated for a document through the HTTP
authentication endpoint for joining that aforementioned document.
*/
func (h *HTTP) AuthoriseJoin(token, documentID string) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if tObj, ok := h.tokensJoin[token]; ok {
		if tObj.value == documentID {
			delete(h.tokensJoin, token)
			return true
		}
	}
	return false
}

/*
AuthoriseReadOnly - Checks whether a specific token has been generated for a document through the HTTP
authentication endpoint for joining that aforementioned document in read only mode.
*/
func (h *HTTP) AuthoriseReadOnly(token, documentID string) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if tObj, ok := h.tokensReadOnly[token]; ok {
		if tObj.value == documentID {
			delete(h.tokensReadOnly, token)
			return true
		}
	}
	return false
}

/*
RegisterHandlers - Register endpoints for adding new auth tokens.
*/
func (h *HTTP) RegisterHandlers(register register.PubPrivEndpointRegister) error {
	if err := register.RegisterPrivate(
		path.Join(h.config.HTTPConfig.Path, "create"),
		`Generate an authentication token for creating a new document, POST: {"key_value":"<user_id>"}`,
		h.createHandler,
	); err != nil {
		return err
	}
	if err := register.RegisterPrivate(
		path.Join(h.config.HTTPConfig.Path, "read"),
		`Generate an authentication token for joining an existing document in read only mode, POST: {"key_value":"<document_id>"}`,
		h.readOnlyHandler,
	); err != nil {
		return err
	}
	return register.RegisterPrivate(
		path.Join(h.config.HTTPConfig.Path, "join"),
		`Generate an authentication token for joining an existing document, POST: {"key_value":"<document_id>"}`,
		h.joinHandler,
	)
}

/*--------------------------------------------------------------------------------------------------
 */
