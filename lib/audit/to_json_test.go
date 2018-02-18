// Copyright (c) 2017 Ashley Jeffs
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, sub to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package audit

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Jeffail/leaps/lib/store"
	"github.com/Jeffail/leaps/lib/text"
)

//------------------------------------------------------------------------------

func TestBasicSerialisation(t *testing.T) {
	j := NewToJSON()

	foo1, err := j.Get("foo1")
	if err != nil {
		t.Error(err)
	}
	foo2, err := j.Get("foo2")
	if err != nil {
		t.Error(err)
	}

	foo1.OnTransform(text.OTransform{Insert: "hello "})
	foo2.OnTransform(text.OTransform{Insert: "yo "})

	foo3, err := j.Get("foo1")
	if err != nil {
		t.Error(err)
	}

	foo3.OnTransform(text.OTransform{Insert: "world", Position: 6})

	expected := []byte(`{"foo1":[{"position":0,"num_delete":0,"insert":"hello world","version":0}],"foo2":[{"position":0,"num_delete":0,"insert":"yo ","version":0}]}`)
	actual, err := j.Serialise()
	if err != nil {
		t.Error(err)
	}

	if string(expected) != string(actual) {
		t.Errorf("Wrong serialised output: %s != %s", expected, actual)
	}
}

type dummyStore struct {
	docs map[string]store.Document
}

func (d *dummyStore) Create(store.Document) error {
	return errors.New("nope")
}

func (d *dummyStore) Read(ID string) (store.Document, error) {
	if d, ok := d.docs[ID]; ok {
		return d, nil
	}
	return store.Document{ID: ID}, nil
}

func (d *dummyStore) Update(doc store.Document) error {
	d.docs[doc.ID] = doc
	return nil
}

func TestBasicDeserialisation(t *testing.T) {
	j := NewToJSON()
	if err := j.Deserialise([]byte(`{
	"foo1":[
		{"position":0,"num_delete":0,"insert":"hello world","version":0}
	],
	"foo2":[
		{"position":0,"num_delete":0,"insert":"yo ","version":0}
	]}`)); err != nil {
		t.Error(err)
	}

	foo1, err := j.Get("foo1")
	if err != nil {
		t.Error(err)
	}
	foo2, err := j.Get("foo2")
	if err != nil {
		t.Error(err)
	}

	if err := foo1.OnTransform(text.OTransform{Insert: "another hello"}); err != nil {
		t.Error(err)
	}
	if err := foo2.OnTransform(text.OTransform{Insert: "another yo"}); err != nil {
		t.Error(err)
	}

	expected := []byte(`{"foo1":[{"position":0,"num_delete":0,"insert":"another hellohello world","version":0}],"foo2":[{"position":0,"num_delete":0,"insert":"another yoyo ","version":0}]}`)
	actual, err := j.Serialise()
	if err != nil {
		t.Error(err)
	}

	if string(expected) != string(actual) {
		t.Errorf("Wrong serialised output: %s != %s", expected, actual)
	}

	docStore := dummyStore{docs: map[string]store.Document{}}
	j.Reapply(&docStore)

	if exp, actual := 2, len(docStore.docs); exp != actual {
		t.Errorf("Wrong count of applied documents: %v != %v", exp, actual)
	}
	if exp, actual := "another hellohello world", docStore.docs["foo1"].Content; exp != actual {
		t.Errorf("Wrong result in applied documents: %v != %v", exp, actual)
	}
	if exp, actual := "another yoyo ", docStore.docs["foo2"].Content; exp != actual {
		t.Errorf("Wrong result in applied documents: %v != %v", exp, actual)
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Try running with: go test -race

	j := NewToJSON()

	startChan := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(5)

	go func() {
		<-startChan
		for i := 0; i < 10; i++ {
			if _, err := j.Serialise(); err != nil {
				t.Error(err)
			}
			<-time.After(time.Millisecond)
		}
		wg.Done()
	}()

	for k := 0; k < 4; k++ {
		go func() {
			<-startChan
			for i := 0; i < 10; i++ {
				foo1, err := j.Get("foo1")
				if err != nil {
					t.Error(err)
				} else {
					err = foo1.OnTransform(text.OTransform{Insert: "Looped"})
					if err != nil {
						t.Error(err)
					}
				}
				<-time.After(time.Millisecond)
			}
			wg.Done()
		}()
	}

	close(startChan)
	if _, err := j.Serialise(); err != nil {
		t.Error(err)
	}
	wg.Wait()
}

//------------------------------------------------------------------------------
