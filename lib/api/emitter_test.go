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
