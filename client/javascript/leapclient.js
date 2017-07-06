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

/*jshint newcap: false, esversion: 6*/

var leap_client = {};
var leap_str = {};

(function() {
"use strict";

/*--------------------------------------------------------------------------------------------------
 */

/* leap_str is a wrapper around strings that stores lazy evaluated codepoint arrays using the ES6
 * triple dot operator.
 *
 * @param any_str either a standard string or an array of unicode codepoints.
 */
leap_str = function(any_str) {
	if ( any_str instanceof leap_str ) {
		this._str = any_str._str;
		this._u_str = any_str._u_str;
	} else if ( any_str instanceof String ) {
		this._str = any_str;
	} else if ( any_str instanceof Array ) {
		this._u_str = any_str;
	} else if ( typeof any_str === "string" ) {
		this._str = any_str;
	} else {
		throw TypeError("attempted to construct leap_str with non-string/array type");
	}
};

// Returns the standard underlying string.
leap_str.prototype.str = function() {
	// Lazy evaluated.
	if ( undefined === this._str ) {
		this._str = this._u_str.join('');
	}
	return this._str;
};

// Returns the underlying unicode codepoint array.
leap_str.prototype.u_str = function() {
	if ( undefined === this._u_str ) {
		this._u_str = [...this._str];
	}
	return this._u_str;
};

/*--------------------------------------------------------------------------------------------------
 */

/* leap_model is an object designed to keep track of the inbound and outgoing transforms
 * for a local document, and updates the caller with the appropriate actions at each stage.
 *
 * leap_model has three states:
 * 1. READY     - No pending sends, transforms received can be applied instantly to local document.
 * 2. SENDING   - Transforms are being sent and we're awaiting the corrected version of those
 *                transforms.
 * 3. BUFFERING - A corrected version has been received for our latest send but we're still waiting
 *                for the transforms that came before that send to be received before moving on.
 */
var leap_model = function(id, base_version) {
	this.id = id;

	this.READY = 1;
	this.SENDING = 2;
	this.BUFFERING = 3;

	this._leap_state = this.READY;

	this._corrected_version = 0;
	this._version = base_version;

	this._unapplied = [];
	this._unsent = [];
	this._sending = null;
};

/* _validate_transforms iterates an array of transform objects and validates that each transform
 * contains the correct fields. Returns an error message as a string if there was a problem.
 */
leap_model.prototype._validate_transforms = function(transforms) {
	for ( var i = 0, l = transforms.length; i < l; i++ ) {
		var tform = transforms[i];

		if ( typeof(tform.position) !== "number" ) {
			tform.position = parseInt(tform.position);
			if ( isNaN(tform.position) ) {
				return "transform contained NaN value for position: " + JSON.stringify(tform);
			}
		}
		if ( tform.num_delete !== undefined ) {
			if ( typeof(tform.num_delete) !== "number" ) {
				tform.num_delete = parseInt(tform.num_delete);
				if ( isNaN(tform.num_delete) ) {
					return "transform contained NaN value for num_delete: " + JSON.stringify(tform);
				}
			}
		} else {
			tform.num_delete = 0;
		}
		if ( tform.version !== undefined && typeof(tform.version) !== "number" ) {
			tform.version = parseInt(tform.version);
			if ( isNaN(tform.version) ) {
				return "transform contained NaN value for version: " + JSON.stringify(tform);
			}
		}
		if ( tform.insert !== undefined ) {
			try {
				tform.insert = new leap_str(tform.insert);
			} catch(e) {
				return "transform contained non-string value for insert: " + JSON.stringify(tform);
			}
		} else {
			tform.insert = new leap_str("");
		}
	}
};

/* merge_transforms takes two transforms (the next to be sent, and the one that follows) and
 * attempts to merge them into one transform. This will not be possible with some combinations, and
 * the function returns a boolean to indicate whether the merge was successful.
 */
leap_model.prototype._merge_transforms = function(first, second) {
	var overlap, remainder;

	var first_len = first.insert.u_str().length;

	if ( ( first.position + first_len) === second.position ) {
		first.insert = new leap_str(first.insert.str() + second.insert.str());
		first.num_delete += second.num_delete;
		return true;
	}
	if ( second.position === first.position ) {
		remainder = Math.max(0, second.num_delete - first_len);
		first.num_delete += remainder;
		first.insert = new leap_str(second.insert.str() + first.insert.u_str().slice(second.num_delete).join(''));
		return true;
	}
	if ( second.position > first.position && second.position < ( first.position + first_len ) ) {
		overlap = second.position - first.position;
		remainder = Math.max(0, second.num_delete - (first_len - overlap));
		first.num_delete += remainder;
		first.insert = new leap_str(first.insert.u_str().slice(0, overlap).join('') +
			second.insert.str() + first.insert.u_str().slice(overlap + second.num_delete).join(''));
		return true;
	}
	return false;
};

/* collide_transforms takes an unapplied transform from the server, and an unsent transform from the
 * client and modifies both transforms.
 *
 * The unapplied transform is fixed so that when applied to the local document is unaffected by the
 * unsent transform that has already been applied. The unsent transform is fixed so that it is
 * unaffected by the unapplied transform when submitted to the server.
 */
leap_model.prototype._collide_transforms = function(unapplied, unsent) {
	var earlier, later;

	if ( unapplied.position <= unsent.position ) {
		earlier = unapplied;
		later = unsent;
	} else {
		earlier = unsent;
		later = unapplied;
	}

	var earlier_len = earlier.insert.u_str().length;
	var later_len = later.insert.u_str().length;

	if ( earlier.num_delete === 0 ) {
		later.position += earlier_len;
	} else if ( ( earlier.num_delete + earlier.position ) <= later.position ) {
		later.position += ( earlier_len - earlier.num_delete );
	} else {
		var pos_gap = later.position - earlier.position;
		var excess = Math.max(0, (earlier.num_delete - pos_gap));

		// earlier changes
		if ( excess > later.num_delete ) {
			earlier.num_delete += later_len - later.num_delete;
			earlier.insert = new leap_str(earlier.insert.str() + later.insert.str());
		} else {
			earlier.num_delete = pos_gap;
		}
		// later changes
		later.num_delete = Math.max(0, later.num_delete - excess);
		later.position = earlier.position + earlier_len;
	}
};

/*--------------------------------------------------------------------------------------------------
 */

/* _resolve_state will prompt the leap_model to re-evalutate its current state for validity. If this
 * state is determined to no longer be appropriate then it will return an object containing the
 * following actions to be performed.
 */
leap_model.prototype._resolve_state = function() {
	switch (this._leap_state) {
	case this.READY:
	case this.SENDING:
		return {};
	case this.BUFFERING:
		if ( ( this._version + this._unapplied.length ) >= (this._corrected_version - 1) ) {

			this._version += this._unapplied.length + 1;
			var to_collide = [ this._sending ].concat(this._unsent);
			var unapplied = this._unapplied;

			this._unapplied = [];

			for ( var i = 0, li = unapplied.length; i < li; i++ ) {
				for ( var j = 0, lj = to_collide.length; j < lj; j++ ) {
					this._collide_transforms(unapplied[i], to_collide[j]);
				}
			}

			this._sending = null;

			if ( this._unsent.length > 0 ) {
				this._sending = this._unsent.shift();
				while ( this._unsent.length > 0 && this._merge_transforms(this._sending, this._unsent[0]) ) {
					this._unsent.shift();
				}
				this._sending.version = this._version + 1;

				this._leap_state = this.SENDING;
				return { send : {
					version: this._sending.version,
					num_delete: this._sending.num_delete,
					insert: this._sending.insert.str(),
					position: this._sending.position
				}, apply : unapplied };
			} else {
				this._leap_state = this.READY;
				return { apply : unapplied };
			}
		}
	}
	return {};
};

/* correct is the function to call following a "correction" from the server, this correction value
 * gives the model the information it needs to determine which changes are missing from our model
 * from before our submission was accepted.
 */
leap_model.prototype.correct = function(version) {
	switch (this._leap_state) {
	case this.READY:
	case this.BUFFERING:
		return { error : "received unexpected correct action" };
	case this.SENDING:
		this._leap_state = this.BUFFERING;
		this._corrected_version = version;
		return this._resolve_state();
	}
	return {};
};

/* submit is the function to call when we wish to submit more local changes to the server. The model
 * will determine whether it is currently safe to dispatch those changes to the server, and will
 * also provide each change with the correct version number.
 */
leap_model.prototype.submit = function(transform) {
	switch (this._leap_state) {
	case this.READY:
		this._leap_state = this.SENDING;
		transform.version = this._version + 1;
		this._sending = transform;
		return { send : {
			version: this._sending.version,
			num_delete: this._sending.num_delete,
			insert: this._sending.insert.str(),
			position: this._sending.position
		} };
	case this.BUFFERING:
	case this.SENDING:
		this._unsent = this._unsent.concat(transform);
	}
	return {};
};

/* receive is the function to call when we have received transforms from our server. If we have
 * recently dispatched transforms and have yet to receive our correction then it is unsafe to apply
 * these changes to our local document, so the model will keep return these transforms to us when it
 * is known to be safe.
 */
leap_model.prototype.receive = function(transforms) {
	var expected_version = this._version + this._unapplied.length + 1;
	if ( (transforms.length > 0) && (transforms[0].version !== expected_version) ) {
		return { error :
			("Received unexpected transform version: " + transforms[0].version +
				", expected: " + expected_version) };
	}

	switch (this._leap_state) {
	case this.READY:
		this._version += transforms.length;
		return { apply : transforms };
	case this.BUFFERING:
		this._unapplied = this._unapplied.concat(transforms);
		return this._resolve_state();
	case this.SENDING:
		this._unapplied = this._unapplied.concat(transforms);
	}
	return {};
};

/*--------------------------------------------------------------------------------------------------
 */

/* leap_client is the main tool provided to allow an easy and stable interface for connecting to a
 * leaps server.
 */
leap_client = function() {
	this._socket = null;

	this._models = [];

	this.EVENT_TYPE = {
		CONNECT: "connect",
		DISCONNECT: "disconnect",
		SUBSCRIBE: "subscribe",
		UNSUBSCRIBE: "unsubscribe",
		TRANSFORMS: "transforms",
		METADATA: "metadata",
		GLOBAL_METADATA: "global_metadata",
		ERROR: "error"
	};

	this._events = {};
	this._single_events = {};
};

/* on, attach a function to an event of the leap_client. Use this to subscribe to
 * transforms, document responses and errors etc.
 */
leap_client.prototype.on = function(name, subscriber) {
	if ( typeof(subscriber) !== "function" ) {
		return "subscriber was not a function";
	}
	var targets = this._events[name];
	if ( targets !== undefined && targets instanceof Array ) {
		targets.push(subscriber);
	} else {
		this._events[name] = [ subscriber ];
	}
};

/* on_next, attach a function to the next trigger only of an event of the
 * leap_client.
 */
leap_client.prototype.on_next = function(name, subscriber) {
	if ( typeof(subscriber) !== "function" ) {
		return "subscriber was not a function";
	}
	var targets = this._single_events[name];
	if ( targets !== undefined && targets instanceof Array ) {
		targets.push(subscriber);
	} else {
		this._single_events[name] = [ subscriber ];
	}
};

/* clear_handlers, removes all functions subscribed to an event.
 */
leap_client.prototype.clear_handlers = function(name) {
	this._events[name] = [];
	this._single_events[name] = [];
};

/* dispatch_event, sends args to all subscribers of an event.
 */
leap_client.prototype._dispatch_event = function(name, args) {
	var targets = this._events[name];
	if ( targets !== undefined && targets instanceof Array ) {
		for ( var i = 0, l = targets.length; i < l; i++ ) {
			if (typeof(targets[i]) === "function") {
				targets[i].apply(this, args);
			}
		}
	}
	var single_events = this._single_events[name];
	if ( single_events !== undefined && single_events instanceof Array ) {
		while (single_events.length > 0) {
			var next = single_events.pop();
			if ( typeof(next) === "function" ) {
				next.apply(this, args);
			}
		}
	}
};

/* _do_action is a call that acts accordingly provided an action_obj from our leap_model.
 */
leap_client.prototype._do_action = function(model_id, action_obj) {
	if ( action_obj.error !== undefined ) {
		return action_obj.error;
	}
	if ( action_obj.apply !== undefined && action_obj.apply instanceof Array ) {
		this._dispatch_event(this.EVENT_TYPE.TRANSFORMS, [ {
			document: {
				id: model_id
			},
			transforms: action_obj.apply
		} ]);
	}
	if ( action_obj.send !== undefined && action_obj.send instanceof Object ) {
		this._socket.send(JSON.stringify({
			type: "transform",
			body: {
				document: {
					id: model_id
				},
				transform: action_obj.send
			}
		}));
	}
};

/* _process_message is a call that takes a server provided message object and decides the
 * appropriate action to take. If an error occurs during this process then an error message is
 * returned.
 */
leap_client.prototype._process_message = function(message) {
	var validate_error, action_obj, action_err;

	if ( message.type === undefined || typeof(message.type) !== "string" ) {
		console.log(JSON.stringify(message));
		return "message received did not contain a valid type";
	}
	if ( message.body === undefined || typeof(message.body) !== "object" ) {
		return "message received did not contain a valid body";
	}

	var msg_body = message.body;
	var document_id = "";

	switch (message.type) {
	case "subscribe":
		if ( "object" !== typeof(msg_body.document) ||
		     "string" !== typeof(msg_body.document.id) ||
		     "string" !== typeof(msg_body.document.content) ||
		     msg_body.document.version <= 0 ) {
			return "message document type contained invalid document object";
		}
		this._models[msg_body.document.id] = new leap_model(
				msg_body.document.id, msg_body.document.version
			);
		this._dispatch_event(this.EVENT_TYPE.SUBSCRIBE, [ msg_body ]);
		break;
	case "unsubscribe":
		if ( "object" !== typeof(msg_body.document) ||
		     "string" !== typeof(msg_body.document.id) ) {
			return "message document type contained invalid document object";
		}
		delete this._models[msg_body.document.id];
		this._dispatch_event(this.EVENT_TYPE.UNSUBSCRIBE, [ msg_body ]);
		break;
	case "transforms":
		document_id = msg_body.document.id;
		var transforms = msg_body.transforms;
		if ( !this._models.hasOwnProperty(document_id) ) {
			return "transforms were received for unsubscribed document";
		}
		var model = this._models[document_id];

		validate_error = model._validate_transforms(transforms);
		if ( validate_error !== undefined ) {
			return "received transforms with error: " + validate_error;
		}
		action_obj = model.receive(transforms);
		action_err = this._do_action(document_id, action_obj);
		if ( action_err !== undefined ) {
			return "failed to receive transforms: " + action_err;
		}
		break;
	case "metadata":
		this._dispatch_event(this.EVENT_TYPE.METADATA, [ msg_body ]);
		break;
	case "global_metadata":
		this._dispatch_event(this.EVENT_TYPE.GLOBAL_METADATA, [ msg_body ]);
		break;
	case "correction":
		document_id = msg_body.document.id;
		if ( !this._models.hasOwnProperty(document_id) ) {
			return "correction was received for unsubscribed document";
		}
		if ( typeof(msg_body.correction) !== "object" ) {
			return "correction received without body";
		}
		if ( typeof(msg_body.correction.version) !== "number" ) {
			msg_body.correction.version = parseInt(msg_body.correction.version);
			if ( isNaN(msg_body.correction.version) ) {
				return "correction received was NaN";
			}
		}
		var model = this._models[document_id];

		action_obj = model.correct(msg_body.correction.version);
		action_err = this._do_action(document_id, action_obj);
		if ( action_err !== undefined ) {
			return "model failed to correct: " + action_err;
		}
		break;
	case "error":
		if ( this._socket !== null ) {
			this._socket.close();
		}
		if ( typeof(msg_body.error.message) === "string" ) {
			return msg_body.error.message;
		}
		return "server sent undeterminable error";
	default:
		return "message received was not a recognised type";
	}
};

/* send_transform is the function to call to send a transform off to the server. To keep the local
 * document responsive this transform should be applied to the document straight away. The
 * leap_client will decide when it is appropriate to dispatch the transform, and will manage
 * internally how incoming messages should be altered to account for the fact that the local
 * change was made out of order.
 */
leap_client.prototype.send_transform = function(document_id, transform) {
	if ( !this._models.hasOwnProperty(document_id) ) {
		return "leap_client must be subscribed to document before submitting transforms";
	}

	var model = this._models[document_id];

	var validate_error = model._validate_transforms([ transform ]);
	if ( validate_error !== undefined ) {
		return validate_error;
	}

	var action_obj = model.submit(transform);
	var action_err = this._do_action(document_id, action_obj);
	if ( action_err !== undefined ) {
		return "model failed to submit: " + action_err;
	}
};

/* send_metadata - send metadata out to all other users connected to your shared document.
 */
leap_client.prototype.send_metadata = function(document_id, metadata) {
	if ( !this._models.hasOwnProperty(document_id) ) {
		return "leap_client must be subscribed to document before submitting metadata";
	}

	this._socket.send(JSON.stringify({
		type: "metadata",
		body: {
			document: {
				id: document_id
			},
			metadata: metadata,
		}
	}));
};

/* send_global_metadata - send global metadata out to all other users connected
 * to the leaps service.
 */
leap_client.prototype.send_global_metadata = function(metadata) {
	this._socket.send(JSON.stringify({
		type: "global_metadata",
		body: {
			metadata: metadata,
		}
	}));
};

/* subscribe to a document session, providing the initial content as well as
 * subsequent changes to the document.
 */
leap_client.prototype.subscribe = function(document_id) {
	if ( this._socket === null || this._socket.readyState !== 1 ) {
		return "leap_client is not currently connected";
	}

	if ( typeof(document_id) !== "string" ) {
		return "document id was not a string type";
	}

	this._socket.send(JSON.stringify({
		type: "subscribe",
		body: {
			document: {
				id: document_id
			}
		}
	}));
};

/* unsubscribe from a document session.
 */
leap_client.prototype.unsubscribe = function(document_id) {
	if ( this._socket === null || this._socket.readyState !== 1 ) {
		return "leap_client is not currently connected";
	}

	if ( typeof(document_id) !== "string" ) {
		return "document id was not a string type";
	}

	this._socket.send(JSON.stringify({
		type: "unsubscribe",
		body: {
			document: {
				id: document_id
			}
		}
	}));
};

/* connect is the first interaction that should occur with the leap_client after defining your event
 * bindings. This function will generate a websocket connection with the server, ready to bind to a
 * document.
 */
leap_client.prototype.connect = function(address, _websocket) {
	try {
		if ( _websocket !== undefined ) {
				this._socket = _websocket;
		} else if ( window.WebSocket !== undefined ) {
				this._socket = new WebSocket(address);
		} else {
			return "no websocket support in this browser";
		}
	} catch(e) {
		return "socket connection failed: " + e.message;
	}

	var leap_obj = this;

	this._socket.onmessage = function(message) {
		var message_text = message.data;
		var message_obj;

		try {
			message_obj = JSON.parse(message_text);
		} catch (e) {
			leap_obj._dispatch_event.apply(leap_obj,
				[ leap_obj.EVENT_TYPE.ERROR,
					[ {
						error: {
							type: "ERR_PARSE_MSG",
							message: JSON.stringify(e.message) + " (" + e.lineNumber + "): " + message_text
						}
					} ] ]);
			return;
		}

		var err = leap_obj._process_message.apply(leap_obj, [ message_obj ]);
		if ( typeof(err) === "string" ) {
			leap_obj._dispatch_event.apply(leap_obj, [ leap_obj.EVENT_TYPE.ERROR, [ {
				error: {
					type: "ERR_INTERNAL_MODEL",
					message: err
				}
			} ] ]);
		}
	};

	this._socket.onclose = function() {
		leap_obj._dispatch_event.apply(leap_obj, [ leap_obj.EVENT_TYPE.DISCONNECT, [] ]);
	};

	this._socket.onopen = function() {
		leap_obj._dispatch_event.apply(leap_obj, [ leap_obj.EVENT_TYPE.CONNECT, arguments ]);
	};

	this._socket.onerror = function() {
		leap_obj._dispatch_event.apply(leap_obj, [ leap_obj.EVENT_TYPE.ERROR, [ {
			error: {
				type: "ERR_SOCKET",
				message: "socket connection error"
			}
		} ] ]);
	};
};

/* Close the connection to the document and halt all operations.
 */
leap_client.prototype.close = function() {
	if ( this._socket !== null && this._socket.readyState === 1 ) {
		this._socket.close();
		this._socket = null;
	}
	this._model = null;
};

/*--------------------------------------------------------------------------------------------------
 */

/* leap_apply is a function that applies a single transform to content and returns the result.
 */
var leap_apply = function(transform, content) {
	var num_delete = 0, to_insert = "";

	if ( typeof(transform.position) !== "number" ) {
		return content;
	}

	if ( typeof(transform.num_delete) === "number" ) {
		num_delete = transform.num_delete;
	}

	if ( undefined !== transform.insert ) {
		to_insert = new leap_str(transform.insert).str();
	}

	if (! ( content instanceof leap_str ) ) {
		content = new leap_str(content);
	}

	return content.u_str().slice(0, transform.position).join('') + to_insert +
		content.u_str().slice(transform.position + num_delete, content.u_str().length).join('');
};

leap_client.prototype.apply = leap_apply;

/*--------------------------------------------------------------------------------------------------
 */

try {
	if ( module !== undefined && typeof(module) === "object" ) {
		module.exports = {
			client : leap_client,
			apply : leap_apply,
			str: leap_str,
			_model : leap_model
		};
	}
} catch(e) {
}

/*--------------------------------------------------------------------------------------------------
 */

})();
