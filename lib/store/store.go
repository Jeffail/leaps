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

import (
	"errors"
	"sync"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
Config - Holds generic configuration options for a document storage solution.
*/
type Config struct {
	Type           string             `json:"type" yaml:"type"`
	Name           string             `json:"name" yaml:"name"`
	StoreDirectory string             `json:"store_directory" yaml:"store_directory"`
	SQLConfig      SQLConfig          `json:"sql" yaml:"sql"`
	AzureBlobStore AzureStorageConfig `json:"azure" yaml:"azure"`
}

/*
NewConfig - Returns a default generic configuration.
*/
func NewConfig() Config {
	return Config{
		Type:           "memory",
		Name:           "",
		StoreDirectory: "",
		SQLConfig:      NewSQLConfig(),
	}
}

/*--------------------------------------------------------------------------------------------------
 */

// Errors for the  type.
var (
	ErrInvalidDocumentType = errors.New("invalid document store type")
)

/*
Store - Implemented by types able to acquire and store documents. This is abstracted in order to
accommodate for multiple storage strategies. These methods should be asynchronous if possible.
*/
type Store interface {
	// Create - Create a new document.
	Create(Document) error

	// Update - Update an existing document.
	Update(Document) error

	// Read - Read a document.
	Read(ID string) (Document, error)
}

/*--------------------------------------------------------------------------------------------------
 */

/*
Factory - Returns a document store object based on a configuration object.
*/
func Factory(config Config) (Store, error) {
	switch config.Type {
	case "file":
		return GetFileStore(config)
	case "memory":
		return GetMemoryStore(config)
	case "mock":
		return GetMockStore(config)
	case "mysql", "postgres":
		return GetSQLStore(config)
	case "azureblobstorage":
		return GetAzureBlobStore(config)
	}
	return nil, ErrInvalidDocumentType
}

/*--------------------------------------------------------------------------------------------------
 */

// Errors for the MemoryStore type.
var (
	ErrDocumentNotExist = errors.New("attempting to fetch memory store that has not been initialized")
)

/*
MemoryStore - Most basic implementation of , simply keeps the document in memory. Has
zero persistence across sessions.
*/
type MemoryStore struct {
	documents map[string]Document
	mutex     sync.RWMutex
}

/*
Create - Store document in memory.
*/
func (s *MemoryStore) Create(doc Document) error {
	return s.Update(doc)
}

/*
Update - Update document in memory.
*/
func (s *MemoryStore) Update(doc Document) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.documents[doc.ID] = doc
	return nil
}

/*
Read - Read document from memory.
*/
func (s *MemoryStore) Read(id string) (Document, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	doc, ok := s.documents[id]
	if !ok {
		return doc, ErrDocumentNotExist
	}
	return doc, nil
}

/*
GetMemoryStore - Just a func that returns a MemoryStore
*/
func GetMemoryStore(config Config) (Store, error) {
	return &MemoryStore{
		documents: make(map[string]Document),
	}, nil
}

/*--------------------------------------------------------------------------------------------------
 */

/*
GetMockStore - returns a MemoryStore with a document already created for testing purposes. The
document has the ID of the config value 'Name'.
*/
func GetMockStore(config Config) (Store, error) {
	memStore := &MemoryStore{
		documents: make(map[string]Document),
	}
	memStore.documents[config.Name] = Document{
		ID:      config.Name,
		Content: "Open this page multiple times to see the edits appear in all of them.",
	}
	return memStore, nil
}

/*--------------------------------------------------------------------------------------------------
 */
