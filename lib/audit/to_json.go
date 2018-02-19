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
	"encoding/json"
	"os"
	"sync"

	"github.com/Jeffail/leaps/lib/store"
	"github.com/Jeffail/leaps/lib/text"
)

//------------------------------------------------------------------------------

// CompressedAuditor - Audit a documents transforms into a compressed structure
// for serialisation.
type CompressedAuditor struct {
	mut        sync.Mutex
	Transforms []text.OTransform
}

// OnTransform - Is called for every transform on a document as they arrive.
func (d *CompressedAuditor) OnTransform(tform text.OTransform) error {
	d.mut.Lock()
	if lTs := len(d.Transforms); lTs > 0 {
		// Attempt to merge this new transform into the last.
		if text.MergeTransforms(&d.Transforms[lTs-1], &tform) {
			d.mut.Unlock()
			return nil
		}
	}
	d.Transforms = append(d.Transforms, tform)
	d.mut.Unlock()
	return nil
}

//------------------------------------------------------------------------------

// ToJSON - An auditor collection that takes streams of operational transforms
// and can serialise them to JSON format:
// {
//   "document_1": [...],
//   "document_2": [...]
// }
type ToJSON struct {
	mut       sync.Mutex
	documents map[string]*CompressedAuditor
}

// NewToJSON - Create a new auditor collection that serialises to JSON
// structure.
func NewToJSON() *ToJSON {
	return &ToJSON{
		documents: map[string]*CompressedAuditor{},
	}
}

// Get - Return an auditor for a document.
func (t *ToJSON) Get(binderID string) (Auditor, error) {
	t.mut.Lock()
	defer t.mut.Unlock()

	a, ok := t.documents[binderID]
	if !ok {
		a = &CompressedAuditor{}
		t.documents[binderID] = a
	}
	return a, nil
}

// Reapply - Reapply the audited changes to a document store.
func (t *ToJSON) Reapply(docStore store.Type) error {
	t.mut.Lock()
	defer t.mut.Unlock()

	for k, v := range t.documents {
		doc, err := docStore.Read(k)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		rContent := []rune(doc.Content)
		v.mut.Lock()
		for i := range v.Transforms {
			if err = text.ApplyTransform(&rContent, &v.Transforms[i]); err != nil {
				return err
			}
		}
		v.mut.Unlock()
		doc.Content = string(rContent)
		if err = docStore.Update(doc); err != nil {
			return err
		}
	}
	return nil
}

// Serialise - Return a JSON serialised copy of all audits.
func (t *ToJSON) Serialise() ([]byte, error) {
	t.mut.Lock()
	defer t.mut.Unlock()

	collection := map[string][]text.OTransform{}

	for k, v := range t.documents {
		v.mut.Lock()
		collection[k] = v.Transforms
	}

	data, err := json.Marshal(collection)

	for _, v := range t.documents {
		v.mut.Unlock()
	}

	return data, err
}

// Deserialise - Repopulate all audits based on a JSON serialised copy.
func (t *ToJSON) Deserialise(data []byte) error {
	t.mut.Lock()
	defer t.mut.Unlock()

	collection := map[string][]text.OTransform{}
	if err := json.Unmarshal(data, &collection); err != nil {
		return err
	}

	for k, v := range collection {
		t.documents[k] = &CompressedAuditor{Transforms: v}
	}

	return nil
}

//------------------------------------------------------------------------------
