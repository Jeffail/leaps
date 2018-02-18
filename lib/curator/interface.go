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

	"github.com/Jeffail/leaps/lib/audit"
	"github.com/Jeffail/leaps/lib/binder"
	"github.com/Jeffail/leaps/lib/store"
)

//------------------------------------------------------------------------------

// AuditorContainer - A type responsible for creating and managing auditors for
// string identified operational transform binders.
type AuditorContainer interface {
	// Get - Return a managed Auditor type for a binder ID.
	Get(binderID string) (audit.Auditor, error)
}

//------------------------------------------------------------------------------

// Type - Provides thread safe implementations of basic document and session
// creation.
type Type interface {
	// EditDocument - Find and return a binder portal to an existing document,
	// providing metadata for identifying content produced by the client.
	EditDocument(
		userMetadata interface{}, token, documentID string, timeout time.Duration,
	) (binder.Portal, error)

	// ReadDocument - Find and return a binder portal to an existing document
	// with read only privileges, providing metadata for identifying content
	// produced by the client.
	ReadDocument(
		userMetadata interface{}, token, documentID string, timeout time.Duration,
	) (binder.Portal, error)

	// CreateDocument - Create and return a binder portal to a new document,
	// providing metadata for identifying content produced by the client.
	CreateDocument(
		userMetadata interface{}, token string, document store.Document, timeout time.Duration,
	) (binder.Portal, error)

	// Close - Close the Curator
	Close()
}

//------------------------------------------------------------------------------
