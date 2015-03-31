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

/*jshint newcap: false*/

(function() {
"use strict";

/*--------------------------------------------------------------------------------------------------
 */

/* leap_bind_codemirror takes an existing leap_client and uses it to convert a codemirro web editor
 * (http://codemirror.net/) into a live leaps shared editor.
 */
var leap_bind_codemirror = function(leap_client, codemirror_object) {
	this._codemirror = codemirror_object;
	this._leap_client = leap_client;

	this._content = "";
	this._ready = false;
	this._blind_eye_turned = false;

	var binder = this;

	this._codemirror.on('beforeChange', function(instance, e) {
		binder._convert_to_transform.apply(binder, [ e ]);
	});

	this._leap_client.subscribe_event("document", function(doc) {
		binder._content = doc.content;

		binder._blind_eye_turned = true;
		binder._codemirror.getDoc().setValue(doc.content);

		binder._ready = true;
		binder._blind_eye_turned = false;

		binder._pos_interval = setInterval(function() {
			var live_document = binder._codemirror.getDoc();
			var position = live_document.indexFromPos(live_document.getCursor());
			binder._leap_client.update_cursor.apply(binder._leap_client, [ position ]);
		}, leap_client._POSITION_POLL_PERIOD);
	});

	this._leap_client.subscribe_event("transforms", function(transforms) {
		for ( var i = 0, l = transforms.length; i < l; i++ ) {
			binder._apply_transform.apply(binder, [ transforms[i] ]);
		}
	});

	this._leap_client.subscribe_event("disconnect", function() {
		if ( undefined !== binder._pos_interval ) {
			clearTimeout(binder._pos_interval);
		}
	});
};

/* apply_transform, applies a single transform to the codemirror document
 */
leap_bind_codemirror.prototype._apply_transform = function(transform) {
	this._blind_eye_turned = true;

	var live_document = this._codemirror.getDoc();
	var start_position = live_document.posFromIndex(transform.position), end_position = start_position;

	if ( transform.num_delete > 0 ) {
		end_position = live_document.posFromIndex(transform.position + transform.num_delete);
	}

	var insert = "";
	if ( typeof(transform.insert) === "string" && transform.insert.length > 0 ) {
		insert = transform.insert;
	}

	live_document.replaceRange(insert, start_position, end_position);

	this._blind_eye_turned = false;

	this._content = this._leap_client.apply(transform, this._content);

	setTimeout((function() {
		if ( this._content !== this._codemirror.getDoc().getValue() ) {
			this._leap_client._dispatch_event.apply(this._leap_client,
				[ this._leap_client.EVENT_TYPE.ERROR, [
					"Local editor has lost synchronization with server"
				] ]);
		}
	}).bind(this), 0);
};

/* convert_to_transform, takes a codemirror edit event, converts it into a transform and sends it.
 */
leap_bind_codemirror.prototype._convert_to_transform = function(e) {
	if ( this._blind_eye_turned ) {
		return;
	}

	var tform = {};

	var live_document = this._codemirror.getDoc();
	var start_index = live_document.indexFromPos(e.from), end_index = live_document.indexFromPos(e.to);

	tform.position = start_index;
	tform.insert = e.text.join('\n') || "";

	tform.num_delete = end_index - start_index;

	if ( tform.insert.length <= 0 && tform.num_delete <= 0 ) {
		this._leap_client._dispatch_event.apply(this._leap_client,
			[ this._leap_client.EVENT_TYPE.ERROR, [
				"Change resulted in invalid transform"
			] ]);
	}

	this._content = this._leap_client.apply(tform, this._content);
	var err = this._leap_client.send_transform(tform);
	if ( err !== undefined ) {
		this._leap_client._dispatch_event.apply(this._leap_client,
			[ this._leap_client.EVENT_TYPE.ERROR, [
				"Change resulted in invalid transform: " + err
			] ]);
	}

	setTimeout((function() {
		if ( this._content !== this._codemirror.getDoc().getValue() ) {
			this._leap_client._dispatch_event.apply(this._leap_client,
				[ this._leap_client.EVENT_TYPE.ERROR, [
					"Local editor has lost synchronization with server"
				] ]);
		}
	}).bind(this), 0);
};

/*--------------------------------------------------------------------------------------------------
 */

try {
	if ( window.leap_client !== undefined && typeof(window.leap_client) === "function" ) {
		window.leap_client.prototype.bind_codemirror = function(codemirror_object) {
			this._codemirror = new leap_bind_codemirror(this, codemirror_object);
		};
	}
} catch (e) {
}

/*--------------------------------------------------------------------------------------------------
 */

})();
