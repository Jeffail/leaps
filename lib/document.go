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

package lib

import (
	"errors"
	"fmt"
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
		ID:          GenerateID(fmt.Sprintf("%v%v", title, description)),
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
SerializeDocumentContent - Serialize the content of a document into a string, based on its type.
*/
func SerializeDocumentContent(doctype string, content interface{}) (string, error) {
	switch doctype {
	case "text":
		if str, ok := content.(string); ok {
			return str, nil
		}
	default:
		return "", errors.New("document type was not recognized")
	}
	return "", errors.New("document content was not expected type")
}

/*
ParseDocumentContent - Parse a string into a document content based on its type.
*/
func ParseDocumentContent(doctype string, content string) (interface{}, error) {
	switch doctype {
	case "text":
		return content, nil
	default:
		return nil, errors.New("document type was not recognized")
	}
}

/*--------------------------------------------------------------------------------------------------
 */
