/*
Copyright (c) 2014 Ashley Jeffs

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

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
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewCurator(t *testing.T) {
	cur, err := CreateNewCurator(DefaultCuratorConfig())
	if err != nil {
		t.Errorf("Create curator error: %v", err)
		return
	}

	cur.Close()
}

func TestCuratorClients(t *testing.T) {
	config := DefaultBinderConfig()
	config.FlushPeriod = 5000

	curator, err := CreateNewCurator(DefaultCuratorConfig())
	if err != nil {
		t.Errorf("error: %v", err)
	}

	doc := CreateNewDocument("test", "test1", "hello world")
	portal, err := curator.NewDocument(doc)
	doc = portal.Document
	if err != nil {
		t.Errorf("error: %v", err)
	}

	tform := func(i int) *OTransform {
		return &OTransform{
			Position: 0,
			Version:  i,
			Delete:   0,
			Insert:   fmt.Sprintf("%v", i),
		}
	}

	if v, err := portal.SendTransforms(
		[]*OTransform{tform(portal.Version + 1)},
	); v != 2 || err != nil {
		t.Errorf("Send Transform error, v: %v, err: %v", v, err)
	}

	wg := sync.WaitGroup{}
	wg.Add(20)

	for i := 0; i < 10; i++ {
		if b, e := curator.FindDocument(doc.ID); e != nil {
			t.Errorf("error: %v", e)
		} else {
			go goodClient(b, t, &wg)
		}
		if b, e := curator.FindDocument(doc.ID); e != nil {
			t.Errorf("error: %v", e)
		} else {
			go badClient(b, t, &wg)
		}
	}

	wg.Add(50)

	for i := 0; i < 50; i++ {
		vstart := i*3 + 3
		if i%2 == 0 {
			if b, e := curator.FindDocument(doc.ID); e != nil {
				t.Errorf("error: %v", e)
			} else {
				go goodClient(b, t, &wg)
			}
			if b, e := curator.FindDocument(doc.ID); e != nil {
				t.Errorf("error: %v", e)
			} else {
				go badClient(b, t, &wg)
			}
		}
		if v, err := portal.SendTransforms(
			[]*OTransform{tform(vstart), tform(vstart + 1), tform(vstart + 2)},
		); v != vstart || err != nil {
			t.Errorf("Send Transform error, expected v: %v, got v: %v, err: %v", vstart, v, err)
		}
	}

	curator.Close()
	for {
		select {
		case err := <-curator.errorChan:
			t.Errorf("Curator received error: %v", err)
		case <-time.After(50 * time.Millisecond):
			return
		}
	}

	wg.Wait()
}
