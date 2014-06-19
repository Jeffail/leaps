// npm install ws

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

var con_err = lc.connect('ws://localhost:8080', mock_websocket);
if ( con_err !== undefined ) {
	console.log(con_err);
}
