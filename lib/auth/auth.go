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
	"github.com/jeffail/leaps/lib/register"
	"github.com/jeffail/util/log"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
Config - Holds generic configuration options for a token based authentication solution.
*/
type Config struct {
	Type        string      `json:"type" yaml:"type"`
	AllowCreate bool        `json:"allow_creation" yaml:"allow_creation"`
	RedisConfig RedisConfig `json:"redis_config" yaml:"redis_config"`
	FileConfig  FileConfig  `json:"file_config" yaml:"file_config"`
	HTTPConfig  HTTPConfig  `json:"http_config" yaml:"http_config"`
}

/*
NewConfig - Returns a default generic configuration.
*/
func NewConfig() Config {
	return Config{
		Type:        "none",
		AllowCreate: true,
		RedisConfig: NewRedisConfig(),
		FileConfig:  NewFileConfig(),
		HTTPConfig:  NewHTTPConfig(),
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Factory - Returns a document store object based on a configuration object.
*/
func Factory(
	config Config, logger *log.Logger, stats *log.Stats,
) (Authenticator, error) {
	switch config.Type {
	case "none":
		return GetAnarchy(config), nil
	case "file":
		return NewFile(config, logger), nil
	case "redis":
		return NewRedis(config, logger), nil
	case "http":
		return NewHTTP(config, logger, stats), nil
	}
	return nil, ErrInvalidAuthType
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Anarchy - Most basic implementation of Authenticator, everyone has access to everything.
*/
type Anarchy struct {
	config Config
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
func (a *Anarchy) RegisterHandlers(register.PubPrivEndpointRegister) error {
	return nil
}

/*
GetAnarchy - Get yourself a little taste of sweet, juicy anarchy.
*/
func GetAnarchy(config Config) Authenticator {
	return &Anarchy{config}
}

/*--------------------------------------------------------------------------------------------------
 */
