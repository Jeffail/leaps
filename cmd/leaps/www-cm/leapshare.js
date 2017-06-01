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
                  File Extensions to CodeMirror Languages
------------------------------------------------------------------------------*/
var langs = {
	as: "actionscript",
	txt: "asciidoc",
	s: "assembly_x86",
	S: "assembly_x86",
	asm: "assembly_x86",
	cpp: "c_cpp",
	hpp: "c_cpp",
	cx: "c_cpp",
	hx: "c_cpp",
	c: "c_cpp",
	h: "c_cpp",
	clj: "clojure",
	cljs: "clojure",
	edn: "clojure",
	coffee: "coffee",
	cs: "csharp",
	css: "css",
	dart: "dart",
	d: "d",
	erl: "erlang",
	hrl: "erlang",
	go: "golang",
	hs: "haskell",
	lhs: "haskell",
	html: "html",
	htm: "html",
	java: "java",
	js: "javascript",
	json: "json",
	jsp: "jsp",
	jl: "julia",
	tex: "latex",
	less: "less",
	lisp: "lisp",
	lsp: "lisp",
	l: "lisp",
	cl: "lisp",
	fasl: "lisp",
	lua: "lua",
	mk: "makefile",
	md: "markdown",
	m: "matlab",
	ml: "ocaml",
	pl: "perl",
	pm: "perl",
	t: "perl",
	pod: "perl",
	php: "php",
	ps: "powershell",
	py: "python",
	rb: "ruby",
	rs: "rust",
	rlib: "rust",
	sass: "sass",
	scss: "sass",
	scala: "scala",
	scm: "scheme",
	ss: "scheme",
	sh: "sh",
	bash: "sh",
	sql: "sql",
	vb: "vbscript",
	xml: "xml",
	yaml: "yaml",
	yml: "yaml",
};

/*------------------------------------------------------------------------------
                           List of CodeMirror Themes
------------------------------------------------------------------------------*/
var themes = {
	ambiance : "Ambiance",
	chaos : "Chaos",
	chrome : "Chrome",
	clouds : "Clouds",
	clouds_midnight : "Clouds/Midnight",
	cobalt : "Cobalt",
	crimson_editor : "Crimson",
	dawn : "Dawn",
	dreamweaver : "Dreamweaver",
	eclipse : "Eclipse",
	github : "Github",
	idle_fingers : "Idle Fingers",
	katzenmilch : "Katzenmilch",
	kuroir : "Kurior",
	merbivore : "Merbivore",
	merbivore_soft : "Merbivore Soft",
	mono_industrial : "Mono Industrial",
	monokai : "Monokai",
	pastel_on_dark : "Pastel on Dark",
	solarized_dark : "Solarized Dark",
	solarized_light : "Solarized Light",
	terminal : "Terminal",
	textmate : "Textmate",
	tomorrow : "Tomorrow",
	tomorrow_night_blue : "Tomorrow Night Blue",
	tomorrow_night_bright : "Tomorrow Night Bright",
	tomorrow_night_eighties : "Tomorrow Night Eighties",
	tomorrow_night : "Tomorrow Night",
	twilight : "Twilight",
	vibrant_ink : "Vibrant Ink",
	xcode : "XCode"
};

/*------------------------------------------------------------------------------
                        List of CodeMirror Key Mappings
------------------------------------------------------------------------------*/
var keymaps = {
	none : "Standard",
	vim : "Vim",
	emacs : "Emacs"
};

/*------------------------------------------------------------------------------
                              Global Variables
------------------------------------------------------------------------------*/

var cm_editor = null;
var leaps_client = null;
var username = "anon";

var users = {};

var theme = "dawn";
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

var join_new_document = function(document_id) {
	if ( leaps_client !== null ) {
		leaps_client.close();
		leaps_client = null;
	}

	if ( cm_editor !== null ) {
		// TODO: Clean up
	}

	users = {};

	// TODO: Create cm_editor
	cm_editor = CodeMirror(document.getElementById("editor"));

	var filetype = "asciidoc";
	try {
		var ext = document_id.substr(document_id.lastIndexOf(".") + 1);
		if ( typeof langs[ext] === 'string' ) {
			filetype = langs[ext];
		}
	} catch (e) {}

	leaps_client = new leap_client();
	leaps_client.bind_codemirror(cm_editor);

	leaps_client.on("error", function(err) {
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

function get_paths(after) {
	AJAX_REQUEST(window.location.pathname + "files", function(data) {
		try {
			var paths_list = JSON.parse(data);
			after(paths_list.paths, paths_list.users);
		} catch (e) {
			console.error("paths parse error", e);
		}
	}, function(code, message) {
		console.error("get_paths error", code, message);
	});
};

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

window.onload = function() {
	join_new_document("index.html");
};

})();
