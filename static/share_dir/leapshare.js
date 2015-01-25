(function() {
"use strict";

/*\
|*|
|*|  :: cookies.js ::
|*|
|*|  A complete cookies reader/writer framework with full unicode support.
|*|
|*|  Revision #1 - September 4, 2014
|*|
|*|  https://developer.mozilla.org/en-US/docs/Web/API/document.cookie
|*|  https://developer.mozilla.org/User:fusionchess
|*|
|*|  This framework is released under the GNU Public License, version 3 or later.
|*|  http://www.gnu.org/licenses/gpl-3.0-standalone.html
|*|
|*|  Syntaxes:
|*|
|*|  * docCookies.setItem(name, value[, end[, path[, domain[, secure]]]])
|*|  * docCookies.getItem(name)
|*|  * docCookies.removeItem(name[, path[, domain]])
|*|  * docCookies.hasItem(name)
|*|  * docCookies.keys()
|*|
\*/

var docCookies = {
	getItem: function (sKey) {
		if (!sKey) { return null; }
		return decodeURIComponent(document.cookie.replace(new RegExp("(?:(?:^|.*;)\\s*" + encodeURIComponent(sKey).replace(/[\-\.\+\*]/g, "\\$&") + "\\s*\\=\\s*([^;]*).*$)|^.*$"), "$1")) || null;
	},
	setItem: function (sKey, sValue, vEnd, sPath, sDomain, bSecure) {
		if (!sKey || /^(?:expires|max\-age|path|domain|secure)$/i.test(sKey)) { return false; }
		var sExpires = "";
		if (vEnd) {
			switch (vEnd.constructor) {
				case Number:
					sExpires = vEnd === Infinity ? "; expires=Fri, 31 Dec 9999 23:59:59 GMT" : "; max-age=" + vEnd;
					break;
				case String:
					sExpires = "; expires=" + vEnd;
					break;
				case Date:
					sExpires = "; expires=" + vEnd.toUTCString();
					break;
				}
		}
		document.cookie = encodeURIComponent(sKey) + "=" + encodeURIComponent(sValue) + sExpires + (sDomain ? "; domain=" + sDomain : "") + (sPath ? "; path=" + sPath : "") + (bSecure ? "; secure" : "");
	return true;
	},
	removeItem: function (sKey, sPath, sDomain) {
		if (!this.hasItem(sKey)) { return false; }
		document.cookie = encodeURIComponent(sKey) + "=; expires=Thu, 01 Jan 1970 00:00:00 GMT" + (sDomain ? "; domain=" + sDomain : "") + (sPath ? "; path=" + sPath : "");
		return true;
	},
	hasItem: function (sKey) {
		if (!sKey) { return false; }
		return (new RegExp("(?:^|;\\s*)" + encodeURIComponent(sKey).replace(/[\-\.\+\*]/g, "\\$&") + "\\s*\\=")).test(document.cookie);
	},
	keys: function () {
		var aKeys = document.cookie.replace(/((?:^|\s*;)[^\=]+)(?=;|$)|^\s*|\s*(?:\=[^;]*)?(?:\1|$)/g, "").split(/\s*(?:\=[^;]*)?;\s*/);
		for (var nLen = aKeys.length, nIdx = 0; nIdx < nLen; nIdx++) { aKeys[nIdx] = decodeURIComponent(aKeys[nIdx]); }
		return aKeys;
	}
};

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

var keymaps = {
	none : "Standard",
	vim : "Vim",
	emacs : "Emacs"
};

var ace_editor = null;
var leaps_client = null;
var username = "anon";

var users = {};

var theme = "dawn";
var binding = "none";
var useTabs = true;
var wrapLines = true;

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

var oob_elements = [];

var ACE_cursor_clear_handler = function() {
	for ( var i = 0, l = oob_elements.length; i < l; i++ ) {
		document.body.removeChild(oob_elements[i]);
	}
	oob_elements = [];
};

var ACE_cursor_handler = function(user_id, lineHeight, top, left, row, column) {
	var id_hash = hash(user_id);
	if ( id_hash < 0 ) {
		id_hash = id_hash * -1;
	}

	var hue = ( id_hash % 10000 ) / 10000;
	var rgb = HSVtoRGB(hue, 1, 0.8);

	var colorStyle = "rgba(" + rgb.r + ", " + rgb.g + ", " + rgb.b + ", 1)";

	// Needs IE9
	var editor_bounds = ace_editor.container.getBoundingClientRect();

	var editor_width = ace_editor.getSession().getScreenWidth();
	var editor_height = ace_editor.getSession().getScreenLength();

	var user_tag = user_id;
	if ( 'string' === typeof users[user_id] ) {
		user_tag = users[user_id];
	}

	var triangle_height = 20;
	var triangle_opacity = 0.5;
	var ball_width = 8;

	var height = lineHeight;
	var extra_height = 6;
	var width = 2;

	var create_ptr_ele = function() {
		var top_ptr_ele = document.createElement('div');
		top_ptr_ele.style.opacity = 0.7 + '';
		top_ptr_ele.style.position = 'absolute';
		top_ptr_ele.style.width = '0';
		top_ptr_ele.style.height = '0';
		top_ptr_ele.style.zIndex = '99';

		return top_ptr_ele;
	}
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
	var left_ptr_obj = '<div style="' +
		'position: absolute; ' +
		'width: 0; height: 0; z-index: 99; ' +
		'top: ' + top + 'px; ' +
		'left: 0; border-top: ' + (triangle_height/2) + 'px solid transparent; ' +
		'border-left: ' + (triangle_height/3) + 'px solid ' + colorStyle + '; ' +
		'border-bottom: ' + (triangle_height/2) + 'px solid transparent; ' +
		'opacity: 0.7; ' +
		'"></div>';

	var tag_obj = '<div class="name-tag" style="z-index: 99; color:#f0f0f0; ' +
		'position: absolute; top: ' + (top - extra_height - ball_width) + 'px; left: ' + left + 'px; ' +
		'background-color:' + colorStyle + ';">' + user_tag + '</div>';

	return left_ptr_obj + tag_obj +
		'<div style="position: absolute; top: ' + (top - extra_height) + 'px; left: ' + left + 'px; color: ' +
		colorStyle + '; height: ' + (height + extra_height) + 'px; border-left: ' + width + 'px solid ' +
		colorStyle + '; ">' +
			'<div style="position: relative; height: ' + ball_width + 'px; width: ' +
			ball_width + 'px; border-radius: ' + (ball_width/2) + 'px; top: -' + (ball_width) +
			'px; left: -' + (ball_width/2 + width/2) + 'px; background-color: ' + colorStyle + '"></div>' +
		'</div>';
};

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
			system_message(document_id + " closed", "red");
		}
	});

	leaps_client.on("connect", function() {
		leaps_client.join_document(document_id);
	});

	leaps_client.on("document", function() {
		system_message("Opened document " + document_id, "blue");
	});

	leaps_client.on("user", function(user_update) {
		var metadata = user_update.message;
		if ( 'string' === typeof metadata ) {
			var data = JSON.parse(metadata);
			if ( 'string' === typeof data.text ) {
				chat_message(user_update.user_id, data.username, data.text);
			}
			if ( 'string' === typeof data.username ) {
				users[user_update.user_id] = data.username;
			}
		}
	});

	leaps_client.ACE_set_cursor_handler(ACE_cursor_handler, ACE_cursor_clear_handler);

	leaps_client.connect("ws://" + window.location.host + "/socket");
};

