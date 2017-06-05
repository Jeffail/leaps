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

	this._cursors = {};

	var binder = this;

	this._codemirror.on('beforeChange', function(instance, e) {
		binder._convert_to_transform.apply(binder, [ e ]);
	});

	this._leap_client.subscribe_event("user", function(update) {
		binder.update_user_info(update);
	});

	this._leap_client.subscribe_event("document", function(doc) {
		binder._content = doc.content;

		binder._blind_eye_turned = true;
		binder._codemirror.getDoc().setValue(doc.content);
		binder._codemirror.getDoc().clearHistory();

		binder._ready = true;
		binder._blind_eye_turned = false;

		binder._pos_interval = setInterval(function() {
			var live_document = binder._codemirror.getDoc();
			var position = u_index_from_pos(live_document, live_document.getCursor());
			binder._leap_client.update_cursor.apply(binder._leap_client, [ position ]);
		}, leap_client._POSITION_POLL_PERIOD);

		binder._codemirror.on("cursorActivity", function() {
			var live_document = binder._codemirror.getDoc();
			var position = u_index_from_pos(live_document, live_document.getCursor());
			binder._leap_client.update_cursor.apply(binder._leap_client, [ position ]);
		});
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
	var history = live_document.getHistory();
	history.done = history.done.slice(0, -2);
	live_document.setHistory(history);

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

function HSVtoRGB(h, s, v) {
	var r, g, b, i, f, p, q, t;
	if (h && s === undefined && v === undefined) {
		s = h.s, v = h.v, h = h.h;
	}
	i = Math.floor(h * 6);
	f = h * 6 - i;
	p = v * (1 - s);
	q = v * (1 - f * s);
	t = v * (1 - (1 - f) * s);
	switch (i % 6) {
		case 0: r = v, g = t, b = p; break;
		case 1: r = q, g = v, b = p; break;
		case 2: r = p, g = v, b = t; break;
		case 3: r = p, g = q, b = v; break;
		case 4: r = t, g = p, b = v; break;
		case 5: r = v, g = p, b = q; break;
	}
	return {
		r: Math.floor(r * 255),
		g: Math.floor(g * 255),
		b: Math.floor(b * 255)
	};
}

function hash(str) {
	var hash = 0, i, chr, len;
	if ('string' !== typeof str || str.length === 0) {
		return hash;
	}
	for (i = 0, len = str.length; i < len; i++) {
		chr   = str.charCodeAt(i);
		hash  = ((hash << 5) - hash) + chr;
		hash |= 0; // Convert to 32bit integer
	}
	return hash;
}

function id_to_color(id) {
	var id_hash = hash(id);
	if ( id_hash < 0 ) {
		id_hash = id_hash * -1;
	}

	var hue = ( id_hash % 10000 ) / 10000;
	var rgb = HSVtoRGB(hue, 1, 0.8);

	return "rgba(" + rgb.r + ", " + rgb.g + ", " + rgb.b + ", 1)";
}

function dom_from_update(cm, update) {
	var line_height = cm.defaultTextHeight();
	var stretch = 4;
	var height = line_height + stretch;

	var label_height = 30;

	var thickness = 2;

	// TODO: If line is above, below or too far right to view
	var root = document.createElement('div');
	root.style.position = 'absolute';
	root.style.zIndex = 200;

	var bar = document.createElement('div');
	bar.style.position = 'relative';
	bar.style.top = '-' + height + 'px';
	bar.style.height = height + 'px';
	bar.style.width =  thickness + 'px';
	bar.style.backgroundColor = id_to_color(update.client.session_id);

	var label = document.createElement('div');
	label.style.position = 'relative';
	label.style.top = '-' + (height + label_height) + 'px';
	label.style.padding = thickness + 'px';
	label.style.backgroundColor = id_to_color(update.client.session_id);
	label.style.color = "#fcfcfc";
	label.appendChild(document.createTextNode(update.client.user_id));

	root.appendChild(bar);
	root.appendChild(label);

	return root;
}

// update_user_info updates any visual state of other users within the
// CodeMirror screen.
leap_bind_codemirror.prototype.update_user_info = function(update) {
	if ( this._cursors.hasOwnProperty(update.client.session_id) ) {
		if ( update.message.active ) {
			if ( this._cursors[update.client.session_id].position !== update.message.position ) {
				this._cursors[update.client.session_id].position = update.message.position;
				this._codemirror.addWidget(
					pos_from_u_index(this._codemirror.getDoc(), update.message.position),
					this._cursors[update.client.session_id].dom, false
				);
			}
		} else {
			let dom = this._cursors[update.client.session_id].dom;
			dom.parentNode.removeChild(dom);
			delete this._cursors[update.client.session_id];
		}
	} else if ( update.message.active ) {
		let dom = dom_from_update(this._codemirror, update);
		this._cursors[update.client.session_id] = {
			position: update.message.position,
			dom: dom
		};
		this._codemirror.addWidget(
			pos_from_u_index(this._codemirror.getDoc(), update.message.position),
			dom, false
		);
	}
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
