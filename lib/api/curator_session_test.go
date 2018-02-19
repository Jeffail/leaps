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
	"reflect"
	"testing"
	"time"

	"github.com/Jeffail/leaps/lib/api/events"
	"github.com/Jeffail/leaps/lib/binder"
	"github.com/Jeffail/leaps/lib/text"
)

//------------------------------------------------------------------------------

func TestBasicCuratorSession(t *testing.T) {
	dCurator := &dudCurator{make(map[string]*dudPortal), make(map[string]struct{}), make(chan struct{})}
	dEmitter := &dudEmitter{
		make(map[string]RequestHandler),
		make(map[string]ResponseHandler),
		nil, make(chan dudSendType, 1),
	}

	dCurator.dudDocs["testdoc1"] = struct{}{}

	NewCuratorSession("testUser1", "nope", dEmitter, dCurator, time.Second, logger, stats)

	// Send ping
	if err := dEmitter.reqHandlers[events.Ping](nil); err != nil {
		t.Error(err)
	}

	// Receive pong
	select {
	case d := <-dEmitter.sendChan:
		if exp, act := events.Pong, d.Type; exp != act {
			t.Errorf("Wrong type returned: %v != %v", exp, act)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for pong")
	}

	// Send subscribe request
	if err := dEmitter.reqHandlers[events.Subscribe](
		[]byte(`{"document":{"id":"testdoc1"}}`),
	); err != nil {
		t.Error(err)
	}

	// Ensure subscription is propagated back up to emitter
	select {
	case d := <-dEmitter.sendChan:
		if exp, act := events.Subscribe, d.Type; exp != act {
			t.Errorf("Wrong event type returned: %v != %v", exp, act)
		}
		if bodyObj, ok := d.Body.(events.SubscriptionMessage); ok {
			if exp, act := "testdoc1", bodyObj.Document.ID; exp != act {
				t.Errorf("Wrong event body returned: %v != %v", exp, act)
			}
		} else {
			t.Errorf("Wrong type of body: %T", d.Body)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for subscriber send")
	}

	// Expect error when subscribing again
	if err := dEmitter.reqHandlers[events.Subscribe](
		[]byte(`{"document":{"id":"testdoc1"}}`),
	); err == nil {
		t.Error("Expected error from double subscribe")
	} else if exp, act := events.ErrExistingSub, err.Type(); exp != act {
		t.Errorf("Wrong error type returned: %v != %v", exp, act)
	}

	// Send transform through binder
	tformMsg := events.TransformsMessage{
		Document: events.DocumentStripped{ID: "testdoc1"},
		Transforms: []text.OTransform{
			{Insert: "hello world"},
		},
	}

	select {
	case dCurator.dudPortals["testdoc1"].tChan <- tformMsg.Transforms[0]:
	case <-time.After(time.Second):
		t.Error("Timed out waiting for transform send binder")
	}

	// Ensure transform is propagated back up to emitter
	select {
	case sent := <-dEmitter.sendChan:
		if exp, act := events.Transforms, sent.Type; exp != act {
			t.Errorf("Wrong event type returned: %v != %v", exp, act)
		}
		if !reflect.DeepEqual(tformMsg, sent.Body) {
			t.Errorf("Wrong event body returned: %v != %v", tformMsg, sent.Body)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for transform send emitter")
	}

	// Send metadata through binder
	metaMsg := events.MetadataMessage{
		Document: events.DocumentStripped{ID: "testdoc1"},
		Client:   "foo",
		Metadata: "bar",
	}

	select {
	case dCurator.dudPortals["testdoc1"].mChan <- binder.ClientMetadata{Client: metaMsg.Client, Metadata: metaMsg.Metadata}:
	case <-time.After(time.Second):
		t.Error("Timed out waiting for metadata send binder")
	}

	select {
	case sent := <-dEmitter.sendChan:
		if exp, act := events.Metadata, sent.Type; exp != act {
			t.Errorf("Wrong event type returned: %v != %v", exp, act)
		}
		if !reflect.DeepEqual(metaMsg, sent.Body) {
			t.Errorf("Wrong event body returned: %v != %v", metaMsg, sent.Body)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for metadata send emitter")
	}

	// Close binder and wait for unsubscription message
	dCurator.dudPortals["testdoc1"].Exit(time.Second)

	select {
	case sent := <-dEmitter.sendChan:
		if exp, act := events.Unsubscribe, sent.Type; exp != act {
			t.Errorf("Wrong event type returned: %v != %v", exp, act)
		}
		exp := events.UnsubscriptionMessage{
			Document: events.DocumentStripped{ID: "testdoc1"},
		}
		if !reflect.DeepEqual(exp, sent.Body) {
			t.Errorf("Wrong event body returned: %v != %v", exp, sent.Body)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for unsub send emitter")
	}
}

func TestCuratorSessionErrors(t *testing.T) {
	dCurator := &dudCurator{make(map[string]*dudPortal), make(map[string]struct{}), make(chan struct{})}
	dEmitter := &dudEmitter{
		make(map[string]RequestHandler),
		make(map[string]ResponseHandler),
		nil, make(chan dudSendType, 1),
	}

	NewCuratorSession("testUser1", "nope", dEmitter, dCurator, time.Second, logger, stats)

	// Send ping
	if err := dEmitter.reqHandlers[events.Ping](nil); err != nil {
		t.Error(err)
	}

	// Receive pong
	select {
	case d := <-dEmitter.sendChan:
		if exp, act := events.Pong, d.Type; exp != act {
			t.Errorf("Wrong type returned: %v != %v", exp, act)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for pong")
	}

	// Send subscribe request for unexisting doc
	if err := dEmitter.reqHandlers[events.Subscribe](
		[]byte(`{"document":{"id":"testdoc1"}}`),
	); err == nil {
		t.Error("Expected error from failed subscribe")
	} else if exp, act := events.ErrSubscribe, err.Type(); exp != act {
		t.Errorf("Wrong error type returned: %v != %v", exp, act)
	}

	// Send subscribe request with mangled JSON
	if err := dEmitter.reqHandlers[events.Subscribe](
		[]byte(`{"}}fghsfjdghfk`),
	); err == nil {
		t.Error("Expected error from failed subscribe")
	} else if exp, act := events.ErrBadJSON, err.Type(); exp != act {
		t.Errorf("Wrong error type returned: %v != %v", exp, act)
	}

	// Send transform without sub
	if err := dEmitter.reqHandlers[events.Transform](
		[]byte(`{"transform":{}}`),
	); err == nil {
		t.Error("Expected error from failed transform")
	} else if exp, act := events.ErrNoSub, err.Type(); exp != act {
		t.Errorf("Wrong error type returned: %v != %v", exp, act)
	}

	// Send metadata without sub
	if err := dEmitter.reqHandlers[events.Metadata](
		[]byte(`{"metadata":{}}`),
	); err == nil {
		t.Error("Expected error from failed metadata")
	} else if exp, act := events.ErrNoSub, err.Type(); exp != act {
		t.Errorf("Wrong error type returned: %v != %v", exp, act)
	}
}

func TestCuratorSessionUnsub(t *testing.T) {
	dCurator := &dudCurator{make(map[string]*dudPortal), make(map[string]struct{}), make(chan struct{})}
	dEmitter := &dudEmitter{
		make(map[string]RequestHandler),
		make(map[string]ResponseHandler),
		nil, make(chan dudSendType, 1),
	}

	dCurator.dudDocs["testdoc1"] = struct{}{}
	dCurator.dudDocs["testdoc2"] = struct{}{}

	NewCuratorSession("testUser1", "nope", dEmitter, dCurator, time.Second, logger, stats)

	// Send subscribe request
	if err := dEmitter.reqHandlers[events.Subscribe](
		[]byte(`{"document":{"id":"testdoc1"}}`),
	); err != nil {
		t.Error(err)
	}

	// Ensure subscription is propagated back up to emitter
	select {
	case d := <-dEmitter.sendChan:
		if exp, act := events.Subscribe, d.Type; exp != act {
			t.Errorf("Wrong event type returned: %v != %v", exp, act)
		}
		if bodyObj, ok := d.Body.(events.SubscriptionMessage); ok {
			if exp, act := "testdoc1", bodyObj.Document.ID; exp != act {
				t.Errorf("Wrong event body returned: %v != %v", exp, act)
			}
		} else {
			t.Errorf("Wrong type of body: %T", d.Body)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for subscriber send")
	}

	// Send unsubscribe request
	if err := dEmitter.reqHandlers[events.Unsubscribe](
		[]byte(`{"document":{"id":"testdoc1"}}`),
	); err != nil {
		t.Error(err)
	}

	// Ensure unsubscription is propagated back up to emitter
	select {
	case d := <-dEmitter.sendChan:
		if exp, act := events.Unsubscribe, d.Type; exp != act {
			t.Errorf("Wrong event type returned: %v != %v", exp, act)
		}
		if bodyObj, ok := d.Body.(events.UnsubscriptionMessage); ok {
			if exp, act := "testdoc1", bodyObj.Document.ID; exp != act {
				t.Errorf("Wrong event body returned: %v != %v", exp, act)
			}
		} else {
			t.Errorf("Wrong type of body: %T", d.Body)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for unsubscriber send")
	}

	// Send new subscribe request
	if err := dEmitter.reqHandlers[events.Subscribe](
		[]byte(`{"document":{"id":"testdoc2"}}`),
	); err != nil {
		t.Error(err)
	}

	// Ensure subscription is propagated back up to emitter
	select {
	case d := <-dEmitter.sendChan:
		if exp, act := events.Subscribe, d.Type; exp != act {
			t.Errorf("Wrong event type returned: %v != %v", exp, act)
		}
		if bodyObj, ok := d.Body.(events.SubscriptionMessage); ok {
			if exp, act := "testdoc2", bodyObj.Document.ID; exp != act {
				t.Errorf("Wrong event body returned: %v != %v", exp, act)
			}
		} else {
			t.Errorf("Wrong type of body: %T", d.Body)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for subscriber send")
	}

	// Close binder and wait for unsubscription message
	dCurator.dudPortals["testdoc2"].Exit(time.Second)

	select {
	case sent := <-dEmitter.sendChan:
		if exp, act := events.Unsubscribe, sent.Type; exp != act {
			t.Errorf("Wrong event type returned: %v != %v", exp, act)
		}
		exp := events.UnsubscriptionMessage{
			Document: events.DocumentStripped{ID: "testdoc2"},
		}
		if !reflect.DeepEqual(exp, sent.Body) {
			t.Errorf("Wrong event body returned: %v != %v", exp, sent.Body)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for unsub send emitter")
	}
}

func TestCuratorSessionEmitterClose(t *testing.T) {
	dCurator := &dudCurator{make(map[string]*dudPortal), make(map[string]struct{}), make(chan struct{})}
	dEmitter := &dudEmitter{
		make(map[string]RequestHandler),
		make(map[string]ResponseHandler),
		nil, make(chan dudSendType, 1),
	}

	dCurator.dudDocs["testdoc1"] = struct{}{}

	NewCuratorSession("testUser1", "nope", dEmitter, dCurator, time.Second, logger, stats)

	// Send subscribe request
	if err := dEmitter.reqHandlers[events.Subscribe](
		[]byte(`{"document":{"id":"testdoc1"}}`),
	); err != nil {
		t.Error(err)
	}

	// Ensure subscription is propagated back up to emitter
	select {
	case d := <-dEmitter.sendChan:
		if exp, act := events.Subscribe, d.Type; exp != act {
			t.Errorf("Wrong event type returned: %v != %v", exp, act)
		}
		if bodyObj, ok := d.Body.(events.SubscriptionMessage); ok {
			if exp, act := "testdoc1", bodyObj.Document.ID; exp != act {
				t.Errorf("Wrong event body returned: %v != %v", exp, act)
			}
		} else {
			t.Errorf("Wrong type of body: %T", d.Body)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for subscriber send")
	}

	// Trigger OnClose event handler
	dEmitter.closeHandler()

	select {
	case _, open := <-dCurator.dudPortals["testdoc1"].closedChan:
		if open {
			t.Error("Appears binder is still open after closed handler")
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for metadata")
	}
}

func TestCuratorSessionSending(t *testing.T) {
	dCurator := &dudCurator{make(map[string]*dudPortal), make(map[string]struct{}), make(chan struct{})}
	dEmitter := &dudEmitter{
		make(map[string]RequestHandler),
		make(map[string]ResponseHandler),
		nil, make(chan dudSendType, 1),
	}

	dCurator.dudDocs["testdoc1"] = struct{}{}

	NewCuratorSession("testUser1", "nope", dEmitter, dCurator, time.Second, logger, stats)

	// Send subscribe request
	if err := dEmitter.reqHandlers[events.Subscribe](
		[]byte(`{"document":{"id":"testdoc1"}}`),
	); err != nil {
		t.Error(err)
	}

	// Ensure subscription is propagated back up to emitter
	select {
	case d := <-dEmitter.sendChan:
		if exp, act := events.Subscribe, d.Type; exp != act {
			t.Errorf("Wrong event type returned: %v != %v", exp, act)
		}
		if bodyObj, ok := d.Body.(events.SubscriptionMessage); ok {
			if exp, act := "testdoc1", bodyObj.Document.ID; exp != act {
				t.Errorf("Wrong event body returned: %v != %v", exp, act)
			}
		} else {
			t.Errorf("Wrong type of body: %T", d.Body)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for subscriber send")
	}

	go func() {
		// Send transform
		if err := dEmitter.reqHandlers[events.Transform](
			[]byte(`{"document":{"id":"testdoc1"},"transform":{"insert":"hello world"}}`),
		); err != nil {
			t.Errorf("Unexpected error from transform: %v", err)
		}
	}()

	select {
	case tform := <-dCurator.dudPortals["testdoc1"].sentTChan:
		if exp, act := "hello world", tform.Insert; exp != act {
			t.Errorf("wrong insert from sent transform: %v != %v", exp, act)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for transform")
	}

	select {
	case sent := <-dEmitter.sendChan:
		if exp, act := events.Correction, sent.Type; exp != act {
			t.Errorf("Wrong event type returned: %v != %v", exp, act)
		}
		exp := events.CorrectionMessage{
			Document:   events.DocumentStripped{ID: "testdoc1"},
			Correction: events.TformCorrection{Version: 10},
		}
		if !reflect.DeepEqual(exp, sent.Body) {
			t.Errorf("Wrong event body returned: %v != %v", exp, sent.Body)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for correction")
	}

	go func() {
		// Send metadata
		if err := dEmitter.reqHandlers[events.Metadata](
			[]byte(`{"document":{"id":"testdoc1"},"metadata":"hello world"}`),
		); err != nil {
			t.Errorf("Unexpected error from metadata: %v", err)
		}
	}()

	select {
	case m := <-dCurator.dudPortals["testdoc1"].sentMChan:
		if exp, act := "hello world", m.Metadata.(string); exp != act {
			t.Errorf("wrong insert from sent metadata: %v != %v", exp, act)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for metadata")
	}

	// Close binder and wait for unsubscription message
	dCurator.dudPortals["testdoc1"].Exit(time.Second)

	select {
	case sent := <-dEmitter.sendChan:
		if exp, act := events.Unsubscribe, sent.Type; exp != act {
			t.Errorf("Wrong event type returned: %v != %v", exp, act)
		}
		exp := events.UnsubscriptionMessage{
			Document: events.DocumentStripped{ID: "testdoc1"},
		}
		if !reflect.DeepEqual(exp, sent.Body) {
			t.Errorf("Wrong event body returned: %v != %v", exp, sent.Body)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for unsub send emitter")
	}
}

func TestCuratorSessionWithBadData(t *testing.T) {
	dCurator := &dudCurator{make(map[string]*dudPortal), make(map[string]struct{}), make(chan struct{})}
	dEmitter := &dudEmitter{
		make(map[string]RequestHandler),
		make(map[string]ResponseHandler),
		nil, make(chan dudSendType, 1),
	}

	dCurator.dudDocs["testdoc1"] = struct{}{}

	NewCuratorSession("testUser1", "nope", dEmitter, dCurator, time.Second, logger, stats)

	// Send subscribe request
	if err := dEmitter.reqHandlers[events.Subscribe](
		[]byte(`{"document":{"id":"testdoc1"}}`),
	); err != nil {
		t.Error(err)
	}

	// Ensure subscription is propagated back up to emitter
	select {
	case d := <-dEmitter.sendChan:
		if exp, act := events.Subscribe, d.Type; exp != act {
			t.Errorf("Wrong event type returned: %v != %v", exp, act)
		}
		if bodyObj, ok := d.Body.(events.SubscriptionMessage); ok {
			if exp, act := "testdoc1", bodyObj.Document.ID; exp != act {
				t.Errorf("Wrong event body returned: %v != %v", exp, act)
			}
		} else {
			t.Errorf("Wrong type of body: %T", d.Body)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for subscriber send")
	}

	// Send transform with bad JSON
	if err := dEmitter.reqHandlers[events.Transform](
		[]byte(`{"tran6457''][]#[]`),
	); err == nil {
		t.Error("Expected error from failed transform")
	} else if exp, act := events.ErrBadJSON, err.Type(); exp != act {
		t.Errorf("Wrong error type returned: %v != %v", exp, act)
	}

	// Send metadata with bad JSON
	if err := dEmitter.reqHandlers[events.Metadata](
		[]byte(`{"me}fdgjhs453'#[]`),
	); err == nil {
		t.Error("Expected error from failed metadata")
	} else if exp, act := events.ErrBadJSON, err.Type(); exp != act {
		t.Errorf("Wrong error type returned: %v != %v", exp, act)
	}

	dCurator.dudPortals["testdoc1"].Exit(time.Second)
}

//------------------------------------------------------------------------------
