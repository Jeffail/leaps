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

package leaplib

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/rand"
	"time"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
Document - A representation of a leap document.
*/
type Document struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

/*--------------------------------------------------------------------------------------------------
 */

/*
CreateNewDocument - Create a fresh leap document with a title, description and the initial content.
*/
func CreateNewDocument(title, description, content string) *Document {

	/* We create a unique ID for each document. We do this using a hash of the title, description,
	 * a timestamp and a pseudo random integer as a base64 encoded string with the timestamp on the
	 * end in the format: "hash-timestamp"
	 */
	tstamp := time.Now().Unix()
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%v%v%v%v", title, description, rand.Int(), tstamp)))
	id := fmt.Sprintf("%v-%v", base64.URLEncoding.EncodeToString(hasher.Sum(nil)), tstamp)

	doc := &Document{
		ID:          id,
		Title:       title,
		Description: description,
		Content:     content,
	}

	return doc
}

/*--------------------------------------------------------------------------------------------------
 */
