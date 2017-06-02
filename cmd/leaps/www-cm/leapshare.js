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

var theme = "zenburn";// "default";
var binding = "none";
var useTabs = true;
var wrapLines = true;

/*------------------------------------------------------------------------------
                      Leaps Editor User Cursor Helpers
------------------------------------------------------------------------------*/

var HSVtoRGB = function(h, s, v) {
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
};

var hash = function(str) {
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
};

var user_id_to_color = function(user_id) {
	var id_hash = hash(user_id);
	if ( id_hash < 0 ) {
		id_hash = id_hash * -1;
	}

	var hue = ( id_hash % 10000 ) / 10000;
	var rgb = HSVtoRGB(hue, 1, 0.8);

	return "rgba(" + rgb.r + ", " + rgb.g + ", " + rgb.b + ", 1)";
};


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
		if ( leaps_client !== null ) {
			last_document_joined = "";
		}
	});

	leaps_client.on("connect", function() {
		leaps_client.join_document(username, "", document_id);
	});

	leaps_client.on("document", function() {
		last_document_joined = document_id;
	});

	leaps_client.on("user", function(user_update) {
		if ( 'string' === typeof user_update.message.content ) {
			// TODO: Show message
		}

		var refresh_user_list = !users.hasOwnProperty(user_update.client.session_id);
		users[user_update.client.session_id] = user_update.client.user_id;

		if ( typeof user_update.message.active === 'boolean' && !user_update.message.active ) {
			refresh_user_list = true;
			delete users[user_update.client.session_id]
		}

		if ( refresh_user_list ) {
			// TODO: Refresh user list
		}
	});

	var protocol = window.location.protocol === "http:" ? "ws:" : "wss:";
	leaps_client.connect(protocol + "//" + window.location.host + window.location.pathname + "leaps/ws");
};

/*------------------------------------------------------------------------------
                       File Path Acquire and Listing
------------------------------------------------------------------------------*/

function inject_paths(root, paths_list) {
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
					children: new_children
				});
				ptr = new_children;
			} else {
				ptr = next_ptr;
			}
		}

		ptr.push({
			name: split_path[k],
			path: paths_list[i]
		});
	}

	root.children = children;
}

function get_paths() {
	AJAX_REQUEST(window.location.pathname + "files", function(data) {
		try {
			var data_arrays = JSON.parse(data);
			inject_paths(file_paths, data_arrays.paths);
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
	Vue.component('item', {
		template: '#file-template',
		props: {
			model: Object
		},
		data: function () {
			return {
				open: true
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

	CodeMirror.modeURL = "cm/mode/%N/%N.js";

	get_paths();
	setInterval(get_paths, 5000);
};

})();
