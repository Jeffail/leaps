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

//------------------------------------------------------------------------------

// Gets a position in unicode codepoints.
function pos_from_u_index(doc, index) {
	let ch = 0, lineNo = doc.first, sepStr = doc.lineSeparator();
	let sepSize = sepStr.length;

	doc.iter(line => {
		let uline = new leap_str(line.text + sepStr);
		let ulength = uline.u_str().length;
		index -= ulength;
		if (index < 0) {
			if ( ulength === (line.text.length + sepSize) ) {
				ch = index + ulength;
			} else {
				ch = uline.u_str().slice(0, index).join('').length;
			}
			return true;
		}
		++lineNo;
	});

	return CodeMirror.Pos(lineNo, ch);
}

// Gets an index from a position in unicode codepoints.
function u_index_from_pos(doc, coords) {
	let index = 0;
	if (coords.line < doc.first || coords.ch < 0) {
		return 0;
	}
	let sepStr = doc.lineSeparator();
	let sepSize = sepStr.length;
	let lineNo = doc.first;
	doc.iter(doc.first, coords.line+1, line => {
		if ( lineNo === coords.line ) {
			index += (new leap_str((line.text+sepStr).slice(0, coords.ch))).u_str().length;
		} else {
			index += (new leap_str(line.text)).u_str().length + sepSize;
		}
		++lineNo;
	});
	return index;
}

//------------------------------------------------------------------------------

// leap_bind_codemirror takes an existing leap_client and uses it to convert a
// codemirror web editor (http://codemirror.net/) into a live shared editor.
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
			var position = u_index_from_pos(live_document, live_document.getCursor());
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

// apply_transform, applies a single transform to the codemirror document.
leap_bind_codemirror.prototype._apply_transform = function(transform) {
	this._blind_eye_turned = true;

	console.log("Received: " + JSON.stringify(transform));

	var live_document = this._codemirror.getDoc();
	var start_position = pos_from_u_index(live_document, transform.position), end_position = start_position;

	if ( transform.num_delete > 0 ) {
		end_position = pos_from_u_index(live_document, transform.position + transform.num_delete);
	}

	var insert = "";
	if ( (transform.insert instanceof leap_str) && transform.insert.str().length > 0 ) {
		insert = transform.insert.str();
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

// convert_to_transform, takes a codemirror edit event, converts it into a
// transform and sends it.
leap_bind_codemirror.prototype._convert_to_transform = function(e) {
	if ( this._blind_eye_turned ) {
		return;
	}

	var tform = {};

	var live_document = this._codemirror.getDoc();
	var start_index = u_index_from_pos(live_document, e.from), end_index = u_index_from_pos(live_document, e.to);

	tform.position = start_index;
	tform.insert = e.text.join('\n') || "";

	tform.num_delete = end_index - start_index;

	if ( tform.insert.length <= 0 && tform.num_delete <= 0 ) {
		return;
		/*
		this._leap_client._dispatch_event.apply(this._leap_client,
			[ this._leap_client.EVENT_TYPE.ERROR, [
				"Change resulted in invalid transform"
			] ]);
		*/
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

//------------------------------------------------------------------------------

try {
	if ( window.leap_client !== undefined && typeof(window.leap_client) === "function" ) {
		window.leap_client.prototype.bind_codemirror = function(codemirror_object) {
			this._codemirror = new leap_bind_codemirror(this, codemirror_object);
		};
	}
} catch (e) {
}

//------------------------------------------------------------------------------

})();
