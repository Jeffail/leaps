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

package binder

import (
	"time"

	"github.com/Jeffail/leaps/lib/store"
	"github.com/Jeffail/leaps/lib/text"
)

//------------------------------------------------------------------------------

// TransformSink - A type that consumes transforms,
type TransformSink interface {
	// PushTransform - Process a newly received transform and return the
	// corrected version.
	PushTransform(ot text.OTransform) (text.OTransform, int, error)

	// IsDirty - Check whether the sink has uncommitted changes.
	IsDirty() bool

	// GetVersion - Returns the current version of the underlying transform
	// model
	GetVersion() int

	// FlushTransforms - apply all unapplied transforms to content and perform
	// any subsequent cleaning up of the transforms stack, transforms within
	// the secondsRetention period will be preserved for corrections. Returns a
	// bool indicating whether any changes were applied.
	FlushTransforms(content *string, secondsRetention int64) (bool, error)
}

//------------------------------------------------------------------------------

// Portal - An interface used by clients to contact a connected binder type.
type Portal interface {
	// ClientMetadata - Returns the user identifying metadata associated with
	// this binder session.
	ClientMetadata() interface{}

	// BaseVersion - Returns the version of the binder as it was when this
	// session opened.
	BaseVersion() int

	// Document - Returns the document contents as it was when this session
	// opened.
	Document() store.Document

	// ReleaseDocument - Releases the cached document.
	ReleaseDocument()

	// TransformReadChan - Get the channel for reading transforms from other
	// binder clients.
	TransformReadChan() <-chan text.OTransform

	// MetadataReadChan - Get the channel for reading meta updates from other
	// binder clients.
	MetadataReadChan() <-chan ClientMetadata

	// SendTransform - Submits an operational transform to the document, this
	// call adds the transform to the stack of pending changes and broadcasts it
	// to all other connected clients. The transform must be submitted with the
	// target version (the version that the client believed it was, at the time
	// it was made), and the actual version is returned.
	SendTransform(ot text.OTransform, timeout time.Duration) (int, error)

	// SendMetadata - Broadcasts metadata out to all other connected clients.
	SendMetadata(metadata interface{})

	// Exit - Inform the binder that this client is shutting down, this call
	// will block until acknowledged by the binder. Therefore, you may specify a
	// timeout.
	Exit(timeout time.Duration)
}

// Type - Provides thread safe implementations of binder and session creation.
type Type interface {
	// ID - Returns the ID of this binder.
	ID() string

	// Subscribe - Register a new client as an editor of this binder document.
	// Metadata can be provided in order to identify submissions from the
	// client.
	Subscribe(metadata interface{}, timeout time.Duration) (Portal, error)

	// SubscribeReadOnly - Register a new client as a read only viewer of this
	// binder document.
	SubscribeReadOnly(metadata interface{}, timeout time.Duration) (Portal, error)

	// Close - Close the binder and shut down all clients, also flushes and
	// cleans up the document.
	Close()
}

//------------------------------------------------------------------------------
