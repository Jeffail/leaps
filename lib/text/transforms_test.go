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

package text

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"
)

//--------------------------------------------------------------------------------------------------

func TestTransformMerge(t *testing.T) {
	type mergeTest struct {
		first  OTransform
		second OTransform
		result OTransform
	}

	tests := []mergeTest{
		{
			first:  OTransform{Position: 5, Insert: "hello", Delete: 0},
			second: OTransform{Position: 10, Insert: " world", Delete: 0},
			result: OTransform{Position: 5, Insert: "hello world", Delete: 0},
		},
		{
			first:  OTransform{Position: 5, Insert: "hello", Delete: 4},
			second: OTransform{Position: 10, Insert: " world", Delete: 3},
			result: OTransform{Position: 5, Insert: "hello world", Delete: 7},
		},
		{
			first:  OTransform{Position: 5, Insert: "hello", Delete: 2},
			second: OTransform{Position: 5, Insert: "j", Delete: 1},
			result: OTransform{Position: 5, Insert: "jello", Delete: 2},
		},
		{
			first:  OTransform{Position: 5, Insert: "hello", Delete: 0},
			second: OTransform{Position: 7, Insert: "y world", Delete: 4},
			result: OTransform{Position: 5, Insert: "hey world", Delete: 1},
		},
		{
			first:  OTransform{Position: 5, Insert: "0", Delete: 1},
			second: OTransform{Position: 6, Insert: "1", Delete: 1},
			result: OTransform{Position: 5, Insert: "01", Delete: 2},
		},
	}

	for _, test := range tests {
		if !MergeTransforms(&test.first, &test.second) {
			t.Errorf("Failed to merge transforms: %v", test)
		}
		if !reflect.DeepEqual(test.first, test.result) {
			t.Errorf("Unexpected result: %v != %v", test.first, test.result)
		}
	}
}

func TestTransformMergeBad(t *testing.T) {
	type mergeFailTest struct {
		first  OTransform
		second OTransform
	}

	tests := []mergeFailTest{
		{
			first:  OTransform{Position: 5, Insert: "hello", Delete: 0},
			second: OTransform{Position: 11, Insert: " world", Delete: 0},
		},
	}

	for _, test := range tests {
		if MergeTransforms(&test.first, &test.second) {
			t.Errorf("Bad merge returns success: %v", test)
		}
	}
}

//--------------------------------------------------------------------------------------------------

func TestPrematureTransforms(t *testing.T) {
	type prematureTest struct {
		Name                  string       `json:"name"`
		Content               string       `json:"content"`
		Result                string       `json:"result"`
		LocalTform            OTransform   `json:"local_tform"`
		RemoteTforms          []OTransform `json:"remote_tforms"`
		CorrectedLocalTform   OTransform   `json:"corrected_local_tform"`
		CorrectedRemoteTforms []OTransform `json:"corrected_remote_tforms"`
	}

	var tests struct {
		Stories []prematureTest `json:"stories"`
	}

	storyData, err := ioutil.ReadFile("../../test/stories/client_doc_sync_stories.js")
	if err != nil {
		t.Errorf("Read file error: %v", err)
		return
	}

	if err := json.Unmarshal(storyData, &tests); err != nil {
		t.Errorf("Story parse error: %v", err)
		return
	}

	for _, test := range tests.Stories {
		// contentClient simulates the local clients doc copy
		contentClient := bytes.Runes([]byte(test.Content))

		// contentServer simulates the servers doc copy
		contentServer := bytes.Runes([]byte(test.Content))

		// contentOrdered is a hypothetical ideal document receiving each change
		// in the intended order
		contentOrdered := bytes.Runes([]byte(test.Content))

		// Apply our local transform to the client doc immediately
		applyTransform(&contentClient, &test.LocalTform)

		// Take a copy of our local tform to represent the servers' copy
		remoteOT := test.LocalTform

		// For each tform dispatched by the server
		for i := range test.RemoteTforms {
			// Fix the server copy so that it can be applied after remote tform
			FixOutOfDateTransform(&remoteOT, &test.RemoteTforms[i])

			// Apply the remote tform in the correct original order for both
			// the server and idealistic docs
			applyTransform(&contentOrdered, &test.RemoteTforms[i])
			applyTransform(&contentServer, &test.RemoteTforms[i])

			// Fix our premature clients tform with respect to remote tform
			FixPrematureTransform(&test.RemoteTforms[i], &test.LocalTform)

			// Validate the fix is what we expected
			if !reflect.DeepEqual(test.RemoteTforms[i], test.CorrectedRemoteTforms[i]) {
				t.Errorf("Wrong remote correction from story %s, %v != %v",
					test.Name, test.RemoteTforms[i], test.CorrectedRemoteTforms[i])
			}

			// Apply the fixed remote tform to our client doc
			applyTransform(&contentClient, &test.RemoteTforms[i])
		}

		applyTransform(&contentOrdered, &test.LocalTform)
		applyTransform(&contentServer, &remoteOT)

		if exp, act := test.Result, string(contentClient); exp != act {
			t.Errorf("Wrong client doc result from story %s, %v != %v", test.Name, exp, act)
		}
		if exp, act := test.Result, string(contentOrdered); exp != act {
			t.Errorf("Wrong ordered doc result from story %s, %v != %v", test.Name, exp, act)
		}
		if exp, act := test.Result, string(contentServer); exp != act {
			t.Errorf("Wrong server doc result from story %s, %v != %v", test.Name, exp, act)
		}

		if !reflect.DeepEqual(remoteOT, test.CorrectedLocalTform) {
			t.Errorf("Story (%s) contains invalid corrected transform, %v != %v",
				test.Name, remoteOT, test.CorrectedLocalTform)
		}
		if !reflect.DeepEqual(test.LocalTform, test.CorrectedLocalTform) {
			t.Errorf("Wrong corrected local tform from story %s, %v != %v",
				test.Name, test.LocalTform, test.CorrectedLocalTform)
		}
	}
}

//--------------------------------------------------------------------------------------------------
