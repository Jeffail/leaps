/*
Copyright (c) 2014 Ashley Jeffs

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

/*--------------------------------------------------------------------------------------------------
 */

var _int_websocket = require('ws')
  , leap_client    = require('./leapclient');

var mock_websocket = function(addr) {
	var ws = new _int_websocket(addr);

	this.onerror = this.onopen = this.onclose = this.onmessage = null;

	this.readyState = 0;

	var _mock_sock = this;

	ws.on('open', function() {
		_mock_sock.readyState = 1;
		if ( typeof(_mock_sock.onopen) === 'function' ) {
			_mock_sock.onopen.apply(null, arguments);
		}
	});
	ws.on('close', function() {
		_mock_sock.readyState = 3;
		if ( typeof(_mock_sock.onclose) === 'function' ) {
			_mock_sock.onclose.apply(null, arguments);
		}
	});
	ws.on('error', function() {
		if ( typeof(_mock_sock.onerror) === 'function' ) {
			_mock_sock.onerror.apply(null, arguments);
		}
	});
	ws.on('message', function(text) {
		if ( typeof(_mock_sock.onmessage) === 'function' ) {
			_mock_sock.onmessage({ data : text });
		}
	});

	this.close = function() {
		ws.close.apply(ws, arguments);
	}

	this.send = function() {
		ws.send.apply(ws, arguments);
	}
};

var lc = new leap_client();

lc.on_error = function(err) {
	console.trace(err);
};

lc.on_connect = function() {
	var err = lc.join_document("testste");
	if ( err !== undefined ) {
		console.log(err);
	}
};

var con_err = lc.connect('ws://localhost:8080/leapsocket', mock_websocket);
if ( con_err !== undefined ) {
	console.log(con_err);
}
