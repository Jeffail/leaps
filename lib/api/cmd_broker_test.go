/*
Copyright (c) 2017 Ashley Jeffs

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

package api

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Jeffail/leaps/lib/api/events"
)

//------------------------------------------------------------------------------

type tOutput struct {
	stdout []byte
	stderr []byte
	err    error
}

type mockRunner struct {
	cmds map[string]tOutput
}

func (m mockRunner) CMDRun(cmd string) ([]byte, []byte, error) {
	o, ok := m.cmds[cmd]
	if ok {
		return o.stdout, o.stderr, o.err
	}
	return nil, nil, errors.New("cmd does not exist")
}

//------------------------------------------------------------------------------

func TestBasicCMDBroker(t *testing.T) {
	eBroker := NewCMDBroker([]string{"foo", "bar"}, mockRunner{
		cmds: map[string]tOutput{
			"foo": {stdout: []byte("foo out")},
			"bar": {stderr: []byte("bar err"), err: errors.New("bar failed")},
		},
	}, time.Second, logger, stats)

	dEmitter1 := &dudEmitter{
		reqHandlers: map[string]RequestHandler{},
		resHandlers: map[string]ResponseHandler{},
		sendChan:    make(chan dudSendType),
	}

	go eBroker.NewEmitter("foo1", "bar1", dEmitter1)

	if err := compareGlobalMetadata(dEmitter1, events.GlobalMetadataMessage{
		Metadata: events.MetadataBody{
			Type: events.CMDList,
			Body: events.CMDListMetadataMessage{
				CMDS: []string{"foo", "bar"},
			},
		},
	}); err != nil {
		t.Error(err)
	}

	if _, ok := dEmitter1.reqHandlers[events.GlobalMetadata]; !ok {
		t.Error("Global metadata handler was not bound")
	}
	if dEmitter1.closeHandler == nil {
		t.Error("Close handler was not bound")
	}

	if exp, act := 1, len(eBroker.emitters); exp != act {
		t.Errorf("Wrong count of emitters: %v != %v", exp, act)
	}

	dEmitter1.reqHandlers[events.GlobalMetadata]([]byte(`{
		"metadata": {
			"type": "cmd",
			"body": {
				"cmd": {
					"id": 0
				}
			}
		}
	}`))

	if err := compareGlobalMetadata(dEmitter1, events.GlobalMetadataMessage{
		Metadata: events.MetadataBody{
			Type: events.CMDOutput,
			Body: events.CMDMetadataMessage{
				CMDData: events.CMDData{
					ID:     0,
					Stdout: "foo out",
				},
			},
		},
	}); err != nil {
		t.Error(err)
	}

	dEmitter1.reqHandlers[events.GlobalMetadata]([]byte(`{
		"metadata": {
			"type": "cmd",
			"body": {
				"cmd": {
					"id": 1
				}
			}
		}
	}`))

	if err := compareGlobalMetadata(dEmitter1, events.GlobalMetadataMessage{
		Metadata: events.MetadataBody{
			Type: events.CMDOutput,
			Body: events.CMDMetadataMessage{
				CMDData: events.CMDData{
					ID:     1,
					Stderr: "bar err",
					Error:  "bar failed",
				},
			},
		},
	}); err != nil {
		t.Error(err)
	}

	errExp := events.APIError{
		T:   "ERR_BAD_REQ",
		Err: "CMD Index was out of bounds",
	}

	errAct := dEmitter1.reqHandlers[events.GlobalMetadata]([]byte(`{
		"metadata": {
			"type": "cmd",
			"body": {
				"cmd": {
					"id": 2
				}
			}
		}
	}`))

	if !reflect.DeepEqual(errExp, errAct) {
		t.Errorf("Unexpected result from oob cmd: %v != %v", errExp, errAct)
	}

	dEmitter1.closeHandler()

	if exp, act := 0, len(eBroker.emitters); exp != act {
		t.Errorf("Wrong count of emitters: %v != %v", exp, act)
	}
}

//------------------------------------------------------------------------------