var fileItemClass = "file-path";

var get_selected_li = function() {
	var li_eles = document.getElementsByTagName('li');
	for ( var i = 0, l = li_eles.length; i < l; i++ ) {
		if ( li_eles[i].className === fileItemClass + ' selected' ) {
			return li_eles[i];
		}
	}
	return null;
};

var create_path_click = function(ele, id) {
	return function() {
		if ( ele.className === fileItemClass + ' selected' ) {
			// Nothing
		} else {
			var current_ele = get_selected_li();
			if ( current_ele !== null ) {
				current_ele.className = fileItemClass;
			}
			ele.className = fileItemClass + ' selected';
			join_new_document(id);
		}
	};
};

var draw_path_object = function(path_object, parent, selected_id) {
	if ( "object" === typeof path_object ) {
		for ( var prop in path_object ) {
			if ( path_object.hasOwnProperty(prop) ) {
				var li = document.createElement("li");
				var text = document.createTextNode(prop);

				if ( "object" === typeof path_object[prop] ) {
					var span = document.createElement("span");
					var list = document.createElement("ul");

					list.className = "narrow-list";
					span.className = "directory-name";

					span.appendChild(text);
					li.appendChild(span);
					li.appendChild(list);

					draw_path_object(path_object[prop], list, selected_id);
					parent.appendChild(li);
				} else if ( "string" === typeof path_object[prop] ) {
					li.id = path_object[prop];
					li.onclick = create_path_click(li, li.id);

					if ( selected_id === li.id ) {
						li.className = fileItemClass + ' selected';
					} else {
						li.className = fileItemClass;
					}
					li.appendChild(text);

					parent.appendChild(li);
				} else {
					console.error("path object wrong type", typeof path_object[prop]);
				}
			}
		}
	}
};

