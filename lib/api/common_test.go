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
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/Jeffail/leaps/lib/api/events"
	"github.com/Jeffail/leaps/lib/binder"
	"github.com/Jeffail/leaps/lib/store"
	"github.com/Jeffail/leaps/lib/text"
	"github.com/Jeffail/leaps/lib/util/service/log"
	"github.com/Jeffail/leaps/lib/util/service/metrics"
)

//------------------------------------------------------------------------------

var logger, stats = func() (log.Modular, metrics.Type) {
	logConf := log.NewLoggerConfig()
	logConf.LogLevel = "OFF"
	return log.NewLogger(os.Stdout, logConf), metrics.DudType{}
}()

//------------------------------------------------------------------------------

type dudPortal struct {
	clientMetadata interface{}
	id             string

	closedChan chan struct{}

	tChan chan text.OTransform
	mChan chan binder.ClientMetadata

	sentTChan chan text.OTransform
	sentMChan chan binder.ClientMetadata
}

func (d *dudPortal) ClientMetadata() interface{} { return d.clientMetadata }
func (d *dudPortal) BaseVersion() int            { return 0 }
func (d *dudPortal) ReleaseDocument()            {}
func (d *dudPortal) Document() store.Document {
	return store.Document{
		ID:      d.id,
		Content: "",
	}
}
func (d *dudPortal) TransformReadChan() <-chan text.OTransform      { return d.tChan }
func (d *dudPortal) MetadataReadChan() <-chan binder.ClientMetadata { return d.mChan }
func (d *dudPortal) SendMetadata(metadata interface{}) {
	d.sentMChan <- struct {
		Client   interface{} `json:"client"`
		Metadata interface{} `json:"metadata"`
	}{
		d.ClientMetadata(),
		metadata,
	}
}
func (d *dudPortal) SendTransform(ot text.OTransform, timeout time.Duration) (int, error) {
	select {
	case d.sentTChan <- ot:
	case <-time.After(timeout):
		return 0, errors.New("Timed out")
	}
	return 10, nil
}
func (d *dudPortal) Exit(timeout time.Duration) {
	close(d.closedChan)
	close(d.tChan)
	close(d.mChan)
}

//------------------------------------------------------------------------------

type dudCurator struct {
	dudPortals map[string]*dudPortal
	dudDocs    map[string]struct{}
	closeChan  chan struct{}
}

func (d *dudCurator) EditDocument(
	userMetadata interface{}, token, documentID string, timeout time.Duration,
) (binder.Portal, error) {
	if _, ok := d.dudDocs[documentID]; ok {
		p := &dudPortal{
			clientMetadata: userMetadata,
			id:             documentID,
			closedChan:     make(chan struct{}),
			tChan:          make(chan text.OTransform),
			mChan:          make(chan binder.ClientMetadata),
			sentTChan:      make(chan text.OTransform),
			sentMChan:      make(chan binder.ClientMetadata),
		}
		d.dudPortals[documentID] = p
		return p, nil
	}
	return nil, errors.New("Not found")
}

func (d *dudCurator) ReadDocument(
	userMetadata interface{}, token, documentID string, timeout time.Duration,
) (binder.Portal, error) {
	return nil, errors.New("Not found")
}

func (d *dudCurator) CreateDocument(
	userMetadata interface{}, token string, document store.Document, timeout time.Duration,
) (binder.Portal, error) {
	return nil, errors.New("Not allowed")
}

func (d *dudCurator) Close() {}

//------------------------------------------------------------------------------

func compareGlobalMetadata(em *dudEmitter, compareTo interface{}) error {
	select {
	case event := <-em.sendChan:
		if exp, act := events.GlobalMetadata, event.Type; exp != act {
			return fmt.Errorf("Wrong event broadcast: %v != %v", exp, act)
		}
		if metaBody, ok := event.Body.(events.GlobalMetadataMessage); ok {
			if !reflect.DeepEqual(compareTo, metaBody) {
				return fmt.Errorf("Wrong GlobalMetadata body: %v != %v", compareTo, metaBody)
			}
		} else {
			return fmt.Errorf("Wrong type: %T", metaBody)
		}
	case <-time.After(time.Second * 2):
		return errors.New("Timed out waiting for global metadata message")
	}
	return nil
}

func compareError(em *dudEmitter, compareTo interface{}) error {
	select {
	case event := <-em.sendChan:
		if exp, act := events.Error, event.Type; exp != act {
			return fmt.Errorf("Wrong event broadcast: %v != %v", exp, act)
		}
		if metaBody, ok := event.Body.(events.ErrorMessage); ok {
			if !reflect.DeepEqual(compareTo, metaBody) {
				return fmt.Errorf("Wrong APIError body: %v != %v", compareTo, metaBody)
			}
		} else {
			return fmt.Errorf("Wrong type: %T", metaBody)
		}
	case <-time.After(time.Second * 2):
		return errors.New("Timed out waiting for error message")
	}
	return nil
}

//------------------------------------------------------------------------------

type dudSendType struct {
	Type string
	Body interface{}
}

type dudEmitter struct {
	reqHandlers  map[string]RequestHandler
	resHandlers  map[string]ResponseHandler
	closeHandler EventHandler

	sendChan chan dudSendType
}

func (d *dudEmitter) OnClose(eventHandler EventHandler) {
	d.closeHandler = eventHandler
}

func (d *dudEmitter) OnReceive(reqType string, handler RequestHandler) {
	d.reqHandlers[reqType] = handler
}

func (d *dudEmitter) OnSend(resType string, handler ResponseHandler) {
	d.resHandlers[resType] = handler
}

func (d *dudEmitter) Send(resType string, body interface{}) error {
	if d.sendChan != nil {
		d.sendChan <- dudSendType{resType, body}
	}
	return nil
}

//------------------------------------------------------------------------------
