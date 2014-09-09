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
)

/*--------------------------------------------------------------------------------------------------
 */

/*
TokenAuthenticatorConfig - Holds generic configuration options for a token based authentication
solution.
*/
type TokenAuthenticatorConfig struct {
	Type string `json:"type"`
}

/*
DefaultTokenAuthenticatorConfig - Returns a default generic configuration.
*/
func DefaultTokenAuthenticatorConfig() TokenAuthenticatorConfig {
	return TokenAuthenticatorConfig{
		Type: "none",
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
TokenAuthenticator - Implemented by types able to validate tokens for editing or creating documents.
This is abstracted in order to accommodate for multiple authentication strategies.
*/
type TokenAuthenticator interface {
	AuthoriseCreate(token string) bool
	AuthoriseJoin(token, documentID string) bool
}

/*--------------------------------------------------------------------------------------------------
 */

/*
TokenAuthenticatorFactory - Returns a document store object based on a configuration object.
*/
func TokenAuthenticatorFactory(config TokenAuthenticatorConfig) (TokenAuthenticator, error) {
	switch config.Type {
	case "none":
		return GetAnarchy(config), nil
	}
	return nil, errors.New("configuration provided invalid token authenticator type")
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Anarchy - Most basic implementation of TokenAuthenticator, everyone has access to everything.
*/
type Anarchy struct {
}

/*
AuthoriseCreate - Always returns true, because anarchy.
*/
func (a *Anarchy) AuthoriseCreate(_ string) bool {
	return true
}

/*
AuthoriseJoin - Always returns true, because anarchy.
*/
func (a *Anarchy) AuthoriseJoin(_, _ string) bool {
	return true
}

/*
GetAnarchy - Get yourself a little taste of sweet, juicy anarchy.
*/
func GetAnarchy(config TokenAuthenticatorConfig) TokenAuthenticator {
	return &Anarchy{}
}

/*--------------------------------------------------------------------------------------------------
 */
