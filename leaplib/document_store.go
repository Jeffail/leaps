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
	"sync"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
DocumentStoreConfig - Holds generic configuration options for a document storage solution.
*/
type DocumentStoreConfig struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	StoreDirectory string `json:"store_directory"`
}

/*
DefaultDocumentStoreConfig - Returns a default generic configuration.
*/
func DefaultDocumentStoreConfig() DocumentStoreConfig {
	return DocumentStoreConfig{
		Type:           "memory",
		Name:           "",
		StoreDirectory: "",
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
DocumentStore - Implemented by types able to acquire and store documents. This is abstracted in
order to accommodate for multiple storage strategies. These methods should be asynchronous if
possible.
*/
type DocumentStore interface {
	Store(string, *Document) error
	Fetch(string) (*Document, error)
}

/*--------------------------------------------------------------------------------------------------
 */

/*
DocumentStoreFactory - Returns a document store object based on a configuration object.
*/
func DocumentStoreFactory(config DocumentStoreConfig) (DocumentStore, error) {
	switch config.Type {
	case "file":
		return GetFileStore(config)
	case "memory":
		return GetMemoryStore(config)
	case "mock":
		return GetMockStore(config)
	}
	return nil, errors.New("configuration provided invalid document store type")
}

/*--------------------------------------------------------------------------------------------------
 */

/*
MemoryStore - Most basic implementation of DocumentStore, simply keeps the document in memory. Has
zero persistence across sessions.
*/
type MemoryStore struct {
	documents map[string]*Document
	mutex     sync.RWMutex
}

/*
Store - Store document in memory.
*/
func (s *MemoryStore) Store(id string, doc *Document) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.documents[id] = doc
	return nil
}

/*
Fetch - Fetch document from memory.
*/
func (s *MemoryStore) Fetch(id string) (*Document, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	doc, ok := s.documents[id]
	if !ok {
		return nil, errors.New("attempting to fetch memory store that has not been initialized")
	}
	return doc, nil
}

/*
GetMemoryStore - Just a func that returns a MemoryStore
*/
func GetMemoryStore(config DocumentStoreConfig) (DocumentStore, error) {
	return &MemoryStore{
		documents: make(map[string]*Document),
	}, nil
}

/*--------------------------------------------------------------------------------------------------
 */

/*
GetMockStore - returns a MemoryStore with a document already created for testing purposes. The
document has the ID of the config value 'Name'.
*/
func GetMockStore(config DocumentStoreConfig) (DocumentStore, error) {
	memStore := &MemoryStore{
		documents: make(map[string]*Document),
	}
	memStore.documents[config.Name] = &Document{
		ID:          config.Name,
		Title:       config.Name,
		Description: config.Name,
		Type:        "text",
		Content:     "Open this page multiple times to see the edits appear in all of them.",
	}
	return memStore, nil
}

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
	if !scanner.Scan() {
		return nil, errors.New("failed to read title from document file")
	}
	doc.Title, err = strconv.Unquote(scanner.Text())
	if err != nil {
		return nil, fmt.Errorf("unquote error: %v", err)
	}

	if !scanner.Scan() {
		return nil, errors.New("failed to read description from document file")
	}
	doc.Description, err = strconv.Unquote(scanner.Text())
	if err != nil {
		return nil, fmt.Errorf("unquote error: %v", err)
	}

	if !scanner.Scan() {
		return nil, errors.New("failed to read type from document file")
	}
	doc.Type, err = strconv.Unquote(scanner.Text())
	if err != nil {
		return nil, fmt.Errorf("unquote error: %v", err)
	}

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
		return nil, errors.New("A file store document configuration requires a valid directory")
	}
	if _, err := os.Stat(config.StoreDirectory); os.IsNotExist(err) {
		if err = os.MkdirAll(config.StoreDirectory, os.ModePerm); err != nil {
			return nil, fmt.Errorf("Cannot create file store for documents: %v", err)
		}
	}
	return &FileStore{config: config}, nil
}

/*--------------------------------------------------------------------------------------------------
 */
