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

var lc = require('../leapclient').client;

module.exports = function(test) {
	"use strict";

	var n_loops = 10;
	var updates = [];

	var socket = { readyState : 1 };

	socket.close = function() {};

	// First send response should be the same doc, emulating creation
	socket.send = function(data) {
		var obj = JSON.parse(data);
		obj.body.document.id = "testdocument";
		obj.body.document.content = "random content";
		obj.body.document.version = 1;
		obj.type = "subscribe";
		socket.onmessage({ data : JSON.stringify(obj) });
	};

	var client = new lc();
	client.connect("", socket);

	client.on("error", function(err) {
		test.ok(false, "client error: " + JSON.stringify(err));
	});

	client.subscribe("testdocument");
	// Should now be primed and ready.

	client.on("metadata", function(body) {
		updates.push(body);
		if ( updates.length < n_loops ) {
			let err = client.send_metadata("testdocument", updates.length);
			if ( typeof err === 'string' ) {
				test.ok(false, err);
			}
		}
	});

	socket.send = function(data) {
		socket.onmessage({ data: data });
	};

	let err = client.send_metadata("testdocument", updates.length);
	if ( typeof err === 'string' ) {
		test.ok(false, err);
	}

	client.close();

	test.ok(updates.length === n_loops, "wrong updates count: " + updates.length + " !== " + n_loops);
	for ( var i = 0, l = updates.length; i < l; i++ ) {
		test.ok(updates[i].metadata === i, "wrong position for update: " + updates[i].metadata + " != " + i);
	}

	test.done();
};

/*--------------------------------------------------------------------------------------------------
 */
