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

try {
	// Returns the underlying unicode codepoint array. We define this with eval
	// in order to support older browsers.
	leap_str.prototype.u_str = eval('(function() { if ( undefined === this._u_str ) { this._u_str = [...this._str]; } return this._u_str; })');
} catch (e) {
	leap_str.prototype.u_str = function() { return this._str.split(''); };
	console.warn("JS Engine without ES6 support detected: this will result in" +
		" unexpected behaviour when working with larger unicode codepoints.");
}

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
var leap_model = function(base_version) {
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

/* _validate_updates iterates an array of user update objects and validates that each update
 * contains the correct fields. Returns an error message as a string if there was a problem.
 */
leap_model.prototype._validate_updates = function(user_updates) {
	for ( var i = 0, l = user_updates.length; i < l; i++ ) {
		var update = user_updates[i];

		if ( undefined === update.client ||
			"object" !== typeof(update.client) ) {
			return "update did not contain valid client object: " + JSON.stringify(update);
		}

		if ( undefined === update.message ||
			"object" !== typeof(update.message) ) {
			return "update did not contain valid message object: " + JSON.stringify(update);
		}

		var message = update.message;

		if ( undefined !== message.position &&
		    "number" !== typeof(message.position) ) {
			message.position = parseInt(message.position);
			if ( isNaN(message.position) ) {
				return "update message contained NaN value for position: " + JSON.stringify(update);
			}
		}
		if ( undefined !== message.content &&
		    "string" !== typeof(message.content) ) {
			return "update message contained invalid type for content: " + JSON.stringify(update);
		}
		if ( undefined !== message.active &&
		    "boolean" !== typeof(message.active) ) {
			if ("string" !== typeof(message.active)) {
				return "update message contained invalid type for active: " + JSON.stringify(update);
			}
			message.active = ("true" === message.active);
		}

		var client = update.client;

		if ( undefined === client.user_id ||
		    "string" !== typeof(client.user_id) ) {
			return "update client contained invalid type for user_id: " + JSON.stringify(update);
		}
		if ( undefined === client.session_id ||
		    "string" !== typeof(client.session_id) ) {
			return "update client contained invalid type for session_id: " + JSON.stringify(update);
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
		later.num_delete = Math.min(0, later.num_delete - excess);
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
	this._document_id = null;

	this._model = null;

	this._cursor_position = 0;

	this.EVENT_TYPE = {
		CONNECT: "connect",
		DISCONNECT: "disconnect",
		DOCUMENT: "document",
		TRANSFORMS: "transforms",
		USER: "user",
		ERROR: "error"
	};

	// Milliseconds period between cursor position updates to server
	this._POSITION_POLL_PERIOD = 500;

	this._events = {};
};

/* subscribe_event, attach a function to an event of the leap_client. Use this to subscribe to
 * transforms, document responses and errors etc. Returns a string if an error occurrs.
 */
leap_client.prototype.subscribe_event = function(name, subscriber) {
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

/* on - an alias for subscribe_event.
 */
leap_client.prototype.on = leap_client.prototype.subscribe_event;

/* clear_subscribers, removes all functions subscribed to an event.
 */
leap_client.prototype.clear_subscribers = function(name) {
	this._events[name] = [];
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
};

/* _do_action is a call that acts accordingly provided an action_obj from our leap_model.
 */
leap_client.prototype._do_action = function(action_obj) {
	if ( action_obj.error !== undefined ) {
		return action_obj.error;
	}
	if ( action_obj.apply !== undefined && action_obj.apply instanceof Array ) {
		this._dispatch_event(this.EVENT_TYPE.TRANSFORMS, [ action_obj.apply ]);
	}
	if ( action_obj.send !== undefined && action_obj.send instanceof Object ) {
		this._socket.send(JSON.stringify({
			command : "submit",
			transform : action_obj.send
		}));
	}
};

/* _process_message is a call that takes a server provided message object and decides the
 * appropriate action to take. If an error occurs during this process then an error message is
 * returned.
 */
leap_client.prototype._process_message = function(message) {
	var validate_error, action_obj, action_err;

	if ( message.response_type === undefined || typeof(message.response_type) !== "string" ) {
		return "message received did not contain a valid type";
	}

	switch (message.response_type) {
	case "document":
		if ( null === message.leap_document ||
		   "object" !== typeof(message.leap_document) ||
		   "string" !== typeof(message.leap_document.id) ||
		   "string" !== typeof(message.leap_document.content) ) {
			return "message document type contained invalid document object";
		}
		if ( message.version <= 0 ) {
			return "message document received but without valid version";
		}
		if ( this._document_id !== null && this._document_id !== message.leap_document.id ) {
			return "received unexpected document, id was mismatched: " +
				this._document_id + " != " + message.leap_document.id;
		}
		this.document_id = message.leap_document.id;
		this._model = new leap_model(message.version);
		this._dispatch_event(this.EVENT_TYPE.DOCUMENT, [ message.leap_document ]);
		break;
	case "transforms":
		if ( this._model === null ) {
			return "transforms were received before initialization";
		}
		if ( !(message.transforms instanceof Array) ) {
			return "received non array transforms";
		}
		validate_error = this._model._validate_transforms(message.transforms);
		if ( validate_error !== undefined ) {
			return "received transforms with error: " + validate_error;
		}
		action_obj = this._model.receive(message.transforms);
		action_err = this._do_action(action_obj);
		if ( action_err !== undefined ) {
			return "failed to receive transforms: " + action_err;
		}
		break;
	case "update":
		if ( null === message.user_updates ||
		   !(message.user_updates instanceof Array) ) {
			return "message update type contained invalid user_updates";
		}
		validate_error = this._model._validate_updates(message.user_updates);
		if ( validate_error !== undefined ) {
			return "received updates with error: " + validate_error;
		}
		for ( var i = 0, l = message.user_updates.length; i < l; i++ ) {
			this._dispatch_event(this.EVENT_TYPE.USER, [ message.user_updates[i] ]);
		}
		break;
	case "correction":
		if ( this._model === null ) {
			return "correction was received before initialization";
		}
		if ( typeof(message.version) !== "number" ) {
			message.version = parseInt(message.version);
			if ( isNaN(message.version) ) {
				return "correction received was NaN";
			}
		}
		action_obj = this._model.correct(message.version);
		action_err = this._do_action(action_obj);
		if ( action_err !== undefined ) {
			return "model failed to correct: " + action_err;
		}
		break;
	case "error":
		if ( this._socket !== null ) {
			this._socket.close();
		}
		if ( typeof(message.error) === "string" ) {
			return message.error;
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
leap_client.prototype.send_transform = function(transform) {
	if ( this._model === null ) {
		return "leap_client must be initialized and joined to a document before submitting transforms";
	}

	var validate_error = this._model._validate_transforms([ transform ]);
	if ( validate_error !== undefined ) {
		return validate_error;
	}

	var action_obj = this._model.submit(transform);
	var action_err = this._do_action(action_obj);
	if ( action_err !== undefined ) {
		return "model failed to submit: " + action_err;
	}
};

/* send_message - send a text message out to all other users connected to your shared document.
 */
leap_client.prototype.send_message = function(message) {
	if ( "string" !== typeof(message) ) {
		return "must supply message as a valid string value";
	}

	this._socket.send(JSON.stringify({
		command:  "update",
		message: message,
		position: this._cursor_position
	}));
};

/* update_cursor is the function to call to send the server (and all other clients) an update to your
 * current cursor position in the document, this shows others where your point of interest is in the
 * shared document.
 */
leap_client.prototype.update_cursor = function(position) {
	if ( "number" !== typeof(position) ) {
		return "must supply position as a valid integer value";
	}

	this._cursor_position = position;
	this._socket.send(JSON.stringify({
		command:  "update",
		position: this._cursor_position
	}));
};

/* join_document prompts the client to request to join a document from the server. It will return an
 * error message if there is a problem with the request.
 */
leap_client.prototype.join_document = function(user_id, token, document_id) {
	if ( this._socket === null || this._socket.readyState !== 1 ) {
		return "leap_client is not currently connected";
	}

	if ( typeof(user_id) !== "string" ) {
		return "user id was not a string type";
	}

	if ( typeof(token) !== "string" ) {
		return "token was not a string type";
	}

	if ( typeof(document_id) !== "string" ) {
		return "document id was not a string type";
	}

	if ( this._document_id !== null ) {
		return "a leap_client can only join a single document";
	}

	this._document_id = document_id;

	this._socket.send(JSON.stringify({
		command : "edit",
		user_id : user_id,
		token : token,
		document_id : this._document_id
	}));
};

/* create_document submits content to be created into a fresh document and then binds to that
 * document.
 */
leap_client.prototype.create_document = function(user_id, token, content) {
	if ( this._socket === null || this._socket.readyState !== 1 ) {
		return "leap_client is not currently connected";
	}

	if ( typeof(user_id) !== "string" ) {
		return "user id was not a string type";
	}

	if ( typeof(token) !== "string" ) {
		return "token was not a string type";
	}

	if ( typeof(content) !== "string" ) {
		return "new document requires valid content (can be empty)";
	}

	if ( this._document_id !== null ) {
		return "a leap_client can only join a single document";
	}

	this._socket.send(JSON.stringify({
		command : "create",
		user_id : user_id,
		token : token,
		leap_document : {
			content : content
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
					[ JSON.stringify(e.message) + " (" + e.lineNumber + "): " + message_text ] ]);
			return;
		}

		var err = leap_obj._process_message.apply(leap_obj, [ message_obj ]);
		if ( typeof(err) === "string" ) {
			leap_obj._dispatch_event.apply(leap_obj, [ leap_obj.EVENT_TYPE.ERROR, [ err ] ]);
		}
	};

	this._socket.onclose = function() {
		if ( undefined !== leap_obj._heartbeat ) {
			clearTimeout(leap_obj._heartbeat);
		}
		leap_obj._dispatch_event.apply(leap_obj, [ leap_obj.EVENT_TYPE.DISCONNECT, [] ]);
	};

	this._socket.onopen = function() {
		leap_obj._heartbeat = setInterval(function() {
			leap_obj._socket.send(JSON.stringify({
				command : "ping"
			}));
		}, 5000); // MAGIC NUMBER OH GOD, we should have a config object.
		leap_obj._dispatch_event.apply(leap_obj, [ leap_obj.EVENT_TYPE.CONNECT, arguments ]);
	};

	this._socket.onerror = function() {
		if ( undefined !== leap_obj._heartbeat ) {
			clearTimeout(leap_obj._heartbeat);
		}
		leap_obj._dispatch_event.apply(leap_obj, [ leap_obj.EVENT_TYPE.ERROR, [ "socket connection error" ] ]);
	};
};

/* Close the connection to the document and halt all operations.
 */
leap_client.prototype.close = function() {
	if ( undefined !== this._heartbeat ) {
		clearTimeout(this._heartbeat);
	}
	if ( this._socket !== null && this._socket.readyState === 1 ) {
		this._socket.close();
		this._socket = null;
	}
	this.document_id = undefined;
	this._model = undefined;
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
