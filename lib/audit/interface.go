// Copyright (c) 2017 Ashley Jeffs
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, sub to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package audit

import (
	"github.com/Jeffail/leaps/lib/text"
)

//------------------------------------------------------------------------------

// Auditor - A type that receives all transforms from a running binder as they
// arrive. The purpose of this type is to expose the flowing data to other
// components.
type Auditor interface {
	// OnTransform - Is called by binder threads synchronously for each received
	// transform. Therefore, the implementation must be thread safe and avoid
	// blocking. An implementation may wish to perform validation on the
	// transform, in the case of a 'fail' an error should be returned, which
	// will prevent the transform from being applied.
	OnTransform(tform text.OTransform) error
}

//------------------------------------------------------------------------------
