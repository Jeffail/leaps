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

/*
_create_leaps_ace_marker - creates a marker for displaying the cursor positions of other users in an
ace editor.
*/
var _create_leaps_ace_marker = function(session) {
	var marker = {};

	marker.draw_handler = null;
	marker.clear_handler = null;
	marker.cursors = [];

	marker.update = function(html, markerLayer, session, config) {
		if ( typeof marker.clear_handler === 'function' ) {
			marker.clear_handler();
		}
		var cursors = marker.cursors;
		for (var i = 0; i < cursors.length; i++) {
			var pos = cursors[i].position;
			var screenPos = session.documentToScreenPosition(pos);

			var height = config.lineHeight;
			var width = config.characterWidth;
			var top = markerLayer.$getTop(screenPos.row, config);
			var left = markerLayer.$padding + screenPos.column * width;

			var stretch = 4;

			if ( typeof marker.draw_handler === 'function' ) {
				var content = (marker.draw_handler(
					cursors[i].user_id, cursors[i].session_id, height, top, left, screenPos.row, screenPos.column
				) || '') + '';
				html.push(content);
			} else {
				html.push(
					"<div class='LeapsAceCursor' style='",
					"height:", (height + stretch), "px;",
					"top:", (top - (stretch/2)), "px;",
					"left:", left, "px; width:", width, "px'></div>");
			}
		}
	};

	marker.redraw = function() {
		marker.session._signal("changeFrontMarker");
	};

	marker.updateCursor = function(update) {
		var cursors = marker.cursors, current, i, l;
		for ( i = 0, l = cursors.length; i < l; i++ ) {
			if ( cursors[i].session_id === update.client.session_id ) {
				current = cursors[i];
				current.position = marker.session.getDocument().indexToPosition(update.message.position, 0);
				current.updated = new Date().getTime();
				break;
			}
		}
		if ( undefined === current ) {
			if ( update.message.active ) {
				current = {
					user_id: update.client.user_id,
					session_id: update.client.session_id,
					position: marker.session.getDocument().indexToPosition(update.client.position, 0),
					updated: new Date().getTime()
				};
				cursors.push(current);
			}
		} else if ( !update.message.active ) {
			cursors.splice(i, 1);
		}

		marker.redraw();
	};

	marker.session = session;
	marker.session.addDynamicMarker(marker, true);

	return marker;
};

/* leap_bind_ace_editor takes an existing leap_client and uses it to convert an Ace web editor
 * (http://ace.c9.io) into a live leaps shared editor.
 */
var leap_bind_ace_editor = function(leap_client, session) {
	if ( null === document.getElementById("leaps-ace-style") ) {
		var node = document.createElement('style');
		node.id = "leaps-ace-style";
		node.innerHTML =
		".LeapsAceCursor {" +
			"position: absolute;" +
			"border-left: 3px solid #D11956;" +
		"}";
		document.body.appendChild(node);
	}

	this._leap_client = leap_client;
	this._session = session;

	this._content = "";
	this._ready = false;
	this._blind_eye_turned = false;

	//this._ace.setReadOnly(true);

	this._marker = _create_leaps_ace_marker(session);

	var binder = this;

	session.on('change', function(e) {
		binder._convert_to_transform.apply(binder, [ e ]);
	});

	this._leap_client.subscribe_event("document", function(doc) {
		console.log(doc)
		binder._content = doc.content;

		binder._blind_eye_turned = true;
		session.setValue(doc.content);
		//binder._ace.setReadOnly(false);
		session.selection.clearSelection();

		var old_undo = session.getUndoManager();
		old_undo.reset();
		session.setUndoManager(old_undo);

		binder._ready = true;
		binder._blind_eye_turned = false;

		binder._pos_interval = setInterval(function() {
			var doc = session.getDocument();
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
		//binder._ace.setReadOnly(true);

		if ( undefined !== binder._pos_interval ) {
			clearTimeout(binder._pos_interval);
		}
	});

	this._leap_client.subscribe_event("user", function(update) {
		binder._marker.updateCursor.apply(binder._marker, [ update ]);
	});

	this._leap_client.ACE_set_cursor_handler = function(handler, clear_handler) {
		binder.set_cursor_handler(handler, clear_handler);
	};
};

/* set_cursor_handler, sets the method call that returns a cursor marker. Also adds an optional
 * clear_handler which is called before each individual cursor is drawn (use it to clear all outside
 * markers before redrawing).
 */
leap_bind_ace_editor.prototype.set_cursor_handler = function(handler, clear_handler) {
	if ( 'function' === typeof handler ) {
		this._marker.draw_handler = handler;
	}
	if ( 'function' === typeof clear_handler ) {
		this._marker.clear_handler = clear_handler;
	}
};

/* apply_transform, applies a single transform to the ace document.
 */
leap_bind_ace_editor.prototype._apply_transform = function(transform) {
	this._blind_eye_turned = true;

	var edit_session = this._session;
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
		if ( this._content !== this._session.getValue() ) {
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
	if ( this._blind_eye_turned ) {
		return;
	}

	var tform = {};

	var live_document = this._session.getDocument();
	var nl = live_document.getNewLineCharacter();

	switch (e.action) {
	case "insert":
		tform.position = live_document.positionToIndex(e.start, 0);
		tform.insert = e.lines.join(nl);
		break;
	case "remove":
		tform.position = live_document.positionToIndex(e.start, 0);
		tform.num_delete = e.lines.join(nl).length;
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
		if ( this._content !== this._session.getValue() ) {
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
		window.leap_client.prototype.bind_ace_editor = function(ace_editor) {
			this._ace_editor = new leap_bind_ace_editor(this, ace_editor);
		};
	}
} catch (e) {
	console.error(e);
}

/*--------------------------------------------------------------------------------------------------
 */

})();