var show_paths = function(paths_list) {
	var i = 0, l = 0, j = 0, k = 0;

	if ( typeof paths_list !== 'object' ) {
		console.error("paths list wrong type", typeof paths_list);
		return;
	}

	var paths_hierarchy = {};
	for ( i = 0, l = paths_list.length; i < l; i++ ) {
		var split_path = paths_list[i].split('/');
		var ptr = paths_hierarchy;
		for ( j = 0, k = split_path.length - 1; j < k; j++ ) {
			if ( 'object' !== typeof ptr[split_path[j]] ) {
				ptr[split_path[j]] = {};
			}
			ptr = ptr[split_path[j]];
		}
		ptr[split_path[split_path.length - 1]] = paths_list[i];
	}

	var selected_path = "";
	var selected_ele = get_selected_li();
	if ( selected_ele !== null ) {
		selected_path = selected_ele.id;
	}

	var paths_ele = document.getElementById("file-list");
	paths_ele.innerHTML = "";

	draw_path_object(paths_hierarchy, paths_ele, selected_path);
};

var AJAX_GET = function(path, onsuccess, onerror) {
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

	xmlhttp.open("GET", path, true);
	xmlhttp.send();
};

var get_paths = function() {
	AJAX_GET("/files", function(data) {
		try {
			var paths_list = JSON.parse(data);
			show_paths(paths_list.paths);
		} catch (e) {
			console.error("paths parse error", e);
		}
	}, function(code, message) {
		console.error("get_paths error", code, message);
	});
};

var chat_message = function(user_id, username, message) {
	var container = document.getElementById("info-window");
	var messages = document.getElementById("info-messages");
	var div = document.createElement("div");

	var ts_span = document.createElement('span');
	var name_span = document.createElement('span');
	var text_span = document.createElement('span');

	if ( 'string' === typeof user_id ) {
		var id_hash = hash(user_id);
		if ( id_hash < 0 ) {
			id_hash = id_hash * -1;
		}

		var hue = ( id_hash % 10000 ) / 10000;
		var rgb = HSVtoRGB(hue, 1, 0.8);

		name_span.style.backgroundColor = "rgba(" + rgb.r + ", " + rgb.g + ", " + rgb.b + ", 1)";
		name_span.style.color = "#f0f0f0";
	}

	div.className = "white";

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
};

