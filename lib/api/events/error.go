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

//------------------------------------------------------------------------------

// Error type strings
var (
	ErrNoSub       = "ERR_NO_SUB"
	ErrSubscribe   = "ERR_SUB"
	ErrExistingSub = "ERR_EXISTING_SUB"
	ErrBadJSON     = "ERR_BAD_JSON"
	ErrTransform   = "ERR_TRANSFORM"
	ErrMetadata    = "ERR_METADATA"
	ErrBadReq      = "ERR_BAD_REQ"
)

//------------------------------------------------------------------------------

// TypedError - An error with a type string so that API clients can easily
// categorise the fault.
type TypedError interface {
	Type() string
	Error() string
}

//------------------------------------------------------------------------------

// APIError - Implementation of TypedError for the API.
type APIError struct {
	T   string `json:"type"`
	Err string `json:"message"`
}

// Type - returns the string type.
func (a APIError) Type() string {
	return a.T
}

// Error - returns the full error string.
func (a APIError) Error() string {
	return a.Err
}

//------------------------------------------------------------------------------

// NewAPIError - Creates a type labelled API error.
func NewAPIError(errType, errString string) TypedError {
	return APIError{errType, errString}
}

//------------------------------------------------------------------------------
