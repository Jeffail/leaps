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

package curator

import (
	"time"

	"github.com/jeffail/leaps/lib/binder"
	"github.com/jeffail/leaps/lib/store"
)

//--------------------------------------------------------------------------------------------------

// Type - Provides thread safe implementations of basic document and session creation.
type Type interface {
	// EditDocument - Find and return a binder portal to an existing document
	EditDocument(userID, token, documentID string, timeout time.Duration) (binder.Portal, error)

	// ReadDocument - Find and return a binder portal to an existing document with read only
	// privileges
	ReadDocument(userID, token, documentID string, timeout time.Duration) (binder.Portal, error)

	// CreateDocument - Create and return a binder portal to a new document
	CreateDocument(
		userID, token string, document store.Document, timeout time.Duration,
	) (binder.Portal, error)

	// Kick a user from a document, needs the documentID and userID.
	KickUser(documentID, userID string, timeout time.Duration) error

	// Get the list of all users connected to all open binders.
	GetUsers(timeout time.Duration) (map[string][]string, error)

	// Close - Close the Curator
	Close()
}

//--------------------------------------------------------------------------------------------------