var system_message = function(text, style) {
	var container = document.getElementById("info-window");
	var messages = document.getElementById("info-messages");
	var div = document.createElement("div");
	if ( typeof style === 'string' ) {
		div.className = style;
	}
	var textNode = document.createTextNode((new Date()).toTimeString().substr(0, 8) + " " + text);

	div.appendChild(textNode);
	messages.appendChild(div);
	container.scrollTop = container.scrollHeight;
};

var set_cookie_option = function(key, value) {
	var expiresDate = new Date();
	expiresDate.setDate(expiresDate.getDate() + 30);

	docCookies.setItem(key, value, expiresDate);
};

window.onload = function() {
	get_paths();

	var refresh_button = document.getElementById("refresh-button");
	refresh_button.onclick = function() {
		get_paths();
	};

	var clear_button = document.getElementById("clear-button");
	clear_button.onclick = function() {
		var messages = document.getElementById("info-messages");
		messages.innerHTML = "";
	};

	// Username option
	var username_bar = document.getElementById("username-bar");
	if ( docCookies.hasItem("username") ) {
		username_bar.value = docCookies.getItem("username");
	}
	username = username_bar.value || "anon";
	username_bar.onkeyup = function() {
		username = username_bar.value || "anon";
		set_cookie_option("username", username_bar.value);
	};

	// Use tabs option
	var input_use_tabs = document.getElementById("input-use-tabs");
	if ( docCookies.hasItem("useTabs") ) {
		useTabs = docCookies.getItem("useTabs") === "true";
	}
	input_use_tabs.checked = useTabs;
	input_use_tabs.onchange = function() {
		useTabs = input_use_tabs.checked;

		set_cookie_option("useTabs", useTabs ? "true" : "false");
		if ( ace_editor !== null ) {
			ace_editor.getSession().setUseSoftTabs(!useTabs);
		}
	};

	// Wrap lines option
	var input_wrap_lines = document.getElementById("input-wrap-lines");
	if ( docCookies.hasItem("wrapLines") ) {
		wrapLines = docCookies.getItem("wrapLines") === "true";
	}
	input_wrap_lines.checked = wrapLines;
	input_wrap_lines.onchange = function() {
		wrapLines = input_wrap_lines.checked;

		set_cookie_option("wrapLines", wrapLines ? "true" : "false");
		if ( ace_editor !== null ) {
			ace_editor.getSession().setUseWrapMode(wrapLines);
		}
	};

	// Key map option
	var input_select = document.getElementById("input-select");
	for ( var keymap in keymaps ) {
		if ( keymaps.hasOwnProperty(keymap) ) {
			input_select.innerHTML += '<option value="' + keymap + '">' + keymaps[keymap] + "</option>";
		}
	}
	if ( docCookies.hasItem("input") ) {
		binding = docCookies.getItem("input");
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

	// Theme option
	var theme_select = document.getElementById("theme-select");
	for ( var prop in themes ) {
		if ( themes.hasOwnProperty(prop) ) {
			theme_select.innerHTML += '<option value="' + prop + '">' + themes[prop] + "</option>";
		}
	}
	if ( docCookies.hasItem("theme") ) {
		theme = docCookies.getItem("theme");
	}
	theme_select.value = theme;
	theme_select.onchange = function() {
		theme = theme_select.value;
		set_cookie_option("theme", theme);

		if ( ace_editor !== null ) {
			ace_editor.setTheme("ace/theme/" + theme);
		}
	};

	// Chat bar
	var chat_bar = document.getElementById("chat-bar");
	chat_bar.onkeypress = function(e) {
		if ( typeof e !== 'object' ) {
			e = window.event;
		}
		var keyCode = e.keyCode || e.which;
		if ( keyCode == '13' && chat_bar.value.length > 0 ) {
			if ( leaps_client !== null ) {
				leaps_client.send_message(JSON.stringify({
					username: username,
					text: chat_bar.value
				}));
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

	setInterval(function() {
		if ( leaps_client !== null ) {
			leaps_client.send_message(JSON.stringify({
				username: username
			}));
		}
	}, 1000);
};

})();
