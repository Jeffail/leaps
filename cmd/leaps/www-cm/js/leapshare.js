(function() {
"use strict";

/*------------------------------------------------------------------------------
                        List of CodeMirror Key Mappings
------------------------------------------------------------------------------*/

var cm_keymaps = {
	None : "default",
	Vim : "vim",
	Emacs : "emacs",
	Sublime : "sublime"
};

var cm_themes = {
	Light: "default",
	Dark: "zenburn"
};

/*------------------------------------------------------------------------------
                              Global Variables
------------------------------------------------------------------------------*/

var cm_editor = null;
var leaps_client = null;
var username = Cookies.get("username") || "anon";

var users = {};
var file_paths = {
	opened: '',
	root: true,
	name: "Files",
	path: "Files",
	children: []
};
var collapsed_dirs = {};

var messages_obj = {
	messages: []
};

// Configuration options
var config = {
	theme: "Dark",
	binding: "None",
	use_tabs: true,
	indent_unit: 4,
	wrap_lines: false,
	hide_numbers: false
};

// Load config from cookies
(function() {
	var conf_str = Cookies.get("config");
	if ( conf_str !== undefined && conf_str.length > 0 ) {
		try {
			config = JSON.parse(conf_str);
		} catch(e) {
			console.error(err);
		}
	}
})();

function save_config() {
	Cookies.set("config", JSON.stringify(config), { path: '' });
}

/*------------------------------------------------------------------------------
                        Leaps Editor Bootstrapping
------------------------------------------------------------------------------*/

function replace_classname(old_class, new_class) {
	var old_elements = document.getElementsByClassName(old_class);
	var i = old_elements.length;
	while ( i-- ) {
		old_elements[i].className = old_elements[i].className.replace(old_class, new_class);
	}
}

function configure_codemirror() {
	if ( config.theme === 'Light' ) {
		replace_classname("dark", "light");
	} else {
		replace_classname("light", "dark");
	}
	if ( cm_editor !== null ) {
		cm_editor.setOption("theme", cm_themes[config.theme]);
		cm_editor.setOption("keyMap", cm_keymaps[config.binding]);
		cm_editor.setOption("indentWithTabs", config.use_tabs);
		cm_editor.setOption("indentUnit", config.indent_unit);
		cm_editor.setOption("lineWrapping", config.wrap_lines);
		cm_editor.setOption("lineNumbers", !config.hide_numbers);
	}
}

function join_new_document(document_id) {
	if ( leaps_client !== null ) {
		leaps_client.close();
		leaps_client = null;
	}

	if ( cm_editor !== null ) {
		cm_editor.getWrapperElement().parentNode.removeChild(cm_editor.getWrapperElement());
		cm_editor = null;
	}

	// Clear existing users list
	for (var key in users) {
		if (users.hasOwnProperty(key)) {
			// Must use Vue.delete otherwise update is not triggered
			Vue.delete(users, key);
		}
	}

	var default_options = CodeMirror.defaults;
	default_options.readOnly = true;
	default_options.viewPortMargin = "Infinity";

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
		show_sys_message("Closed " + document_id);
		if ( cm_editor !== null ) {
			cm_editor.options.readOnly = true;
		}
	});

	leaps_client.on("connect", function() {
		leaps_client.join_document(username, "", document_id);
	});

	leaps_client.on("document", function() {
		file_paths.opened = document_id;
		cm_editor.options.readOnly = false;
		show_sys_message("Opened " + document_id);

		// Set the hash of our URL to the path
		window.location.hash = "path:" + document_id;
	});

	leaps_client.on("user", function(user_update) {
		if ( 'string' === typeof user_update.message.content ) {
			show_user_message(user_update.client.user_id, user_update.client.session_id, user_update.message.content);
		}

		if ( !users.hasOwnProperty(user_update.client.session_id) && user_update.message.active ) {
			show_sys_message("User " + user_update.client.user_id + " joined");
		}
		Vue.set(users, user_update.client.session_id, {
			name: user_update.client.user_id,
			position: user_update.message.position
		});
		if ( typeof user_update.message.active === 'boolean' && !user_update.message.active ) {
			if ( users.hasOwnProperty(user_update.client.session_id) ) {
				show_sys_message("User " + user_update.client.user_id + " left");
			}
			// Must use Vue.delete otherwise update is not triggered
			Vue.delete(users, user_update.client.session_id);
		}
	});

	var protocol = window.location.protocol === "http:" ? "ws:" : "wss:";
	leaps_client.connect(protocol + "//" + window.location.host + window.location.pathname + "leaps/ws");
}

