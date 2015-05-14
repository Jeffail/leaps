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

package lib

import (
	"errors"

	"github.com/jeffail/util/log"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
TokenAuthenticatorConfig - Holds generic configuration options for a token based authentication
solution.
*/
type TokenAuthenticatorConfig struct {
	Type        string                   `json:"type" yaml:"type"`
	AllowCreate bool                     `json:"allow_creation" yaml:"allow_creation"`
	RedisConfig RedisAuthenticatorConfig `json:"redis_config" yaml:"redis_config"`
	FileConfig  FileAuthenticatorConfig  `json:"file_config" yaml:"file_config"`
	HTTPConfig  HTTPAuthenticatorConfig  `json:"http_config" yaml:"http_config"`
}

/*
DefaultTokenAuthenticatorConfig - Returns a default generic configuration.
*/
func DefaultTokenAuthenticatorConfig() TokenAuthenticatorConfig {
	return TokenAuthenticatorConfig{
		Type:        "none",
		AllowCreate: true,
		RedisConfig: DefaultRedisAuthenticatorConfig(),
		FileConfig:  DefaultFileAuthenticatorConfig(),
		HTTPConfig:  NewHTTPAuthenticatorConfig(),
	}
}

/*--------------------------------------------------------------------------------------------------
 */

// Errors for the TokenAuthentication type.
var (
	ErrInvalidAuthenticatorType = errors.New("invalid token authenticator type")
)

/*
TokenAuthenticator - Implemented by types able to validate tokens for editing or creating documents.
This is abstracted in order to accommodate for multiple authentication strategies.
*/
type TokenAuthenticator interface {
	// AuthoriseCreate - Validate that a `create action` token corresponds to a particular user.
	AuthoriseCreate(token, userID string) bool

	// AuthoriseJoin - Validate that a `join action` token corresponds to a particular document.
	AuthoriseJoin(token, documentID string) bool

	// RegisterHandlers - Allow the TokenAuthenticator to register any API endpoints it needs.
	RegisterHandlers(register PubPrivEndpointRegister) error
}

/*--------------------------------------------------------------------------------------------------
 */

/*
TokenAuthenticatorFactory - Returns a document store object based on a configuration object.
*/
func TokenAuthenticatorFactory(
	config TokenAuthenticatorConfig, logger *log.Logger, stats *log.Stats,
) (TokenAuthenticator, error) {
	switch config.Type {
	case "none":
		return GetAnarchy(config), nil
	case "file":
		return NewFileAuthenticator(config, logger), nil
	case "redis":
		return NewRedisAuthenticator(config, logger), nil
	case "http":
		return NewHTTPAuthenticator(config, logger, stats), nil
	}
	return nil, ErrInvalidAuthenticatorType
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Anarchy - Most basic implementation of TokenAuthenticator, everyone has access to everything.
*/
type Anarchy struct {
	config TokenAuthenticatorConfig
}

/*
AuthoriseCreate - Always returns true, because anarchy.
*/
func (a *Anarchy) AuthoriseCreate(_, _ string) bool {
	if !a.config.AllowCreate {
		return false
	}
	return true
}

/*
AuthoriseJoin - Always returns true, because anarchy.
*/
func (a *Anarchy) AuthoriseJoin(_, _ string) bool {
	return true
}

/*
RegisterHandlers - Nothing to register.
*/
func (a *Anarchy) RegisterHandlers(PubPrivEndpointRegister) error {
	return nil
}

/*
GetAnarchy - Get yourself a little taste of sweet, juicy anarchy.
*/
func GetAnarchy(config TokenAuthenticatorConfig) TokenAuthenticator {
	return &Anarchy{config}
}

/*--------------------------------------------------------------------------------------------------
 */
