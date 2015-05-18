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

package store

import "github.com/jeffail/leaps/lib/util"

/*--------------------------------------------------------------------------------------------------
 */

/*
Document - A representation of a leap document.
*/
type Document struct {
	ID      string `json:"id" yaml:"id"`
	Content string `json:"content" yaml:"content"`
}

/*--------------------------------------------------------------------------------------------------
 */

/*
NewDocument - Create a fresh leap document with a title, description, type and the initial content.
*/
func NewDocument(content string) (*Document, error) {
	doc := &Document{
		ID:      util.GenerateStampedUUID(),
		Content: content,
	}
	return doc, nil
}

/*--------------------------------------------------------------------------------------------------
 */
