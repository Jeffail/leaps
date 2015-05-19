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

/*
Package auth - Contains multiple solutions for introducing token based authentication for accessing,
creating and reading internal leaps documents.
*/
package auth

import (
	"errors"

	"github.com/jeffail/leaps/lib/register"
)

// Errors for the auth package.
var (
	ErrInvalidAuthType = errors.New("invalid token authenticator type")
)

/*
Authenticator - Implemented by types able to validate tokens for editing or creating documents.
This is abstracted in order to accommodate for multiple authentication strategies.
*/
type Authenticator interface {
	// AuthoriseCreate - Validate that a `create action` token corresponds to a particular user.
	AuthoriseCreate(token, userID string) bool

	// AuthoriseJoin - Validate that a `join action` token corresponds to a particular document.
	AuthoriseJoin(token, documentID string) bool

	// AuthoriseReadOnly - Validate that a `read only` token corresponds to a particular document.
	AuthoriseReadOnly(token, documentID string) bool

	// RegisterHandlers - Allow the Auth to register any API endpoints it needs.
	RegisterHandlers(register register.PubPrivEndpointRegister) error
}
