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

package log

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestStats(t *testing.T) {
	statInc := 10

	config := DefaultStatsConfig()
	config.RootPath = ""

	stats := NewStats(config)

	wg := sync.WaitGroup{}
	wg.Add(statInc)

	stats.Gauge("test.stats.generic", 10)
	stats.Timing("test.other.thing", 5)

	for i := 0; i < statInc; i++ {
		go func() {
			stats.Incr("test.stats.increment", 1)
			wg.Done()
		}()
	}

	wg.Wait()

	expectedJSON := struct {
		Test struct {
			Stats struct {
				Generic   int `json:"generic"`
				Increment int `json:"increment"`
			} `json:"stats"`
			Other struct {
				Thing string `json:"thing"`
			} `json:"other"`
		} `json:"test"`
	}{}

	fullResult, err := stats.GetStats(time.Second)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	json.Unmarshal([]byte(fullResult), &expectedJSON)
	byteResult, _ := json.Marshal(expectedJSON)

	result := string(byteResult)
	expected := `{"test":{"stats":{"generic":10,"increment":10},"other":{"thing":"5ms"}}}`

	if result != expected {
		t.Errorf("Result != expected: %v != %v", result, expected)
	}
}

func TestRootPathStats(t *testing.T) {
	statInc := 10
	conf := DefaultStatsConfig()
	conf.RootPath = "root.base"
	stats := NewStats(conf)

	wg := sync.WaitGroup{}
	wg.Add(statInc)

	stats.Gauge("test.stats.generic", 10)
	stats.Timing("test.other.thing", 5)

	for i := 0; i < statInc; i++ {
		go func() {
			stats.Incr("test.stats.increment", 1)
			wg.Done()
		}()
	}

	wg.Wait()

	expectedJSON := struct {
		Root struct {
			Base struct {
				Test struct {
					Stats struct {
						Generic   int `json:"generic"`
						Increment int `json:"increment"`
					} `json:"stats"`
					Other struct {
						Thing string `json:"thing"`
					} `json:"other"`
				} `json:"test"`
			} `json:"base"`
		} `json:"root"`
	}{}

	fullResult, err := stats.GetStats(time.Second)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	json.Unmarshal([]byte(fullResult), &expectedJSON)
	byteResult, _ := json.Marshal(expectedJSON.Root.Base)

	result := string(byteResult)
	expected := `{"test":{"stats":{"generic":10,"increment":10},"other":{"thing":"5ms"}}}`

	if result != expected {
		t.Errorf("Result != expected: %v != %v", result, expected)
	}
}
