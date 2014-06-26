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

func TestTextModelSimpleTransforms(t *testing.T) {
	doc, err := CreateNewDocument("Test", "Doc test", "text", "hello world")
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}

	model := CreateTextModel(doc.ID)
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			if _, _, err = model.PushTransform(OTransform{
				Version:  model.GetVersion() + 1,
				Position: i + (j * 3) + 5,
				Insert:   fmt.Sprintf("%v", i+(j*3)),
				Delete:   0,
			}); err != nil {
				t.Errorf("Error: %v", err)
			}
		}
		if err = model.FlushTransforms(&doc.Content, time.Second*60); err != nil {
			t.Errorf("Error flushing: %v", err)
		}
	}

	if _, _, err = model.PushTransform(OTransform{
		Version:  model.GetVersion() + 1,
		Position: 3,
		Insert:   "*",
		Delete:   2,
	}); err != nil {
		t.Errorf("Error: %v", err)
	}

	if err = model.FlushTransforms(&doc.Content, time.Second*60); err != nil {
		t.Errorf("Error flushing: %v", err)
	}

	expected := "hel*012345678 world"
	received := doc.Content.(string)
	if expected != received {
		t.Errorf("Expected %v, received %v", expected, received)
	}
}

func TestPushPullTransforms(t *testing.T) {
	numTransforms := 100
	arrTransforms := make([]OTransform, numTransforms)
	doc, err := CreateNewDocument("Test", "Doc test", "text", "hello world")
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	model := CreateTextModel(doc.ID)

	for j := 0; j < 2; j++ {
		for i := 0; i < numTransforms; i++ {
			arrTransforms[i] = OTransform{
				Version:  model.GetVersion() + 1,
				Position: 0,
				Insert:   fmt.Sprintf("Transform%v", i+(j*numTransforms)),
				Delete:   0,
			}

			if _, _, err := model.PushTransform(arrTransforms[i]); err != nil {
				t.Errorf("Error: %v", err)
			}

			if i%50 == 0 {
				model.FlushTransforms(&doc.Content, 60*time.Second)
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

		doc, err := CreateNewDocument("Test", "Test doc", "text", story.Content)
		if err != nil {
			t.Errorf("Error: %v", err)
			return
		}
		model := CreateTextModel(doc.ID)

		stages = append(stages,
			[]byte(fmt.Sprintf("\tInitial : %v\n", doc.Content.(string)))...)

		for j, change := range story.Transforms {
			if tsWrap, _, err := model.PushTransform(change); err != nil {
				t.Errorf("Failed to insert: %v", err)
			} else {
				ts, ok := tsWrap.(OTransform)
				if ok {
					if len(story.TCorrected) > j {
						if story.TCorrected[j].Position != ts.Position ||
							story.TCorrected[j].Version != ts.Version ||
							story.TCorrected[j].Delete != ts.Delete ||
							story.TCorrected[j].Insert != ts.Insert {
							t.Errorf("Tform does not match corrected form: %v != %v",
								story.TCorrected[j], ts)
						}
					}
				} else {
					t.Errorf("did not receive expected OTransform")
				}
			}
			for _, at := range story.Flushes {
				if at == j {
					if err = model.FlushTransforms(&doc.Content, 60*time.Second); err != nil {
						t.Errorf("Failed to flush: %v", err)
					}
					stages = append(stages,
						[]byte(fmt.Sprintf("\tStage %-2v: %v\n", j, doc.Content.(string)))...)
				}
			}
		}
		if err = model.FlushTransforms(&doc.Content, 60*time.Second); err != nil {
			t.Errorf("Failed to flush: %v", err)
		}
		result := doc.Content.(string)
		if result != story.Result {
			t.Errorf("Failed transform story: %v\nexpected:\n\t%v\nresult:\n\t%v\n%v",
				story.Name, story.Result, result, string(stages))
		}
	}
}
