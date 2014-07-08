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
	"errors"
	"sync"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
DocumentStoreConfig - Holds generic configuration options for a document storage solution.
*/
type DocumentStoreConfig struct {
	Type           string    `json:"type"`
	Name           string    `json:"name"`
	StoreDirectory string    `json:"store_directory"`
	SQLConfig      SQLConfig `json:"sql"`
}

/*
DefaultDocumentStoreConfig - Returns a default generic configuration.
*/
func DefaultDocumentStoreConfig() DocumentStoreConfig {
	return DocumentStoreConfig{
		Type:           "memory",
		Name:           "",
		StoreDirectory: "",
		SQLConfig:      DefaultSQLConfig(),
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
	Create(string, *Document) error
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
	case "mysql", "sqlite3", "postgres":
		return GetSQLStore(config)
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
Create - Store document in memory.
*/
func (s *MemoryStore) Create(id string, doc *Document) error {
	return s.Store(id, doc)
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
