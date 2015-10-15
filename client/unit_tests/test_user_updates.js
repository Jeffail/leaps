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
		obj.leap_document.id = "testdocument";
		obj.version = 1;
		obj.response_type = "document";
		socket.onmessage({ data : JSON.stringify(obj) });
	};

	var client = new lc();
	client.connect("", socket);

	client.subscribe_event("error", function(err) {
		test.ok(false, "client error: " + JSON.stringify(err));
	});

	client.create_document("test_id", "test_token", "random content");
	// Should now be primed and ready.

	client.on("user", function(user) {
		updates.push(user);
		if ( updates.length < n_loops ) {
			client.update_cursor(updates.length);
		}
	});

	socket.send = function(data) {
		var update = JSON.parse(data);

		socket.onmessage({ data : JSON.stringify({
			response_type: "update",
			user_updates: [ {
				client: {
					user_id    : "test",
					session_id : "test"
				},
				message: {
					active   : true,
					position : update.position,
					content  : update.message
				},
			} ]
		}) });
	};

	client.update_cursor(updates.length);

	client.close();

	test.ok(updates.length === n_loops, "wrong updates count: " + updates.length + " !== " + n_loops);
	for ( var i = 0, l = updates.length; i < l; i++ ) {
		test.ok(updates[i].message.position === i, "wrong position for update: " + updates[i].message.position + " != " + i);
	}

	test.done();
};

/*--------------------------------------------------------------------------------------------------
 */
