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
	"testing"

	"github.com/jeffail/leaps/lib/text"
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

//------------------------------------------------------------------------------
