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
	"errors"
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
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Content     interface{} `json:"content"`
}

/*--------------------------------------------------------------------------------------------------
 */

/*
CreateNewDocument - Create a fresh leap document with a title, description, type and the initial
content. We also validate here that the content type provided matches the type.
*/
func CreateNewDocument(title, description, doctype string, content interface{}) (*Document, error) {

	doc := &Document{
		ID:          GenerateID(title, description),
		Title:       title,
		Description: description,
		Type:        doctype,
		Content:     content,
	}

	if err := ValidateDocument(doc); err != nil {
		return nil, err
	}

	return doc, nil
}

/*
ValidateDocument - validates that a given document has content that matches its type. If this
isn't the case it will attempt to convert the content into the expected type, and failing that
will return an error message.
*/
func ValidateDocument(doc *Document) error {
	switch doc.Type {
	case "text":
		if _, ok := doc.Content.(string); !ok {
			return errors.New("document content was not expected type (string)")
		}
	default:
		return errors.New("document type was not recognized")
	}
	return nil
}

/*
GenerateID - Create a unique ID for a document using a hash of the title, description, a timestamp
and a pseudo random integer as a base64 encoded string with the timestamp on the end in the format:
"hash-timestamp"
*/
func GenerateID(title, description string) string {
	tstamp := time.Now().Unix()
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%v%v%v%v", title, description, rand.Int(), tstamp)))

	return fmt.Sprintf("%v-%v", base64.URLEncoding.EncodeToString(hasher.Sum(nil)), tstamp)
}

/*--------------------------------------------------------------------------------------------------
 */
