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

//--------------------------------------------------------------------------------------------------

// Errors for the Memory type.
var (
	ErrDocumentNotExist = errors.New("attempted to fetch memory doc that has not been initialized")
)

// Memory - Simply keeps documents in memory. Has zero persistence across sessions.
type Memory struct {
	documents map[string]Document
	mutex     sync.RWMutex
}

// NewMemory - Returns a Memory store type.
func NewMemory() Type {
	return &Memory{
		documents: make(map[string]Document),
	}
}

// Create - Store document in memory.
func (s *Memory) Create(doc Document) error {
	return s.Update(doc)
}

// Update - Update document in memory.
func (s *Memory) Update(doc Document) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.documents[doc.ID] = doc
	return nil
}

// Read - Read document from memory.
func (s *Memory) Read(id string) (Document, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	doc, ok := s.documents[id]
	if !ok {
		return doc, ErrDocumentNotExist
	}
	return doc, nil
}

//--------------------------------------------------------------------------------------------------
