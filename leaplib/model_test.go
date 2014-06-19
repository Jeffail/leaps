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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"
)

func TestSimpleTransforms(t *testing.T) {
	doc := CreateNewDocument("Test", "Doc test", "hello world")
	model := CreateModel(doc.ID)
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			if _, err := model.PushTransform(OTransform{
				Version:  model.Version + 1,
				Position: i + (j * 3) + 5,
				Insert:   fmt.Sprintf("%v", i+(j*3)),
				Delete:   0,
			}); err != nil {
				t.Errorf("Error: %v", err)
			}
		}
		if err := model.FlushTransforms(&doc.Content, time.Second*60); err != nil {
			t.Errorf("Error flushing: %v", err)
		}
	}

	if _, err := model.PushTransform(OTransform{
		Version:  model.Version + 1,
		Position: 3,
		Insert:   "*",
		Delete:   2,
	}); err != nil {
		t.Errorf("Error: %v", err)
	}

	if err := model.FlushTransforms(&doc.Content, time.Second*60); err != nil {
		t.Errorf("Error flushing: %v", err)
	}

	expected := "hel*012345678 world"
	if expected != string(doc.Content) {
		t.Errorf("Expected %v, received %v", expected, string(doc.Content))
	}
}

func TestPushPullTransforms(t *testing.T) {
	numTransforms := 100
	arrTransforms := make([]*OTransform, numTransforms)
	doc := CreateNewDocument("Test", "Doc test", "hello world")
	model := CreateModel(doc.ID)

	for j := 0; j < 2; j++ {
		for i := 0; i < numTransforms; i++ {
			arrTransforms[i] = &OTransform{
				Version:  model.Version + 1,
				Position: 0,
				Insert:   fmt.Sprintf("Transform%v", i+(j*numTransforms)),
				Delete:   0,
			}

			if _, err := model.PushTransform(*arrTransforms[i]); err != nil {
				t.Errorf("Error: %v", err)
			}

			if i%50 == 0 {
				model.FlushTransforms(&doc.Content, 60*time.Second)
			}

			tforms, _, err := model.GetTransforms(i + 1 + (j * numTransforms))
			if err != nil {
				t.Errorf("Error with i=%v, j=%v: %v", i, j, err)
				continue
			}
			if len(tforms) != 1 {
				t.Errorf("Wrong number of transforms, expected 1, received %v", len(tforms))
				continue
			}
			if arrTransforms[i].Insert != tforms[0].Insert {
				t.Errorf("\n%v\n!=\n%v", *arrTransforms[i], *tforms[0])
			}
		}

		for i := 0; i > numTransforms; i++ {
			tforms, _, err := model.GetTransforms(i + 1 + (j * numTransforms))
			if err != nil {
				t.Errorf("Error with i=%v, j=%v: %v", i, j, err)
				continue
			}
			if len(tforms) != (i + 1) {
				t.Errorf("Wrong number of transforms, expected %v, received %v",
					(i + 1), len(tforms))
				continue
			}
			if arrTransforms[i].Insert != tforms[0].Insert {
				t.Errorf("\n%v\n!=\n%v", *arrTransforms[i], *tforms[0])
			}
		}
		model.FlushTransforms(&doc.Content, 60*time.Second)
	}
}

type tformStory struct {
	Name       string       `json:"name"`
	Content    string       `json:"content"`
	Transforms []OTransform `json:"transforms"`
	TCorrected []OTransform `json:"corrected_transforms"`
	Result     string       `json:"result"`
	Flushes    []int        `json:"flushes"`
}

type storiesContainer struct {
	Stories []tformStory `json:"stories"`
}

func TestTransformStories(t *testing.T) {
	bytes, err := ioutil.ReadFile("../data/transform_stories.js")
	if err != nil {
		t.Errorf("Read file error: %v", err)
		return
	}

	var scont storiesContainer
	if err := json.Unmarshal(bytes, &scont); err != nil {
		t.Errorf("Story parse error: %v", err)
		return
	}

	for _, story := range scont.Stories {
		stages := []byte("Stages of story:\n")

		doc := CreateNewDocument("Test", "Test doc", story.Content)
		model := CreateModel(doc.ID)

		stages = append(stages,
			[]byte(fmt.Sprintf("\tInitial : %v\n", string(doc.Content)))...)

		for j, change := range story.Transforms {
			if ts, err := model.PushTransform(change); err != nil {
				t.Errorf("Failed to insert: %v", err)
			} else {
				if len(story.TCorrected) > j {
					if story.TCorrected[j].Position != ts.Position ||
						story.TCorrected[j].Version != ts.Version ||
						story.TCorrected[j].Delete != ts.Delete ||
						story.TCorrected[j].Insert != ts.Insert {
						t.Errorf("Tform does not match corrected form: %v != %v",
							story.TCorrected[j], *ts)
					}
				}
			}
			for _, at := range story.Flushes {
				if at == j {
					if err := model.FlushTransforms(&doc.Content, 60*time.Second); err != nil {
						t.Errorf("Failed to flush: %v", err)
					}
					stages = append(stages,
						[]byte(fmt.Sprintf("\tStage %-2v: %v\n", j, string(doc.Content)))...)
				}
			}
		}
		if err := model.FlushTransforms(&doc.Content, 60*time.Second); err != nil {
			t.Errorf("Failed to flush: %v", err)
		}
		if string(doc.Content) != story.Result {
			t.Errorf("Failed transform story: %v\nexpected:\n\t%v\nresult:\n\t%v\n%v",
				story.Name, story.Result, string(doc.Content), string(stages))
		}
	}
}
