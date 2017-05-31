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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

//------------------------------------------------------------------------------

// Errors for the FileStore type.
var (
	ErrInvalidDirectory = errors.New("invalid directory")
)

//------------------------------------------------------------------------------

/*
File - Most basic persistent implementation of store.Crud. Simply stores each
document into a file within a configured directory. The ID represents the
filepath relative to the configured directory.

For example, with StoreDirectory set to /var/www, a document can be given the ID
css/main.css to create and edit the file /var/www/css/main.css
*/
type File struct {
	storeDirectory string
	allowWrites    bool

	cacheLock      sync.Mutex
	unwrittenCache map[string]Document
}

// NewFile - Just a func that returns a File based store type.
func NewFile(storeDirectory string, allowWrites bool) (Type, error) {
	if len(storeDirectory) == 0 {
		return nil, ErrInvalidDirectory
	}
	if _, err := os.Stat(storeDirectory); os.IsNotExist(err) {
		if allowWrites {
			err = os.MkdirAll(storeDirectory, os.ModePerm)
		}
		if err != nil {
			return nil, fmt.Errorf("cannot access file store for documents: %v", err)
		}
	}
	return &File{
		storeDirectory: storeDirectory,
		allowWrites:    allowWrites,
		unwrittenCache: map[string]Document{},
	}, nil
}

//------------------------------------------------------------------------------

// Create - Create a new document in a file location
func (s *File) Create(doc Document) error {
	return s.Update(doc)
}

// Update - Update a document in its file location.
func (s *File) Update(doc Document) error {
	if !s.allowWrites {
		// Write changes to a local cache rather than the file itself.
		s.cacheLock.Lock()
		s.unwrittenCache[doc.ID] = doc
		s.cacheLock.Unlock()
		return nil
	}

	filePath := filepath.Join(s.storeDirectory, doc.ID)
	fileDir := filepath.Dir(filePath)

	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		if err = os.MkdirAll(fileDir, os.ModePerm); err != nil {
			return fmt.Errorf("cannot create file path for document: %v, err: %v", doc.ID, err)
		}
	}
	return ioutil.WriteFile(filePath, []byte(doc.Content), 0666)
}

// Read - Read document from its file location.
func (s *File) Read(id string) (Document, error) {
	if !s.allowWrites {
		// If we aren't writing to disk then we might have cached our changes.
		s.cacheLock.Lock()
		d, ok := s.unwrittenCache[id]
		s.cacheLock.Unlock()
		if ok {
			return d, nil
		}
	}

	bytes, err := ioutil.ReadFile(filepath.Join(s.storeDirectory, id))
	if err != nil {
		return Document{}, fmt.Errorf("failed to read content from document file: %v", err)
	}
	return Document{
		Content: string(bytes),
		ID:      id,
	}, nil
}

//------------------------------------------------------------------------------
