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

package acl

import (
	"errors"
)

// Errors for the auth package.
var (
	ErrInvalidAuthType = errors.New("invalid token authenticator type")
)

// AccessLevel - A unit of access to a particular document ID.
type AccessLevel int

// Units of AccessLevel for expressing a users access to a document.
const (
	NoAccess AccessLevel = iota
	ReadAccess
	EditAccess
	CreateAccess
)

/*
Authenticator - Implemented by types able to validate tokens for editing or creating documents.
This is abstracted in order to accommodate for multiple authentication strategies.

Authenticate is the single call for all access level queries. The userID and token strings are used
to identify a specific user and session token respectively, it is up to the implementation to make
use of these fields or disregard them entirely.

The documentID field is used to a identify a specific document for accessing. If this field is
populated then the authenticator is expected to check the access level the user has for that
specific document, if the document does not exist then NoAccess should be returned. If, however, the
field is left blank then the authenticator is expected to check for CreateAccess.
*/
type Authenticator interface {
	// Authenticate - Check a users access level. Leave documentID blank to check for CreateAccess.
	Authenticate(userMetadata interface{}, token, documentID string) AccessLevel
}
