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

/*
_create_leaps_ace_marker - creates a marker for displaying the cursor positions of other users in an
ace editor.
*/
var _create_leaps_ace_marker = function(ace_editor) {
	"use strict";

	var marker = {};

	marker.cursors = [];

	marker.update = function(html, markerLayer, session, config) {
		var cursors = marker.cursors;
		for (var i = 0; i < cursors.length; i++) {
			var pos = cursors[i].position;
			var screenPos = session.documentToScreenPosition(pos);

            var height = config.lineHeight;
            var width = config.characterWidth;
            var top = markerLayer.$getTop(screenPos.row, config);
            var left = markerLayer.$padding + screenPos.column * width;

            html.push(
                "<div class='LeapsAceCursor' style='",
                "height:", height, "px;",
                "top:", top, "px;",
                "left:", left, "px; width:", width, "px'></div>"
            );
		}
	};

	marker.redraw = function() {
	   marker.session._signal("changeFrontMarker");
	};

	marker.updateCursor = function(user) {
		var cursors = marker.cursors, current, i, l;
		for ( i = 0, l = cursors.length; i < l; i++ ) {
			if ( cursors[i].user_id === user.user_id ) {
				current = cursors[i];
				current.position = marker.session.getDocument().indexToPosition(user.position, 0);
				current.updated = new Date().getTime();
				break;
			}
		}
		if ( undefined === current ) {
			if ( user.active ) {
				current = {
					user_id: user.user_id,
					position: marker.session.getDocument().indexToPosition(user.position, 0),
					updated: new Date().getTime()
				};
				cursors.push(current);
			}
		} else if ( !user.active ) {
			cursors.splice(i, 1);
		}

		marker.redraw();
	};

	marker.session = ace_editor.getSession();
	marker.session.addDynamicMarker(marker, true);

	return marker;
};

/* leap_bind_ace_editor takes an existing leap_client and uses it to convert an Ace web editor
 * (http://ace.c9.io) into a live leaps shared editor.
 */
var leap_bind_ace_editor = function(leap_client, ace_editor) {
	"use strict";

	if ( null === document.getElementById("leaps-ace-style") ) {
		var node = document.createElement('style');
		node.id = "leaps-ace-style";
		node.innerHTML =
		".LeapsAceCursor {" +
			"position: absolute;" +
			"border-left: 2px solid gold;" +
			"background-color: rgba(50, 50, 50, 0.2);" +
		"}";
		document.body.appendChild(node);
	}

	this._ace = ace_editor;
	this._leap_client = leap_client;

	this._content = "";
	this._ready = false;
	this._blind_eye_turned = false;

	this._ace.setReadOnly(true);

	this._marker = _create_leaps_ace_marker(this._ace);

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

		var old_undo = binder._ace.getSession().getUndoManager();
		old_undo.reset();
		binder._ace.getSession().setUndoManager(old_undo);

		binder._ready = true;
		binder._blind_eye_turned = false;

		binder._pos_interval = setInterval(function() {
			var session = binder._ace.getSession(), doc = session.getDocument();
			var position = session.getSelection().getCursor();
			var index = doc.positionToIndex(position, 0);

			binder._leap_client.update_cursor.apply(binder._leap_client, [ index ]);
		}, leap_client._POSITION_POLL_PERIOD);
	});

	this._leap_client.subscribe_event("transforms", function(transforms) {
		for ( var i = 0, l = transforms.length; i < l; i++ ) {
			binder._apply_transform.apply(binder, [ transforms[i] ]);
		}
	});

	this._leap_client.subscribe_event("disconnect", function() {
		binder._ace.setReadOnly(true);

		if ( undefined !== binder._pos_interval ) {
			clearTimeout(binder._pos_interval);
		}
	});

	this._leap_client.subscribe_event("user", function(user) {
		binder._marker.updateCursor.apply(binder._marker, [ user ]);
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
			this._leap_client._dispatch_event.apply(this._leap_client,
				[ this._leap_client.EVENT_TYPE.ERROR, [
					"Local editor has lost synchronization with server"
				] ]);
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
	var nl = live_document.getNewLineCharacter();

	switch (e.data.action) {
	case "insertText":
		tform.position = live_document.positionToIndex(e.data.range.start, 0);
		tform.insert = e.data.text;
		break;
	case "insertLines":
		tform.position = live_document.positionToIndex(e.data.range.start, 0);
		tform.insert = e.data.lines.join(nl) + nl;
		break;
	case "removeText":
		tform.position = live_document.positionToIndex(e.data.range.start, 0);
		tform.num_delete = e.data.text.length;
		break;
	case "removeLines":
		tform.position = live_document.positionToIndex(e.data.range.start, 0);
		tform.num_delete = e.data.lines.join(nl).length + nl.length;
		break;
	}

	if ( tform.insert === undefined && tform.num_delete === undefined ) {
		this._leap_client._dispatch_event.apply(this._leap_client,
			[ this._leap_client.EVENT_TYPE.ERROR, [
				"Local change resulted in invalid transform"
			] ]);
	}

	this._content = this._leap_client.apply(tform, this._content);
	var err = this._leap_client.send_transform(tform);
	if ( err !== undefined ) {
		this._leap_client._dispatch_event.apply(this._leap_client,
			[ this._leap_client.EVENT_TYPE.ERROR, [
				"Local change resulted in invalid transform: " + err
			] ]);
	}

	setTimeout((function() {
		if ( this._content !== this._ace.getValue() ) {
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
	if ( leap_client !== undefined && typeof(leap_client) === "function" ) {
		leap_client.prototype.bind_ace_editor = function(ace_editor) {
			this._ace_editor = new leap_bind_ace_editor(this, ace_editor);
		};
	}
} catch (e) {
}

/*--------------------------------------------------------------------------------------------------
 */
