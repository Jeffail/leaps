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

package io

import "sync"

//------------------------------------------------------------------------------

// ConcurrentJSON - Wraps a ReadWriteJSONCloser with mutexes that allow multiple
// concurrent readers/writers.
type ConcurrentJSON struct {
	readMut  sync.Mutex
	writeMut sync.Mutex
	C        ReadWriteJSONCloser
}

// ReadJSON - Safely read JSON.
func (c *ConcurrentJSON) ReadJSON(v interface{}) error {
	c.readMut.Lock()
	err := c.C.ReadJSON(v)
	c.readMut.Unlock()
	return err
}

// WriteJSON - Safely write JSON.
func (c *ConcurrentJSON) WriteJSON(v interface{}) error {
	c.writeMut.Lock()
	err := c.C.WriteJSON(v)
	c.writeMut.Unlock()
	return err
}

// Close - Close the read/writer
func (c *ConcurrentJSON) Close() error {
	return c.C.Close()
}

//------------------------------------------------------------------------------
