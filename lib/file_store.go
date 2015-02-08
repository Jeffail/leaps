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
	"io/ioutil"
	"os"
	"path"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
FileStore - Most basic persistent implementation of DocumentStore. Simply stores each document into
a file within a configured directory. The ID represents the filepath relative to the configured
directory.

For example, with StoreDirectory set to /var/www, a document can be given the ID css/main.css to
create and edit the file /var/www/css/main.css
*/
type FileStore struct {
	config DocumentStoreConfig
}

/*
Create - Store document in a file location
*/
func (s *FileStore) Create(id string, doc Document) error {
	return s.Store(id, doc)
}

/*
Store - Store document in its file location.
*/
func (s *FileStore) Store(id string, doc Document) error {
	filePath := path.Join(s.config.StoreDirectory, id)
	fileDir := path.Dir(filePath)

	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		if err = os.MkdirAll(fileDir, os.ModePerm); err != nil {
			return fmt.Errorf("cannot create file path for document: %v, err: %v", id, err)
		}
	}
	return ioutil.WriteFile(filePath, []byte(doc.Content), 0666)
}

/*
Fetch - Fetch document from its file location.
*/
func (s *FileStore) Fetch(id string) (Document, error) {
	bytes, err := ioutil.ReadFile(path.Join(s.config.StoreDirectory, id))
	if err != nil {
		return Document{}, fmt.Errorf("failed to read content from document file: %v", err)
	}
	return Document{
		Content: string(bytes),
		ID:      id,
	}, nil
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
