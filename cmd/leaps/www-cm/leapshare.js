(function() {
"use strict";

// Cookies.get(key)
// Cookies.set(key, value, { path: '' });

/*!
 * JavaScript Cookie v2.1.1
 * https://github.com/js-cookie/js-cookie
 *
 * Copyright 2006, 2015 Klaus Hartl & Fagner Brack
 * Released under the MIT license
 */
;(function (factory) {
	if (typeof define === 'function' && define.amd) {
		define(factory);
	} else if (typeof exports === 'object') {
		module.exports = factory();
	} else {
		var OldCookies = window.Cookies;
		var api = window.Cookies = factory();
		api.noConflict = function () {
			window.Cookies = OldCookies;
			return api;
		};
	}
}(function () {
	function extend () {
		var i = 0;
		var result = {};
		for (; i < arguments.length; i++) {
			var attributes = arguments[ i ];
			for (var key in attributes) {
				result[key] = attributes[key];
			}
		}
		return result;
	}

	function init (converter) {
		function api (key, value, attributes) {
			var result;
			if (typeof document === 'undefined') {
				return;
			}

			// Write

			if (arguments.length > 1) {
				attributes = extend({
					path: '/'
				}, api.defaults, attributes);

				if (typeof attributes.expires === 'number') {
					var expires = new Date();
					expires.setMilliseconds(expires.getMilliseconds() + attributes.expires * 864e+5);
					attributes.expires = expires;
				}

				try {
					result = JSON.stringify(value);
					if (/^[\{\[]/.test(result)) {
						value = result;
					}
				} catch (e) {}

				if (!converter.write) {
					value = encodeURIComponent(String(value))
						.replace(/%(23|24|26|2B|3A|3C|3E|3D|2F|3F|40|5B|5D|5E|60|7B|7D|7C)/g, decodeURIComponent);
				} else {
					value = converter.write(value, key);
				}

				key = encodeURIComponent(String(key));
				key = key.replace(/%(23|24|26|2B|5E|60|7C)/g, decodeURIComponent);
				key = key.replace(/[\(\)]/g, escape);

				return (document.cookie = [
					key, '=', value,
					attributes.expires && '; expires=' + attributes.expires.toUTCString(), // use expires attribute, max-age is not supported by IE
					attributes.path    && '; path=' + attributes.path,
					attributes.domain  && '; domain=' + attributes.domain,
					attributes.secure ? '; secure' : ''
				].join(''));
			}

			// Read

			if (!key) {
				result = {};
			}

			// To prevent the for loop in the first place assign an empty array
			// in case there are no cookies at all. Also prevents odd result when
			// calling "get()"
			var cookies = document.cookie ? document.cookie.split('; ') : [];
			var rdecode = /(%[0-9A-Z]{2})+/g;
			var i = 0;

			for (; i < cookies.length; i++) {
				var parts = cookies[i].split('=');
				var name = parts[0].replace(rdecode, decodeURIComponent);
				var cookie = parts.slice(1).join('=');

				if (cookie.charAt(0) === '"') {
					cookie = cookie.slice(1, -1);
				}

				try {
					cookie = converter.read ?
						converter.read(cookie, name) : converter(cookie, name) ||
						cookie.replace(rdecode, decodeURIComponent);

					if (key === name) {
						result = cookie;
						break;
					}

					if (!key) {
						result[name] = cookie;
					}
				} catch (e) { console.error(e); }
			}

			return result;
		}

		api.set = api;
		api.get = function (key) {
			return api(key);
		};
		api.defaults = {};

		api.remove = function (key, attributes) {
			api(key, '', extend(attributes, {
				expires: -1
			}));
		};

		api.withConverter = init;

		return api;
	}

	return init(function () {});
}));

/*------------------------------------------------------------------------------
                        List of CodeMirror Key Mappings
------------------------------------------------------------------------------*/

var keymaps = {
	none : "default",
	vim : "vim",
	emacs : "emacs",
	sublime : "sublime"
};

/*------------------------------------------------------------------------------
                              Global Variables
------------------------------------------------------------------------------*/

var cm_editor = null;
var leaps_client = null;
var username = "anon";

var users = {};
var file_paths = {
	root: true,
	name: "Files",
	path: "Files",
	children: []
};
var collapsed_dirs = {};

var messages_obj = {
	messages: []
};

var theme = "zenburn";// "default";
var binding = "none";
var useTabs = true;
var wrapLines = true;

/*------------------------------------------------------------------------------
                        Leaps Editor Bootstrapping
------------------------------------------------------------------------------*/

var last_document_joined = "";

function configure_codemirror() {
	cm_editor.setOption("theme", theme);
	cm_editor.setOption("keyMap", keymaps[binding]);
}

var join_new_document = function(document_id) {
	if ( leaps_client !== null ) {
		leaps_client.close();
		leaps_client = null;
	}

	if ( cm_editor !== null ) {
		cm_editor.getWrapperElement().parentNode.removeChild(cm_editor.getWrapperElement());
		cm_editor = null;
	}

	users = {};

	var default_options = CodeMirror.defaults;
	default_options.lineNumbers = true;

	cm_editor = CodeMirror(document.getElementById("editor"), default_options);
	cm_editor.options.readOnly = true;

	configure_codemirror();

	try {
		var ext = document_id.substr(document_id.lastIndexOf(".") + 1);
		var info = CodeMirror.findModeByExtension(ext);
		if (info.mode) {
			cm_editor.setOption("mode", info.mime);
			CodeMirror.autoLoadMode(cm_editor, info.mode);
		}
	} catch (e) {}

	leaps_client = new leap_client();
	leaps_client.bind_codemirror(cm_editor);

	leaps_client.on("error", function(err) {
		show_err_message(err);
		if ( cm_editor !== null ) {
			cm_editor.options.readOnly = true;
		}
		if ( leaps_client !== null ) {
			console.error(err);
			leaps_client.close();
			leaps_client = null;
		}
	});

	leaps_client.on("disconnect", function(err) {
		show_sys_message("Closed " + last_document_joined);
		if ( cm_editor !== null ) {
			cm_editor.options.readOnly = true;
		}
		if ( leaps_client !== null ) {
			last_document_joined = "";
		}
	});

	leaps_client.on("connect", function() {
		leaps_client.join_document(username, "", document_id);
	});

	leaps_client.on("document", function() {
		cm_editor.options.readOnly = false;
		last_document_joined = document_id;
		show_sys_message("Opened " + document_id);
	});

	leaps_client.on("user", function(user_update) {
		if ( 'string' === typeof user_update.message.content ) {
			show_user_message(user_update.client.user_id, user_update.message.content);
		}

		var refresh_user_list = !users.hasOwnProperty(user_update.client.session_id);
		users[user_update.client.session_id] = user_update.client.user_id;

		if ( typeof user_update.message.active === 'boolean' && !user_update.message.active ) {
			refresh_user_list = true;
			delete users[user_update.client.session_id];
		}

		if ( refresh_user_list ) {
			// TODO: Refresh user list
		}
	});

	var protocol = window.location.protocol === "http:" ? "ws:" : "wss:";
	leaps_client.connect(protocol + "//" + window.location.host + window.location.pathname + "leaps/ws");
};

/*------------------------------------------------------------------------------
                                  Messages
------------------------------------------------------------------------------*/

function clip_messages() {
	if ( messages_obj.messages.length > 200 ) {
		messages_obj.messages = messages_obj.messages.slice(-200);
	}
}

function show_user_message(username, content) {
	var now = new Date();
	messages_obj.messages.push({
		timestamp: now.toLocaleTimeString(),
		name: username,
		content: content
	});
	clip_messages();
}

function show_sys_message(content) {
	var now = new Date();
	messages_obj.messages.push({
		timestamp: now.toLocaleTimeString(),
		is_sys: true,
		name: "INFO",
		content: content
	});
	clip_messages();
}

function show_err_message(content) {
	var now = new Date();
	messages_obj.messages.push({
		timestamp: now.toLocaleTimeString(),
		is_err: true,
		name: "ERROR",
		content: content
	});
	clip_messages();
}

/*------------------------------------------------------------------------------
                       File Path Acquire and Listing
------------------------------------------------------------------------------*/

function inject_paths(root, paths_list, users_obj) {
	var i = 0, l = 0, j = 0, k = 0, m = 0, n = 0;

	var children = [];

	for ( i = 0, l = paths_list.length; i < l; i++ ) {
		var ptr = children;
		var split_path = paths_list[i].split('/');

		for ( j = 0, k = split_path.length - 1; j < k; j++ ) {
			var next_ptr = null;
			for ( m = 0, n = ptr.length; m < n; m++ ) {
				if ( ptr[m].name === split_path[j] ) {
					next_ptr = ptr[m].children;
				}
			}
			if ( next_ptr === null ) {
				var new_children = [];
				ptr.push({
					name: split_path[j],
					path: split_path.slice(0, j+1).join('/'),
					children: new_children
				});
				ptr = new_children;
			} else {
				ptr = next_ptr;
			}
		}

		var users_count = 0;
		if ( users_obj[paths_list[i]] !== undefined ) {
			users_count = users_obj[paths_list[i]].length;
		}
		ptr.push({
			name: split_path[k],
			path: paths_list[i],
			num_users: users_count
		});
	}

	root.children = children;
}

function get_paths() {
	AJAX_REQUEST(window.location.pathname + "files", function(data) {
		try {
			var data_arrays = JSON.parse(data);
			inject_paths(file_paths, data_arrays.paths, data_arrays.users);
		} catch (e) {
			console.error("paths parse error", e);
		}
	}, function(code, message) {
		console.error("get_paths error", code, message);
	});
}

var AJAX_REQUEST = function(path, onsuccess, onerror, data) {
	var xmlhttp;
	if (window.XMLHttpRequest)  {
		// code for IE7+, Firefox, Chrome, Opera, Safari
		xmlhttp=new XMLHttpRequest();
	} else {
		// code for IE6, IE5
		xmlhttp=new ActiveXObject("Microsoft.XMLHTTP");
	}

	xmlhttp.onreadystatechange = function() {
		if ( xmlhttp.readyState == 4 ) { // DONE
			if ( xmlhttp.status == 200 ) {
				onsuccess(xmlhttp.responseText);
			} else {
				onerror(xmlhttp.status, xmlhttp.responseText);
			}
		}
	};

	if ( 'undefined' !== typeof data ) {
		xmlhttp.open("POST", path, true);
		xmlhttp.setRequestHeader("Content-Type","text/plain");
		xmlhttp.send(data);
	} else {
		xmlhttp.open("GET", path, true);
		xmlhttp.send();
	}
};

/*------------------------------------------------------------------------------
                           Vue.js UI bindings
------------------------------------------------------------------------------*/

window.onload = function() {
	// define the item component
	Vue.component('file-item', {
		template: '#file-template',
		props: {
			model: Object
		},
		data: function () {
			return {
				open: !collapsed_dirs[this.model.path]
			};
		},
		computed: {
			is_folder: function () {
				return this.model.children &&
					this.model.children.length;
			}
		},
		methods: {
			toggle: function () {
				if (this.is_folder) {
					this.open = !this.open;
					if ( !this.open ) {
						collapsed_dirs[this.model.path] = true;
					} else {
						delete collapsed_dirs[this.model.path];
					}
					Cookies.set("collapsed_dirs", collapsed_dirs, { path: '' });
				} else {
					join_new_document(this.model.path);
				}
			}
		}
	});

	(new Vue({
		el: '#file-list',
		data: {
			file_data: file_paths
		}
	}));

	(new Vue({
		el: '#message-list',
		data: messages_obj
	}));

	var chat_bar = document.getElementById("chat-bar");
	chat_bar.onkeypress = function(e) {
		if ( typeof e !== 'object' ) {
			e = window.event;
		}
		var keyCode = e.keyCode || e.which;
		if ( keyCode == '13' && chat_bar.value.length > 0 ) {
			if ( leaps_client !== null ) {
				leaps_client.send_message(chat_bar.value);
				show_user_message(username, chat_bar.value);
				chat_bar.value = "";
				return false;
			} else {
				show_err_message(
					"You must open a document in order to send messages, " +
					"they will be readable by other users editing that document"
				);
				return true;
			}
		}
	};

	CodeMirror.modeURL = "cm/mode/%N/%N.js";

	try {
		collapsed_dirs = JSON.parse(Cookies.get("collapsed_dirs"));
	} catch (e) {}
	get_paths();
	setInterval(get_paths, 1000);
};

})();
