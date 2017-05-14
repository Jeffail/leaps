(function() {
"use strict";

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

/*--------------------------------------------------------------------------------------------------
                             File Extensions to ACE Editor Languages
--------------------------------------------------------------------------------------------------*/
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

/*--------------------------------------------------------------------------------------------------
                                  List of ACE Editor Themes
--------------------------------------------------------------------------------------------------*/
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

/*--------------------------------------------------------------------------------------------------
                                  List of ACE Editor Key Mappings
--------------------------------------------------------------------------------------------------*/
var keymaps = {
	none : "Standard",
	vim : "Vim",
	emacs : "Emacs"
};

/*--------------------------------------------------------------------------------------------------
                                        Global Variables
--------------------------------------------------------------------------------------------------*/
var ace_editor = null;
var leaps_client = null;
var username = "anon";

var users = {};

var theme = "dawn";
var binding = "none";
var useTabs = true;
var wrapLines = true;

/*--------------------------------------------------------------------------------------------------
                                    ACE Editor Configuration
--------------------------------------------------------------------------------------------------*/
var configure_ace_editor = function() {
	if ( ace_editor === null ) {
		return;
	}
	ace_editor.setTheme("ace/theme/" + theme);

	var map = "";
	if ( binding !== "none" ) {
		map = "ace/keyboard/" + binding;
	}
	ace_editor.setKeyboardHandler(map);
	ace_editor.getSession().setUseSoftTabs(!useTabs);
	ace_editor.getSession().setUseWrapMode(wrapLines);
};

/*--------------------------------------------------------------------------------------------------
                                 Leaps Editor User Cursor Helpers
--------------------------------------------------------------------------------------------------*/
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

var oob_elements = [];

var ACE_cursor_clear_handler = function() {
	for ( var i = 0, l = oob_elements.length; i < l; i++ ) {
		document.body.removeChild(oob_elements[i]);
	}
	oob_elements = [];
};

var ACE_cursor_handler = function(user_id, session_id, lineHeight, top, left, row, column) {
	var colorStyle = user_id_to_color(session_id);

	// Needs IE9
	var editor_bounds = ace_editor.container.getBoundingClientRect();

	var editor_width = ace_editor.getSession().getScreenWidth();
	var editor_height = ace_editor.getSession().getScreenLength();

	var triangle_height = 20;
	var triangle_opacity = 0.5;
	var ball_width = 8;

	var height = lineHeight;
	var extra_height = 6;
	var width = 2;

	var tag_height = 30;

	var create_ptr_ele = function() {
		var top_ptr_ele = document.createElement('div');
		top_ptr_ele.style.opacity = 0.7 + '';
		top_ptr_ele.style.position = 'absolute';
		top_ptr_ele.style.width = '0';
		top_ptr_ele.style.height = '0';
		top_ptr_ele.style.zIndex = '99';

		return top_ptr_ele;
	};
	if ( top < 0 ) {
		var top_ptr_ele = create_ptr_ele();
		top_ptr_ele.style.top = editor_bounds.top + 'px';
		top_ptr_ele.style.left = (editor_bounds.left + triangle_height) + 'px';
		top_ptr_ele.style.borderBottom = triangle_height + 'px solid ' + colorStyle;
		top_ptr_ele.style.borderLeft = (triangle_height/2) + 'px solid transparent';
		top_ptr_ele.style.borderRight = (triangle_height/2) + 'px solid transparent';

		document.body.appendChild(top_ptr_ele);
		oob_elements.push(top_ptr_ele);
	} else if ( top > editor_bounds.height ) {
		var bottom_ptr_ele = create_ptr_ele();
		bottom_ptr_ele.style.top = (editor_bounds.top + editor_bounds.height - triangle_height) + 'px';
		bottom_ptr_ele.style.left = (editor_bounds.left + triangle_height) + 'px';
		bottom_ptr_ele.style.borderTop = triangle_height + 'px solid ' + colorStyle;
		bottom_ptr_ele.style.borderLeft = (triangle_height/2) + 'px solid transparent';
		bottom_ptr_ele.style.borderRight = (triangle_height/2) + 'px solid transparent';

		document.body.appendChild(bottom_ptr_ele);
		oob_elements.push(bottom_ptr_ele);
	}
	var left_ptr_obj = '';
	if ( left > editor_bounds.width ) {
		left_ptr_obj = '<div style="' +
			'position: absolute; ' +
			'width: 0; height: 0; z-index: 99; ' +
			'top: ' + top + 'px; ' +
			'left: 0; border-top: ' + (triangle_height/2) + 'px solid transparent; ' +
			'border-left: ' + (triangle_height/3) + 'px solid ' + colorStyle + '; ' +
			'border-bottom: ' + (triangle_height/2) + 'px solid transparent; ' +
			'opacity: 0.7; ' +
			'"></div>';
	}
	var tag_obj = '<div style="background-color: ' + colorStyle +
		'; opacity: 0.5; z-index: 99; position: absolute; top: ' + (top - tag_height) + 'px; padding: 2px; left: ' +
		(left + ball_width) + 'px; color: #f0f0f0;">' + user_id + '</div>';

	return left_ptr_obj + tag_obj +
		'<div style="position: absolute; top: ' + (top - extra_height) + 'px; left: ' + left + 'px; color: ' +
		colorStyle + '; height: ' + (height + extra_height) + 'px; border-left: ' + width + 'px solid ' +
		colorStyle + '; ">' +
			'<div style="position: relative; height: ' + ball_width + 'px; width: ' +
			ball_width + 'px; border-radius: ' + (ball_width/2) + 'px; top: -' + (ball_width) +
			'px; left: -' + (ball_width/2 + width/2) + 'px; background-color: ' + colorStyle + '"></div>' +
		'</div>';
};

/*--------------------------------------------------------------------------------------------------
                                 Leaps Editor Bootstrapping
--------------------------------------------------------------------------------------------------*/

var last_document_joined = "";

var join_new_document = function(document_id) {
	if ( leaps_client !== null ) {
		leaps_client.close();
		leaps_client = null;
	}

	if ( ace_editor !== null ) {
		var oldDiv = ace_editor.container;
		var newDiv = oldDiv.cloneNode(false);

		ace_editor.destroy();
		ace_editor = null;

		oldDiv.parentNode.replaceChild(newDiv, oldDiv);
	}

	users = {};
	refresh_users_list();

	ace_editor = ace.edit("editor");
	configure_ace_editor();

	var filetype = "asciidoc";
	try {
		var ext = document_id.substr(document_id.lastIndexOf(".") + 1);
		if ( typeof langs[ext] === 'string' ) {
			filetype = langs[ext];
		}
	} catch (e) {}

	ace_editor.getSession().setMode("ace/mode/" + filetype);

	leaps_client = new leap_client();
	leaps_client.bind_ace_editor(ace_editor);

	leaps_client.on("error", function(err) {
		if ( leaps_client !== null ) {
			console.error(err);
			system_message("Connection to document closed, document is now READ ONLY", "red");
			leaps_client.close();
			leaps_client = null;
		}
	});

	leaps_client.on("disconnect", function(err) {
		if ( leaps_client !== null ) {
			last_document_joined = "";
			system_message(document_id + " closed", "red");
		}
	});

	leaps_client.on("connect", function() {
		leaps_client.join_document(username, "", document_id);
	});

	leaps_client.on("document", function() {
		last_document_joined = document_id;
		system_message("Opened document " + document_id, "ash");
	});

	leaps_client.on("user", function(user_update) {

		if ( 'string' === typeof user_update.message.content ) {
			chat_message(user_update.client.session_id, user_update.client.user_id, user_update.message.content);
		}

		var refresh_user_list = !users.hasOwnProperty(user_update.client.session_id);
		users[user_update.client.session_id] = user_update.client.user_id;

		if ( typeof user_update.message.active === 'boolean' && !user_update.message.active ) {
			refresh_user_list = true;
			delete users[user_update.client.session_id]
		}

		if ( refresh_user_list ) {
			refresh_users_list();
		}
	});

	leaps_client.ACE_set_cursor_handler(ACE_cursor_handler, ACE_cursor_clear_handler);

	var protocol = window.location.protocol === "http:" ? "ws:" : "wss:";
	leaps_client.connect(protocol + "//" + window.location.host + window.location.pathname + "leaps/ws");
};

var refresh_document = function() {
	if ( last_document_joined.length > 0 ) {
		system_message("Rejoining document " + last_document_joined, "ash");
		join_new_document(last_document_joined);
	}
};

/*--------------------------------------------------------------------------------------------------
                                    File Path Acquire and Listing
--------------------------------------------------------------------------------------------------*/

/* Creates a manager for the file path list.
 *
 * paths_ele   - the element to append list items to
 * opened_file - string of the current opened document (empty string if not applicable)
 * action      - callback for when a file path is clicked (function(path))
 */
var file_list = function(paths_ele, opened_file, action) {
	// Reset the element.
	paths_ele.innerHTML = "";

	this.file_item_class = "file-path";
	this.dir_item_class = "dir-path";

	this.element = paths_ele;
	this.files_obj = {};
	this.dirs_collapsed = [];
	this.file_opened = opened_file;

	this.open_action = action;

	try {
		if ( Cookies.get("collapsed-dirs") ) {
			this.dirs_collapsed = JSON.parse(Cookies.get("collapsed-dirs"));
		}
	} catch (e) {
		console.error("collapsed-dirs parse error", e);
	}
};

file_list.prototype.get_selected_li = function() {
	var li_eles = this.element.getElementsByTagName('li');
	for ( var i = 0, l = li_eles.length; i < l; i++ ) {
		if ( li_eles[i].className === this.file_item_class + ' selected' ) {
			return li_eles[i];
		}
	}
	return null;
};

file_list.prototype.create_path_click = function(ele, id) {
	var _this = this;

	return function() {
		if ( ele.className === _this.file_item_class + ' selected' ) {
			// Nothing
		} else {
			var current_ele = _this.get_selected_li();
			if ( current_ele !== null ) {
				current_ele.className = _this.file_item_class;
			}
			ele.className = _this.file_item_class + ' selected';
			window.location.hash = "path:" + id;
			_this.file_opened = id;
			_this.open_action(id);
		}
	};
};

file_list.prototype.create_dir_ele = function(id, name) {
	var _this = this;

	var ele = document.createElement("li");
	ele.className = _this.dir_item_class;

	for ( var i = 0, l = _this.dirs_collapsed.length; i < l; i++ ) {
		if ( _this.dirs_collapsed[i] === id ) {
			ele.className = _this.dir_item_class + ' collapsed';
			break;
		}
	}

	var span = document.createElement("div");
	span.className = "directory-name";

	span.appendChild(document.createTextNode(name));
	ele.appendChild(span);

	ele.id = id;
	span.onclick = function() {
		if ( ele.className === _this.dir_item_class + ' collapsed' ) {
			// Uncollapse
			ele.className = _this.dir_item_class;
			for ( var i = 0, l = _this.dirs_collapsed.length; i < l; i++ ) {
				if ( _this.dirs_collapsed[i] === id ) {
					_this.dirs_collapsed.splice(i, 1);
					set_cookie_option("collapsed-dirs", JSON.stringify(_this.dirs_collapsed));
					return;
				}
			}
		} else {
			// Collapse
			ele.className = _this.dir_item_class + ' collapsed';
			_this.dirs_collapsed.push(id);
			set_cookie_option("collapsed-dirs", JSON.stringify(_this.dirs_collapsed));
		}
	};
	return ele;
};

file_list.prototype.refresh = function() {
	var _this = this;

	AJAX_REQUEST(window.location.pathname + "files", function(data) {
		try {
			var paths_list = JSON.parse(data);
			_this.update_paths(paths_list.paths, paths_list.users);
		} catch (e) {
			console.error("paths parse error", e);
		}
		setTimeout(function() { _this.refresh() }, 5000);
	}, function(code, message) {
		console.error("get_paths error", code, message);
		setTimeout(function() { _this.refresh() }, 1000);
	});
};

file_list.prototype.draw_path_object = function(path_object, users_map, parent, path) {
	if ( "object" !== typeof path_object ) {
		console.error("paths object wrong type", typeof paths_object);
		return;
	}
	for ( var prop in path_object ) {
		if ( !path_object.hasOwnProperty(prop) ) {
			continue
		}

		var tmpPath = path + "/" + prop;
		if ( "object" === typeof path_object[prop] ) {
			var li = this.create_dir_ele(tmpPath, prop);

			var list = document.createElement("ul");
			list.className = "narrow-list";
			li.appendChild(list);

			this.draw_path_object(path_object[prop], users_map, list, tmpPath);
			parent.appendChild(li);
		} else if ( "string" === typeof path_object[prop] ) {
			var text = document.createTextNode(prop);
			var li = document.createElement("li");

			li.id = path_object[prop];
			li.onclick = this.create_path_click(li, li.id);

			var path_users_list = users_map[li.id];
			if ( "object" === typeof path_users_list ) {
				text = document.createTextNode(prop + " (" + path_users_list.length + ")");
			}

			if ( this.file_opened === li.id ) {
				li.className = this.file_item_class + ' selected';
			} else {
				li.className = this.file_item_class;
			}
			li.appendChild(text);

			parent.appendChild(li);
		} else {
			console.error("path object wrong type", typeof path_object[prop]);
		}
	}
};

file_list.prototype.update_paths = function(paths_list, users_map) {
	var i = 0, l = 0, j = 0, k = 0;

	if ( typeof paths_list !== 'object' ) {
		console.error("paths list wrong type", typeof paths_list);
		return;
	}
	if ( typeof users_map !== 'object' ) {
		console.error("users map wrong type", typeof users_map);
		return;
	}

	for ( i = 0, l = paths_list.length; i < l; i++ ) {
		var split_path = paths_list[i].split('/');
		var ptr = this.files_obj;
		for ( j = 0, k = split_path.length - 1; j < k; j++ ) {
			if ( 'object' !== typeof ptr[split_path[j]] ) {
				ptr[split_path[j]] = {};
			}
			ptr = ptr[split_path[j]];
		}
		ptr[split_path[split_path.length - 1]] = paths_list[i];
	}

	this.element.innerHTML = "";
	this.draw_path_object(this.files_obj, users_map, this.element, "");
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

/*--------------------------------------------------------------------------------------------------
                                      Chat UI Helpers
--------------------------------------------------------------------------------------------------*/
// Use to alert users when new messages appear
var flash_chat_window = function() {
	var info_window = document.getElementById("info-window");
	info_window.style.boxShadow = "0px 0px 0px 5px #ffffff";

	setTimeout(function() {
		info_window.style.boxShadow = "0px 0px 0px 0px #ffffff";
	}, 300);
};

var chat_message = function(user_id, username, message) {
	var container = document.getElementById("info-window");
	var messages = document.getElementById("info-messages");
	var div = document.createElement("div");

	var ts_span = document.createElement('span');
	var name_span = document.createElement('span');
	var text_span = document.createElement('span');

	if ( 'string' === typeof user_id ) {
		name_span.style.backgroundColor = user_id_to_color(user_id);
		name_span.style.color = "#f0f0f0";
	}

	div.className = "ash";

	name_span.style.fontWeight = "700";
	name_span.style.paddingLeft = "4px";
	name_span.style.paddingRight = "4px";

	var date_txt = document.createTextNode((new Date()).toTimeString().substr(0, 8) + " ");
	var user_txt = document.createTextNode(username);
	var msg_txt = document.createTextNode(" " + message);

	ts_span.appendChild(date_txt);
	div.appendChild(ts_span);

	name_span.appendChild(user_txt);
	div.appendChild(name_span);

	text_span.appendChild(msg_txt);
	div.appendChild(text_span);

	messages.appendChild(div);
	container.scrollTop = container.scrollHeight;

	flash_chat_window();
};

var system_message = function(text, style) {
	var container = document.getElementById("info-window");
	var messages = document.getElementById("info-messages");
	var div = document.createElement("div");
	if ( typeof style === 'string' ) {
		div.className = style + " bold";
	}
	var textNode = document.createTextNode((new Date()).toTimeString().substr(0, 8) + " " + text);

	div.appendChild(textNode);
	messages.appendChild(div);
	container.scrollTop = container.scrollHeight;

	flash_chat_window();
};

/*--------------------------------------------------------------------------------------------------
                                    Users List UI Helpers
--------------------------------------------------------------------------------------------------*/
var refresh_users_list = function() {
	var styles = ["ash","light-grey"];
	var style_index = 0;

	var users_element = document.getElementById("users-list");
	users_element.innerHTML = "";

	var self_element = document.createElement("div");
	var self_text_ele = document.createTextNode(username);

	self_element.className = styles[((style_index++)%styles.length)];

	self_element.appendChild(self_text_ele);
	users_element.appendChild(self_element);

	for (var user in users) {
		if (users.hasOwnProperty(user)) {
			var user_element = document.createElement("div");
			var user_text_ele = document.createTextNode(users[user]);

			user_element.className = styles[((style_index++)%styles.length)];
			user_element.style.color = user_id_to_color(user);

			user_element.appendChild(user_text_ele);
			users_element.appendChild(user_element);
		}
	}
};

/*--------------------------------------------------------------------------------------------------
                                    Set Cookies Helper
--------------------------------------------------------------------------------------------------*/
var set_cookie_option = function(key, value) {
	Cookies.set(key, value, { path: '' });
};

window.onload = function() {

/*--------------------------------------------------------------------------------------------------
                                    Messages Clear Button
--------------------------------------------------------------------------------------------------*/
	var clear_button = document.getElementById("clear-button") || {};
	clear_button.onclick = function() {
		var messages = document.getElementById("info-messages");
		messages.innerHTML = "";
	};

/*--------------------------------------------------------------------------------------------------
                                       Username Input
--------------------------------------------------------------------------------------------------*/
	var username_bar = document.getElementById("username-bar");
	if ( Cookies.get("username") ) {
		username_bar.value = Cookies.get("username");
	}
	username = username_bar.value || "anon";
	username_bar.onkeypress = function(e) {
		if ( typeof e !== 'object' ) {
			e = window.event;
		}
		var keyCode = e.keyCode || e.which;
		if ( keyCode == '13' ) {
			username = username_bar.value || "anon";
			set_cookie_option("username", username_bar.value);
			refresh_users_list();
			refresh_document();
		}
	};
	refresh_users_list();

/*--------------------------------------------------------------------------------------------------
                                     Use Tabs Checkbox
--------------------------------------------------------------------------------------------------*/
	var input_use_tabs = document.getElementById("input-use-tabs");
	if ( Cookies.get("useTabs") ) {
		useTabs = Cookies.get("useTabs") === "true";
	}
	input_use_tabs.checked = useTabs;
	input_use_tabs.onchange = function() {
		useTabs = input_use_tabs.checked;

		set_cookie_option("useTabs", useTabs ? "true" : "false");
		if ( ace_editor !== null ) {
			ace_editor.getSession().setUseSoftTabs(!useTabs);
		}
	};

/*--------------------------------------------------------------------------------------------------
                                     Wrap Lines Checkbox
--------------------------------------------------------------------------------------------------*/
	var input_wrap_lines = document.getElementById("input-wrap-lines");
	if ( Cookies.get("wrapLines") ) {
		wrapLines = Cookies.get("wrapLines") === "true";
	}
	input_wrap_lines.checked = wrapLines;
	input_wrap_lines.onchange = function() {
		wrapLines = input_wrap_lines.checked;

		set_cookie_option("wrapLines", wrapLines ? "true" : "false");
		if ( ace_editor !== null ) {
			ace_editor.getSession().setUseWrapMode(wrapLines);
		}
	};

/*--------------------------------------------------------------------------------------------------
                                  Key Mapping Drop Down Menu
--------------------------------------------------------------------------------------------------*/
	var input_select = document.getElementById("input-select");
	for ( var keymap in keymaps ) {
		if ( keymaps.hasOwnProperty(keymap) ) {
			input_select.innerHTML += '<option value="' + keymap + '">' + keymaps[keymap] + "</option>";
		}
	}
	if ( Cookies.get("input") ) {
		binding = Cookies.get("input");
	}
	input_select.value = binding;
	input_select.onchange = function() {
		binding = input_select.value;
		set_cookie_option("input", binding);

		if ( ace_editor !== null ) {
			var map = "";
			if ( binding !== "none" ) {
				map = "ace/keyboard/" + binding;
			}
			ace_editor.setKeyboardHandler(map);
		}
	};

/*--------------------------------------------------------------------------------------------------
                                        Theme Drop Down Menu
--------------------------------------------------------------------------------------------------*/
	var theme_select = document.getElementById("theme-select");
	for ( var prop in themes ) {
		if ( themes.hasOwnProperty(prop) ) {
			theme_select.innerHTML += '<option value="' + prop + '">' + themes[prop] + "</option>";
		}
	}
	if ( Cookies.get("theme") ) {
		theme = Cookies.get("theme");
	}
	theme_select.value = theme;
	theme_select.onchange = function() {
		theme = theme_select.value;
		set_cookie_option("theme", theme);

		if ( ace_editor !== null ) {
			ace_editor.setTheme("ace/theme/" + theme);
		}
	};

/*--------------------------------------------------------------------------------------------------
                                           Chat Bar
--------------------------------------------------------------------------------------------------*/
	var chat_bar = document.getElementById("chat-bar");
	chat_bar.onkeypress = function(e) {
		if ( typeof e !== 'object' ) {
			e = window.event;
		}
		var keyCode = e.keyCode || e.which;
		if ( keyCode == '13' && chat_bar.value.length > 0 ) {
			if ( leaps_client !== null ) {
				leaps_client.send_message(chat_bar.value);
				chat_message(null, username, chat_bar.value);
				chat_bar.value = "";
				return false;
			} else {
				system_message(
					"You must open a document in order to send messages, " +
					"they will be readable by other users editing that document", "red"
				);
				return true;
			}
		}
	};

	var info_window = document.getElementById("info-window");
	info_window.onclick = function() {
		chat_bar.focus();
	};

	var opened_path = "";
	// You can link directly to a filepath with <URL>#path:/this/is/the/path.go
	if ( window.location.hash.length > 0 &&
			window.location.hash.substr(1, 5) === "path:" ) {
		opened_path = window.location.hash.substr(6);
		join_new_document(opened_path);
	}

	var file_list_obj = new file_list(document.getElementById("file-list"), opened_path, join_new_document);
	file_list_obj.refresh();
};

})();
