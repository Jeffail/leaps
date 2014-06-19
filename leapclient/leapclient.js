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

/* _leap_model is an object designed to keep track of the inbound and outgoing transforms
 * for a local document, and updates the caller with the appropriate actions at each stage.
 *
 * _leap_model has three states:
 * 1. READY     - No pending sends, transforms received can be applied instantly to local document.
 * 2. SENDING   - Transforms are being sent and we're awaiting the corrected version of those
 *                transforms.
 * 3. BUFFERING - A corrected version has been received for our latest send but we're still waiting
 *                for the transforms that came before that send to be received before moving on.
 */
var _leap_model = function(base_version) {
	"use strict";

	this.READY = 1;
	this.SENDING = 2;
	this.BUFFERING = 3;

	this._leap_state = this.READY;

	this._corrected_version = 0;
	this._version = base_version;

	this._unapplied = [];
	this._unsent = [];
	this._sending = [];
};

/* _validate_transforms iterates an array of transform objects and validates that each transform
 * contains the correct fields. Returns an error message as a string if there was a problem.
 */
_leap_model.prototype._validate_transforms = function(transforms) {
	"use strict";

	for ( var i = 0, l = transforms.length; i < l; i++ ) {
		tform = transforms[i];

		if ( typeof(tform.position) !== "number" ) {
			tform.position = parseInt(tform.position);
			if ( isNaN(tform.position) ) {
				return "transform contained NaN value for position";
			}
		}
		if ( typeof(tform.num_delete) !== undefined && typeof(tform.num_delete) !== "number" ) {
			tform.num_delete = parseInt(tform.num_delete);
			if ( isNaN(tform.num_delete) ) {
				return "transform contained NaN value for num_delete";
			}
		}
		if ( typeof(tform.version) !== undefined && typeof(tform.version) !== "number" ) {
			tform.version = parseInt(tform.version);
			if ( isNaN(tform.version) ) {
				return "transform contained NaN value for version";
			}
		}
		if ( typeof(tform.insert) !== undefined ) {
			if ( typeof(tform.insert) !== "string" ) {
				return "transform contained non-string value for insert";
			}
		} else {
			tform.insert = "";
		}
	};
};

/* collide_transforms takes an unapplied transform from the server, and an unsent transform from the
 * client and modifies both transforms.
 *
 * The unapplied transform is fixed so that when applied to the local document is unaffected by the
 * unsent transform that has already been applied. The unsent transform is fixed so that it is
 * unaffected by the unapplied transform when submitted to the server.
 */
_leap_model.prototype._collide_transforms = function(unapplied, unsent) {
	"use strict";

	var earlier, later;

	if ( unapplied.position <= unsent.position ) {
		earlier = unapplied;
		later = unsent;
	} else {
		earlier = unsent;
		later = unapplied;
	}

	if ( earlier.num_delete === 0 ) {
		later.position += earlier.insert.length;
	} else if ( ( earlier.num_delete + earlier.position ) <= later.position ) {
		later.position += ( earlier.insert.length - earlier.num_delete );
	} else {
		var pos_gap = later.position - earlier.position;
		var over_hang = Math.min(later.insert.length, earlier.num_delete - pos_gap);
		var excess = Math.max(0, (earlier.num_delete - pos_gap));

		// earlier changes
		if ( excess > later.num_delete ) {
			earlier.num_delete += later.insert.length - later.num_delete;
			earlier.insert = earlier.insert + later.insert;
		} else {
			earlier.num_delete = pos_gap;
		}

		// later changes
		later.num_delete = Math.min(0, later.num_delete - excess);
		later.position = earlier.position + earlier.insert.length;
	}
};

/*--------------------------------------------------------------------------------------------------
 */

/* _resolve_state will prompt the leap_model to re-evalutate its current state for validity. If this
 * state is determined to no longer be appropriate then it will return an object containing the
 * following actions to be performed.
 */
