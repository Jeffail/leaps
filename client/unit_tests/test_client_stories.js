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

var fs   = require('fs'),
    path = require('path'),
    lc   = require('../leapclient').client,
    la   = require('../leapclient').apply;

var client_stories_text = fs.readFileSync(
		path.resolve(__dirname, "./../../test/stories/", "./client_stories.js"), "utf8");

var stories = JSON.parse(client_stories_text).client_stories;

var run_story = function(story, test) {
	"use strict";

	var content = story.content;
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

	client.create_document("test", "test", story.content);

	for ( var i = 0, l = story.epochs.length; i < l; i++ ) {
		var epoch_sends = story.epochs[i].send;
		var epoch_receives = story.epochs[i].receive;

		client.clear_subscribers("transforms");
		client.on("transforms", function(tforms) {
			for ( var tn = 0, ln = tforms.length; tn < ln; tn++ ) {
				content = la(tforms[tn], content);
			}
			if ( epoch_sends.length > 0 ) {
				var tform = epoch_sends.shift();
				content = la(tform, content);
				var err = client.send_transform(tform);
				test.ok(err === undefined, "story '" + story.name + "' epoch " + i + ": " + err);
			} else if ( epoch_receives.length > 0 ) {
				socket.onmessage({ data : JSON.stringify(epoch_receives.shift())});
			}
		});

		socket.send = function() {
			if ( epoch_receives.length > 0 ) {
				socket.onmessage({ data : JSON.stringify(epoch_receives.shift()) });
			}
		};

		if ( epoch_sends.length > 0 ) {
			var tmp = epoch_sends.shift();
			content = la(tmp, content);
			var err = client.send_transform(tmp);
			test.ok(err === undefined, "story '" + story.name + "' epoch " + i + ": " + err);
		} else if ( epoch_receives.length > 0 ) {
			socket.onmessage({ data : JSON.stringify(epoch_receives.shift())});
		}

		while ( epoch_receives.length > 0 ) {
			socket.onmessage({ data : JSON.stringify(epoch_receives.shift())});
		}

		test.ok(epoch_sends.length === 0,
				"story '" + story.name + "' epoch " + i + ": epoch_sends (" + epoch_sends.length + ") != 0");

		test.ok(content === story.epochs[i].result,
				"story '" + story.name + "' epoch " + i + ": " + content + " != " + story.epochs[i].result);
	}

	client.close();
	test.ok(content === story.result,
			"story " + story.name + ": " + content + " != " + story.result);
};

module.exports = function(test) {
	"use strict";

	test.ok(stories.length > 0, "no stories found");

	// All stories are actually synchronous
	for ( var i = 0, l = stories.length; i < l; i++ ) {
		run_story(stories[i], test);
	}
	test.ok(stories !== undefined, "test for stories");
	test.done();
};

/*--------------------------------------------------------------------------------------------------
 */