function cursor_to_position(position) {
	if ( cm_editor !== null ) {
		var pos = leap_client.pos_from_u_index(cm_editor.getDoc(), position);
		cm_editor.setCursor(pos);
		cm_editor.focus();
	}
}

/*------------------------------------------------------------------------------
                                  Messages
------------------------------------------------------------------------------*/

function clip_messages() {
	if ( messages_obj.messages.length > 200 ) {
		messages_obj.messages = messages_obj.messages.slice(-200);
	}
	setTimeout(function() {
		// Yield for the Vue renderer before scrolling.
		var messages_ele = document.getElementById("message-list");
		messages_ele.scrollTop = messages_ele.scrollHeight;
	}, 1);
}

function show_user_message(username, session, content) {
	var now = new Date();
	var name_style = {};
	if ( session !== null ) {
		name_style.backgroundColor = leap_client.session_id_to_colour(session);
		name_style.color = "#fcfcfc";
	}
	messages_obj.messages.push({
		timestamp: now.toLocaleTimeString(),
		name: username,
		name_style: name_style,
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

function AJAX_REQUEST(path, onsuccess, onerror, data) {
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
                           Input field bindings
------------------------------------------------------------------------------*/

function text_input(element, on_change) {
	if ( typeof element === 'string' ) {
		element = document.getElementById(element);
	}
	element.onkeypress = function(e) {
		if ( typeof e !== 'object' ) {
			e = window.event;
		}
		var keyCode = e.keyCode || e.which;
		if ( keyCode == '13' ) {
			on_change(element, element.value);
		}
	};
}

function init_input_fields() {
	var username_bar = document.getElementById("username-bar");
	username_bar.value = username;
	text_input(username_bar, function(ele, content) {
		if ( content === username ) {
			return;
		}
		if ( content.length === 0 ) {
			content = "anon";
			username_bar.value = content;
		}
		username = content;
		Cookies.set("username", username, { path: '' });
		if ( file_paths.opened.length > 0 ) {
			join_new_document(file_paths.opened);
		}
	});

	var chat_bar = document.getElementById("chat-bar");
	text_input(chat_bar, function(ele, content) {
		if ( content.length > 0 ) {
			if ( leaps_client !== null ) {
				leaps_client.send_message(content);
				show_user_message(username, null, content);
				ele.value = "";
			} else {
				show_err_message(
					"You must open a document in order to send messages, " +
					"they will be readable by other users editing that document"
				);
			}
		}
	});
	var message_list = document.getElementById("message-list");
	message_list.onclick = function() {
		chat_bar.focus();
	};

	var settings_window = document.getElementById("settings");
	document.getElementById("settings-open-btn").onclick = function() {
		settings_window.style.display = "";
	};
	document.getElementById("settings-close-btn").onclick = function() {
		settings_window.style.display = "none";
	};
	settings_window.onclick = function() {
		settings_window.style.display = "none";
	};

	var settings_inner_window = document.getElementById("settings-window");
	settings_inner_window.onclick = function(e) {
		if (!e) {
			e = window.event;
		}
		if (e.stopPropagation) {
			e.stopPropagation();
		}
		return true;
	};
}

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
			is_open: function() {
				if ( this.model.path.length <= file_paths.opened.length ) {
					return file_paths.opened.substring(0, this.model.path.length) === this.model.path;
				}
				return false;
			},
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

	(new Vue({ el: '#file-list', data: { file_data: file_paths } }));
	(new Vue({ el: '#message-list', data: messages_obj }));
	(new Vue({
		el: '#users-list',
		data: { users: users },
		methods: {
			style: function(id) {
				return {
					color: "#fcfcfc",
					backgroundColor: leap_client.session_id_to_colour(id)
				}
			},
			go_to: function(position) {
				cursor_to_position(position);
			}
		}
	}));
	(new Vue({
		el: '#settings',
		data: {
			themes: cm_themes,
			bindings: cm_keymaps,
			config: config
		},
		methods: {
			on_config_change: function() {
				save_config();
				configure_codemirror();
			}
		}
	}));

	init_input_fields();

	CodeMirror.modeURL = "mode/%N/%N.js";

	try {
		collapsed_dirs = JSON.parse(Cookies.get("collapsed_dirs"));
	} catch (e) {}
	get_paths();
	setInterval(get_paths, 1000);

	window.onhashchange = function() {
		// You can link directly to a filepath with <URL>#path:/this/is/the/path.go
		if ( window.location.hash.length > 0 &&
			window.location.hash.substr(1, 5) === "path:" ) {
			var new_path = window.location.hash.substr(6);
			if ( new_path !== file_paths.opened ) {
				join_new_document(window.location.hash.substr(6));
			}
		}
	};

	// Event isn't triggered on page load but we might want to check.
	window.onhashchange();
};

})();