_leap_model.prototype._resolve_state = function() {
	"use strict";

	switch (this._leap_state) {
	case this.READY:
	case this.SENDING:
		return;
	case this.BUFFERING:
		if ( this._version + this._unapplied.length >= (this._corrected_version - 1) ) {

			this._version += this._unapplied.length + this._sent.length;

			var to_collide = this._sent.concat(this._unsent);
			var unapplied = this._unapplied;

			this._unapplied = [];

			for ( var i = 0, l = unapplied.length; i < l; i++ ) {
				for ( var j = 0, l = to_collide.length; j < l; j++ ) {
					this._collide_transforms(unapplied[i], to_collide[j]);
				}
			}

			this._sent = [];

			if ( this._unsent.length > 0 ) {
				this._sent = this._unsent;
				this._unsent = [];

				for ( var i = 0, l = this._sent.length; i < l; i++ ) {
					this._sent[i].version = this._version + 1 + i;
				}

				this._leap_state = this.SENDING;
				return { send : this._sent, apply : unapplied };
			} else {
				this._leap_state = this.READY;
				return { apply : unapplied };
			}
		}
	}
	return {}
};

/* correct is the function to call following a "correction" from the server, this correction value
 * gives the model the information it needs to determine which changes are missing from our model
 * from before our submission was accepted.
 */
