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

package leapnet

import (
	"github.com/jeffail/leaps/leaplib"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
LeapLocator - An interface capable of locating and creating leaps documents. This can either be a
curator, which deals with documents on the local service, or a TBD, which load balances between
servers of curators.
*/
type LeapLocator interface {
	// FindDocument - Find and return a binder portal to an existing document
	FindDocument(string, string) (*leaplib.BinderPortal, error)

	// NewDocument - Create and return a binder portal to a new document
	NewDocument(string, *leaplib.Document) (*leaplib.BinderPortal, error)

	// GetLogger - Obtain a reference to the LeapsLogger held by our curator
	GetLogger() *leaplib.LeapsLogger

	// Close - Close the LeapLocator
	Close()
}

/*--------------------------------------------------------------------------------------------------
 */
