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

package io

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Jeffail/leaps/lib/api/events"
)

//------------------------------------------------------------------------------

type chanEmitter struct {
	cReceived chan string
	cSent     chan string
}

func (c chanEmitter) ReadJSON(v interface{}) error {
	tito, ok := v.(*struct {
		Type string          `json:"type"`
		Body json.RawMessage `json:"body"`
	})
	if !ok {
		return errors.New("failed assertion")
	}
	select {
	case t, open := <-c.cReceived:
		if !open {
			return errors.New("closed")
		}
		tito.Type = t
	}
	return nil
}

func (c chanEmitter) WriteJSON(v interface{}) error {
	tito, ok := v.(struct {
		Type string      `json:"type"`
		Body interface{} `json:"body"`
	})
	if !ok {
		return errors.New("failed assertion")
	}
	c.cSent <- tito.Type
	return nil
}

func (c chanEmitter) Close() error {
	close(c.cReceived)
	close(c.cSent)
	return nil
}

func TestBasicJSONEmitter(t *testing.T) {
	c := chanEmitter{
		cReceived: make(chan string),
		cSent:     make(chan string),
	}
	e := NewJSONEmitter(c)

	readyForClose := false
	closedChan := make(chan struct{})
	e.OnClose(func() {
		if !readyForClose {
			t.Errorf("Closed prematurely")
		} else {
			close(closedChan)
		}
	})

	e.OnReceive("test1", func(body []byte) events.TypedError {
		if err := e.Send("test2", nil); err != nil {
			t.Error(err)
		}
		return nil
	})

	go e.ListenAndEmit()

	select {
	case c.cReceived <- "test1":
	case <-time.After(time.Second):
		t.Errorf("Timed out waiting on receive")
	}
	select {
	case mType := <-c.cSent:
		if mType != "test2" {
			t.Errorf("Wrong message type: %v != %v", mType, "test2")
		}
	case <-time.After(time.Second):
		t.Errorf("Timed out waiting on send")
	}

	readyForClose = true
	close(c.cReceived)
	select {
	case <-closedChan:
	case <-time.After(time.Second):
		t.Errorf("Timed out waiting on close")
	}
}

func TestUnrecognisedJSONEmitter(t *testing.T) {
	c := chanEmitter{
		cReceived: make(chan string),
		cSent:     make(chan string),
	}
	e := NewJSONEmitter(c)

	readyForClose := false
	closedChan := make(chan struct{})
	e.OnClose(func() {
		if !readyForClose {
			t.Errorf("Closed prematurely")
		} else {
			close(closedChan)
		}
	})

	go e.ListenAndEmit()

	select {
	case c.cReceived <- "test1":
	case <-time.After(time.Second * 10):
		t.Errorf("Timed out waiting on receive")
	}
	select {
	case mType := <-c.cSent:
		if mType != "error" {
			t.Errorf("Wrong message type: %v != %v", mType, "error")
		}
	case <-time.After(time.Second * 10):
		t.Errorf("Timed out waiting on send")
	}

	readyForClose = true
	if err := e.Close(); err != nil {
		t.Error(err)
	}
	select {
	case <-closedChan:
	case <-time.After(time.Second * 10):
		t.Errorf("Timed out waiting on close")
	}
	select {
	case <-c.cReceived:
	case <-time.After(time.Second * 10):
		t.Errorf("Timed out waiting on propagated close")
	}
}

func TestErrorEmitter(t *testing.T) {
	c := chanEmitter{
		cReceived: make(chan string),
		cSent:     make(chan string),
	}
	e := NewJSONEmitter(c)

	readyForClose := false
	closedChan := make(chan struct{})
	e.OnClose(func() {
		if !readyForClose {
			t.Errorf("Closed prematurely")
		} else {
			close(closedChan)
		}
	})

	e.OnReceive("test1", func(body []byte) events.TypedError {
		return events.NewAPIError("test2", "test3")
	})

	go e.ListenAndEmit()

	select {
	case c.cReceived <- "test1":
	case <-time.After(time.Second * 10):
		t.Errorf("Timed out waiting on receive")
	}
	select {
	case mType := <-c.cSent:
		if mType != "error" {
			t.Errorf("Wrong message type: %v != %v", mType, "error")
		}
	case <-time.After(time.Second * 10):
		t.Errorf("Timed out waiting on send")
	}

	readyForClose = true
	close(c.cReceived)
	select {
	case <-closedChan:
	case <-time.After(time.Second * 10):
		t.Errorf("Timed out waiting on close")
	}
}

func TestSendEmittingJSONEmitter(t *testing.T) {
	c := chanEmitter{
		cReceived: make(chan string),
		cSent:     make(chan string),
	}
	e := NewJSONEmitter(c)

	readyForClose := false
	closedChan := make(chan struct{})
	e.OnClose(func() {
		if !readyForClose {
			t.Errorf("Closed prematurely")
		} else {
			close(closedChan)
		}
	})

	body1, body2 := "test1", "test2"

	e.OnSend("test1", func(body interface{}) bool {
		if !reflect.DeepEqual(body, body1) {
			t.Errorf("Wrong body from send event: %v != %v", body1, body)
		}
		return true
	})
	e.OnSend("test2", func(body interface{}) bool {
		if !reflect.DeepEqual(body, body2) {
			t.Errorf("Wrong body from send event: %v != %v", body2, body)
		}
		return false // Skip test2 message
	})

	go e.ListenAndEmit()

	go func() {
		e.Send("test2", body2)
		e.Send("test1", body1)
	}()

	select {
	case mType := <-c.cSent:
		if mType != "test1" {
			t.Errorf("Wrong message type: %v != %v", mType, "test1")
		}
	case <-time.After(time.Second * 10):
		t.Errorf("Timed out waiting on send")
	}

	readyForClose = true
	close(c.cReceived)
	select {
	case <-closedChan:
	case <-time.After(time.Second * 10):
		t.Errorf("Timed out waiting on close")
	}
}

//------------------------------------------------------------------------------
