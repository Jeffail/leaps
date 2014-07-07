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

/* leap_bind_ace_editor takes an existing leap_client and uses it to convert an Ace web editor
 * (http://ace.c9.io) into a live leaps shared editor.
 */
var leap_bind_ace_editor = function(leap_client, ace_editor) {
	"use strict";

	this._ace = ace_editor;
	this._leap_client = leap_client;

	this._content = "";
	this._ready = false;
	this._blind_eye_turned = false;

	this._ace.setReadOnly(true);

	var binder = this;

	this._ace.getSession().on('change', function(e) {
		binder._convert_to_transform.apply(binder, [ e ]);
	});

	this._leap_client.subscribe_event("document", function(doc) {
		binder._content = doc.content;

		binder._blind_eye_turned = true;
		binder._ace.setValue(doc.content);
		binder._ace.setReadOnly(false);
		binder._ace.clearSelection();

		binder._ready = true;
		binder._blind_eye_turned = false;
	});

	this._leap_client.subscribe_event("transforms", function(transforms) {
		for ( var i = 0, l = transforms.length; i < l; i++ ) {
			binder._apply_transform.apply(binder, [ transforms[i] ]);
		}
	});

	this._leap_client.subscribe_event("disconnect", function() {
		binder._ace.setReadOnly(true);
	});
};

/* apply_transform, applies a single transform to the ace document.
 */
leap_bind_ace_editor.prototype._apply_transform = function(transform) {
	"use strict";

	this._blind_eye_turned = true;

	var edit_session = this._ace.getSession();
	var live_document = edit_session.getDocument();

	var position = live_document.indexToPosition(transform.position, 0);

	if ( transform.num_delete > 0 ) {
		edit_session.remove({
			start: position,
			end: live_document.indexToPosition(transform.position + transform.num_delete, 0)
		});
	}
	if ( typeof(transform.insert) === "string" && transform.insert.length > 0 ) {
		edit_session.insert(position, transform.insert);
	}

	this._blind_eye_turned = false;

	this._content = this._leap_client.apply(transform, this._content);

	setTimeout((function() {
		if ( this._content !== this._ace.getValue() ) {
			console.error("internal content and editor content do not match");
		}
	}).bind(this), 0);
};

/* convert_to_transform, takes an ace editor event, converts it into a transform and sends it.
 */
leap_bind_ace_editor.prototype._convert_to_transform = function(e) {
	"use strict";

	if ( this._blind_eye_turned ) {
		return;
	}

	var tform = {};

	var live_document = this._ace.getSession().getDocument();

	switch (e.data.action) {
	case "insertText":
		tform.position = live_document.positionToIndex(e.data.range.start, 0);
		tform.insert = e.data.text;
		break;
	case "removeText":
		tform.position = live_document.positionToIndex(e.data.range.start, 0);
		tform.num_delete = e.data.text.length;
		break;
	case "removeLines":
		tform.position = live_document.positionToIndex(e.data.range.start, 0);
		tform.num_delete = e.data.lines.join(e.data.nl).length + e.data.nl.length;
		break;
	}

	if ( tform.insert === undefined && tform.num_delete === undefined ) {
		console.error("change resulted in invalid transform: " + JSON.stringify(e.data));
	}

	this._content = this._leap_client.apply(tform, this._content);
	var err = this._leap_client.send_transform(tform);
	if ( err !== undefined ) {
		console.error(err);
	}

	setTimeout((function() {
		if ( this._content !== this._ace.getValue() ) {
			console.error("internal content and editor content do not match");
		}
	}).bind(this), 0);
};

/*--------------------------------------------------------------------------------------------------
 */

try {
	if ( leap_client !== undefined && typeof(leap_client) === "function" ) {
		leap_client.prototype.bind_ace_editor = function(ace_editor) {
			this._ace_editor = new leap_bind_ace_editor(this, ace_editor);
		};
	}
} catch (e) {
}

/*--------------------------------------------------------------------------------------------------
 */
