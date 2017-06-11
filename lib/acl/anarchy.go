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

//--------------------------------------------------------------------------------------------------

// Anarchy - Most basic implementation of an ACL, everyone has access to everything.
type Anarchy struct {
	AllowCreate bool
}

// NewAnarchy - Create a new anarchy authenticator.
func NewAnarchy(allowCreate bool) Authenticator {
	return Anarchy{AllowCreate: allowCreate}
}

// Authenticate - Always returns at least edit access, because anarchy.
func (a Anarchy) Authenticate(_ interface{}, _, _ string) AccessLevel {
	if !a.AllowCreate {
		return EditAccess
	}
	return CreateAccess
}

//--------------------------------------------------------------------------------------------------
