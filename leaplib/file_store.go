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
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
FileStore - Most basic persistent implementation of DocumentStore. Simple stores each document into
a file within a configured directory.
*/
type FileStore struct {
	config DocumentStoreConfig
}

/*
Create - Store document in a file location
*/
func (s *FileStore) Create(id string, doc *Document) error {
	return s.Store(id, doc)
}

/*
Store - Store document in its file location.
*/
func (s *FileStore) Store(id string, doc *Document) error {
	file, err := os.Create(path.Join(s.config.StoreDirectory, id))
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err = fmt.Fprintln(file, strconv.QuoteToASCII(doc.Title)); err != nil {
		return err
	}
	if _, err = fmt.Fprintln(file, strconv.QuoteToASCII(doc.Description)); err != nil {
		return err
	}
	if _, err = fmt.Fprintln(file, strconv.QuoteToASCII(doc.Type)); err != nil {
		return err
	}
	serialized, err := SerializeDocumentContent(doc.Type, doc.Content)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintln(file, strconv.QuoteToASCII(serialized)); err != nil {
		return err
	}

	return nil
}

/*
Fetch - Fetch document from its file location.
*/
func (s *FileStore) Fetch(id string) (*Document, error) {
	file, err := os.Open(path.Join(s.config.StoreDirectory, id))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	doc := Document{ID: id}

	scanner := bufio.NewScanner(file)

	// Get title
	if !scanner.Scan() {
		return nil, errors.New("failed to read title from document file")
	}
	doc.Title, err = strconv.Unquote(scanner.Text())
	if err != nil {
		return nil, fmt.Errorf("unquote error: %v", err)
	}

	// Get description
	if !scanner.Scan() {
		return nil, errors.New("failed to read description from document file")
	}
	doc.Description, err = strconv.Unquote(scanner.Text())
	if err != nil {
		return nil, fmt.Errorf("unquote error: %v", err)
	}

	// Get type
	if !scanner.Scan() {
		return nil, errors.New("failed to read type from document file")
	}
	doc.Type, err = strconv.Unquote(scanner.Text())
	if err != nil {
		return nil, fmt.Errorf("unquote error: %v", err)
	}

	// Get content
	if !scanner.Scan() {
		return nil, errors.New("failed to read content from document file")
	}
	unquotedContent, err := strconv.Unquote(scanner.Text())
	if err != nil {
		return nil, fmt.Errorf("unquote error: %v", err)
	}
	if doc.Content, err = ParseDocumentContent(doc.Type, unquotedContent); err != nil {
		return nil, err
	}

	return &doc, nil
}

/*
GetFileStore - Just a func that returns a FileStore
*/
func GetFileStore(config DocumentStoreConfig) (DocumentStore, error) {
	if len(config.StoreDirectory) == 0 {
		return nil, errors.New("a file store document configuration requires a valid directory")
	}
	if _, err := os.Stat(config.StoreDirectory); os.IsNotExist(err) {
		if err = os.MkdirAll(config.StoreDirectory, os.ModePerm); err != nil {
			return nil, fmt.Errorf("cannot create file store for documents: %v", err)
		}
	}
	return &FileStore{config: config}, nil
}

/*--------------------------------------------------------------------------------------------------
 */
