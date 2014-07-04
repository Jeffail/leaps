#!/usr/bin/node
// Requires 'ws'

var client = new (require('../leapclient/leapclient.js').client)();
var ws     = require('ws');

var mock_socket = function() {
	var socket = new ws('ws://localhost:8080/socket');
	this.readyState = 0;

	this.send = function(data) {
		socket.send.apply(socket, [ data ]);
	}

	this.close = function() {
		socket.close.apply(socket, [ arguments ]);
	}

	var _mock = this;

	socket.on('message', function(data) {
		if ( typeof(_mock.onmessage) === "function" ) {
			_mock.onmessage.apply(_mock, [ { data: data } ]);
		}
	});

	socket.on('open', function(data) {
		_mock.readyState = 1;
		if ( typeof(_mock.onopen) === "function" ) {
			_mock.onopen.apply(_mock);
		}
	});

	socket.on('close', function(data) {
		if ( typeof(_mock.onclose) === "function" ) {
			_mock.onclose.apply(_mock);
		}
	});
};

var spammer;
var socket = new mock_socket();

client.on("connect", function() {
	console.log("joining document...");
	var err = client.create_document("test", "test_document", "hello world 123");
	if ( err !== undefined ) {
		console.error(JSON.stringify(err));
	}
});

client.on("document", function() {
	console.log("spamming...");
	spammer = setInterval(function() {
		client.send_transform({
			position: 0,
			insert: "hello world",
			num_delete: 11
		});
	}, 10);
});


client.on("error", function(err) {
	console.error("Error" + JSON.stringify(err));
});

client.on("disconnect", function() {
	console.log("disconnected");
	clearInterval(spammer);
});

console.log("connecting...");
client.connect("", socket);