_leap_model.prototype.correct = function(version) {
	"use strict";

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
_leap_model.prototype.submit = function(transforms) {
	"use strict";

	switch (this._leap_state) {
	case this.READY:
		this._leap_state = this.SENDING;
		for ( var i = 0, l = transforms.length; i < l; i++ ) {
			transforms[i].version = this._version + i + 1;
		}
		this._sending = transforms;
		return { send : transforms };
	case this.BUFFERING:
	case this.SENDING:
		this._unsent = this._unsent.concat(transforms);
	}

	return {};
};

/* receive is the function to call when we have received transforms from our server. If we have
 * recently dispatched transforms and have yet to receive our correction then it is unsafe to apply
 * these changes to our local document, so the model will keep return these transforms to us when it
 * is known to be safe.
 */
_leap_model.prototype.receive = function(transforms) {
	"use strict";

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
var leap_client = function() {
	"use strict";

	this._socket = null;
	this._document_id = null;

	this._model = null;

	this.on_transform = null;
	this.on_document = null;
	this.on_connect = null;
	this.on_disconnect = null;
	this.on_error = null;
};

/* _process_message is a call that takes a server provided message object and decides the
 * appropriate action to take. If an error occurs during this process then an error message is
 * returned.
 */
leap_client.prototype._process_message = function(message) {
	"use strict";

	if ( message.response_type === undefined
	  || typeof(message.response_type) !== "string" ) {
		return "message received did not contain a valid type";
	}

	switch (message.response_type) {
	case "document":
		if ( null === message.leap_document
		  || "object" !== typeof(message.leap_document)
		  || "string" !== typeof(message.leap_document.id)
		  || "string" !== typeof(message.leap_document.title)
		  || "string" !== typeof(message.leap_document.description)
		  || "string" !== typeof(message.leap_document.content) ) {
			return "message document type contained invalid document object";
		}
		if ( !(message.version > 0) ) {
			return "message document received but without valid version";
		}
		if ( this._document_id !== message.leap_document.id ) {
			return "received unexpected document, id was mismatched";
		}
		this._model = new _leap_model(message.version);
		this.on_document(message.leap_document);
		break;
	case "transforms":
		if ( this._model === null ) {
			return "transforms were received before initialization";
		}
		if ( !(message.transforms instanceof Array) ) {
			return "received non array transforms";
		}
		var validate_error = this._model._validate_transforms(message.transforms);
		if ( validate_error !== undefined ) {
			return "received transforms with error: " + validate_error;
		}
		var action_obj = this._model.receive(message.transforms);
		if ( action_obj.error !== undefined ) {
			return "model failed to receive transforms: " + action_obj.error;
		}
		if ( action_obj.apply !== undefined && action_obj.apply instanceof Array ) {
			if ( typeof(this.on_transform) === "function" ) {
				for ( var i = 0, l = action_obj.apply.length; i < l; i++ ) {
					this.on_transform(action_obj[i]);
				}
			}
		}
		if ( action_obj.send !== undefined && action_obj.send instanceof Array ) {
			this._socket.send(JSON.stringify({
				command : "submit",
				transforms : action_obj.send
			}));
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
		var action_obj = this._model.correct(message.version);
		if ( action_obj.error !== undefined ) {
			return "model failed to correct: " + action_obj.error;
		}
		if ( action_obj.apply !== undefined && action_obj.apply instanceof Array ) {
			if ( typeof(this.on_transform) === "function" ) {
				for ( var i = 0, l = action_obj.apply.length; i < l; i++ ) {
					this.on_transform(action_obj[i]);
				}
			}
		}
		if ( action_obj.send !== undefined && action_obj.send instanceof Array ) {
			this._socket.send(JSON.stringify({
				command : "submit",
				transforms : action_obj.send
			}));
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
		break;
	default:
		return "message received was not a recognised type"
	}
};

/* send_transform is the function to call to send a transform off to the server. To keep the local
 * document responsive this transform should be applied to the document straight away. The
 * leap_client will decide when it is appropriate to dispatch the transform, and will manage
 * internally how incoming messages should be altered to account for the fact that the local
 * change was made out of order.
 */
leap_client.prototype.send_transform = function(transform) {
	"use strict";

	if ( this._model === null ) {
		return "leap_client must be initialized and joined to a document before submitting transforms"
	}

	action_obj = this._model.submit([ transform ]);
	if ( action_obj.error !== undefined ) {
		return "model failed to submit: " + action_obj.error;
	}
	if ( action_obj.apply !== undefined && action_obj.apply instanceof Array ) {
		if ( typeof(this.on_transform) === "function" ) {
			for ( var i = 0, l = action_obj.apply.length; i < l; i++ ) {
				this.on_transform(action_obj[i]);
			}
		}
	}
	if ( action_obj.send !== undefined && action_obj.send instanceof Array ) {
		this._socket.send(JSON.stringify({
			command : "submit",
			transforms : action_obj.send
		}));
	}
};

/* join_document prompts the client to request to join a document from the server. It will return an
 * error message if there is a problem with the request.
 */
leap_client.prototype.join_document = function(id) {
	"use strict";

	if ( this._socket === null || this._socket.readyState !== 1 ) {
		return "leap_client is not currently connected";
	}

	if ( typeof(id) !== "string" ) {
		return "document id was not a string type";
	}

	if ( this._document_id !== null ) {
		return "a leap_client can only join a single document";
	}

	this._document_id = id;

	this._socket.send(JSON.stringify({
		command : "find",
		document_id : this._document_id
	}));
};

/* connect is the first interaction that should occur with the leap_client after defining your event
 * bindings. This function will generate a websocket connection with the server, ready to bind to a
 * document.
 */
leap_client.prototype.connect = function(address, _websocket) {
	"use strict";

	try {
		if ( _websocket !== undefined ) {
				this._socket = new _websocket(address);
		} else if ( window.WebSocket !== undefined ) {
				this._socket = new WebSocket(address);
		} else {
			return "no websocket support in this browser";
		}
	} catch(e) {
		return "socket connection failed: " + e.message;
	}

	var leap_obj = this;

	this._socket.onmessage = function(e) {
		var message_text = e.data;
		try {
			var message_obj = JSON.parse(message_text);

			var err = leap_obj._process_message.apply(leap_obj, [ message_obj ]);
			if ( typeof(err) === "string" ) {
				if ( typeof(leap_obj.on_error) === "function" ) {
					leap_obj.on_error.apply(leap_obj, [ err ]);
				}
			}
		} catch (e) {
			if ( typeof(leap_obj.on_error) === "function" ) {
				leap_obj.on_error.apply(leap_obj, [ JSON.stringify(e.message) + ": " + message_text ]);
			}
		}
	};

	this._socket.onclose = function() {
		if ( typeof(leap_obj.on_disconnect) === "function" ) {
			leap_obj.on_disconnect.apply(leap_obj, []);
		}
	};

	this._socket.onopen = function() {
		if ( typeof(leap_obj.on_connect) === "function" ) {
			leap_obj.on_connect.apply(leap_obj, arguments);
		}
	};

	this._socket.onerror = function() {
		if ( typeof(leap_obj.on_error) === "function" ) {
			leap_obj.on_error.apply(leap_obj, arguments);
		}
	};
};

/*--------------------------------------------------------------------------------------------------
 */

if ( module !== undefined && typeof(module) === "object" ) {
	module.exports = leap_client;
}

/*--------------------------------------------------------------------------------------------------
 */
