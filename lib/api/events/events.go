/*
Copyright (c) 2017 Ashley Jeffs

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

package events

import (
	"github.com/Jeffail/leaps/lib/text"
)

//------------------------------------------------------------------------------

// All outbound/inbound event types
const (
	// Subscribe event type
	// Client: Send intent to subscribe to a document
	// Server: Send confirmation that document is now subscribed
	Subscribe = "subscribe"

	// Unsubscribe event type
	// Client: Send intent to unsubscribe from a document
	// Server: Send confirmation that document is now unsubscribed
	Unsubscribe = "unsubscribe"

	// Transforms event type
	// Server: Send intent to transform local copy of document with multiple
	// transforms
	Transforms = "transforms"

	// Transform event type
	// Client: Send intent to transform server copy of document
	// Server: Send intent to transform local copy of document
	Transform = "transform"

	// Correction event type
	// Server: Send correction of prior received transform from client
	Correction = "correction"

	// Metadata event type
	// Client: Send metadata to other users of document
	// Server: Send metadata from other user of document
	Metadata = "metadata"

	// GlobalMetadata event type
	// Client: Send metadata to other users of leaps service
	// Server: Send metadata from other user of leaps service
	GlobalMetadata = "global_metadata"

	// Error event type
	// Server: Send information regarding an API error
	Error = "error"

	// Ping event type
	// Client: Send intent to annoy the server
	Ping = "ping"

	// Pong event type
	// Server: Send confirm of annoyance
	Pong = "pong"
)

//------------------------------------------------------------------------------

// DocumentStripped contains fields for identifying docs but not carrying its
// contents.
type DocumentStripped struct {
	ID string `json:"id"`
}

// DocumentFull contains all data related to a document, including the current
// version of the content.
type DocumentFull struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Version int    `json:"version"`
}

// Client contains data about a client session.
type Client struct {
	Username  string `json:"username"`
	SessionID string `json:"session_id"`
}

// TformCorrection contains fields used to correct a transform.
type TformCorrection struct {
	Version int `json:"version"`
}

//------------------------------------------------------------------------------

// ErrorMessage is an API body encompassing an API error.
type ErrorMessage struct {
	Error APIError `json:"error"`
}

// TransformsMessage is an API body encompassing a slice of transforms and
// fields identifying the document target.
type TransformsMessage struct {
	Document   DocumentStripped  `json:"document"`
	Transforms []text.OTransform `json:"transforms"`
}

// TransformMessage is an API body encompassing a transform and fields
// identifying the document target.
type TransformMessage struct {
	Document  DocumentStripped `json:"document"`
	Transform text.OTransform  `json:"transform"`
}

// MetadataMessage is an API body encompassing a metadata message, fields
// identifying the document target, and fields identifying the client source.
type MetadataMessage struct {
	Document DocumentStripped `json:"document"`
	Client   interface{}      `json:"client"`
	Metadata interface{}      `json:"metadata"`
}

// GlobalMetadataMessage is an API body encompassing a global metadata message,
// and fields identifying the client source.
type GlobalMetadataMessage struct {
	Client   interface{} `json:"client"`
	Metadata interface{} `json:"metadata"`
}

// CorrectionMessage is an API body encompassing a correction to a submitted
// transform.
type CorrectionMessage struct {
	Document   DocumentStripped `json:"document"`
	Correction TformCorrection  `json:"correction"`
}

// UnsubscriptionMessage is an API body encompassing fields identifying a
// document that has been unsubscribed.
type UnsubscriptionMessage struct {
	Document DocumentStripped `json:"document"`
}

// SubscriptionMessage is an API body encompassing fields identifying a document
// that has been subscribed as well as its full contents.
type SubscriptionMessage struct {
	Document DocumentFull `json:"document"`
}

//------------------------------------------------------------------------------
