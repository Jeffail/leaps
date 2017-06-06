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

var last_document_joined = "";

function configure_codemirror() {
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

	// Set the hash of our URL to the path
	window.location.hash = "path:" + document_id;

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
		Vue.set(users, user_update.client.session_id, user_update.client.user_id);

		if ( typeof user_update.message.active === 'boolean' && !user_update.message.active ) {
			refresh_user_list = true;
			// Must use Vue.delete otherwise update is not triggered
			Vue.delete(users, user_update.client.session_id);
		}
	});

	var protocol = window.location.protocol === "http:" ? "ws:" : "wss:";
	leaps_client.connect(protocol + "//" + window.location.host + window.location.pathname + "leaps/ws");
}

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
		if ( last_document_joined.length > 0 ) {
			join_new_document(last_document_joined);
		}
	});

	// Set up chat bar
	text_input("chat-bar", function(ele, content) {
		if ( content.length > 0 ) {
			if ( leaps_client !== null ) {
				leaps_client.send_message(content);
				show_user_message(username, content);
				ele.value = "";
			} else {
				show_err_message(
					"You must open a document in order to send messages, " +
					"they will be readable by other users editing that document"
				);
			}
		}
	});

	var settings_window = document.getElementById("settings");
	document.getElementById("settings-open-btn").onclick = function() {
		settings_window.style.display = "";
	};
	document.getElementById("settings-close-btn").onclick = function() {
		settings_window.style.display = "none";
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
	(new Vue({ el: '#users-list', data: { users: users } }));
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

	CodeMirror.modeURL = "cm/mode/%N/%N.js";

	try {
		collapsed_dirs = JSON.parse(Cookies.get("collapsed_dirs"));
	} catch (e) {}
	get_paths();
	setInterval(get_paths, 1000);

	// You can link directly to a filepath with <URL>#path:/this/is/the/path.go
	if ( window.location.hash.length > 0 &&
		window.location.hash.substr(1, 5) === "path:" ) {
		join_new_document(window.location.hash.substr(6));
	}
};

})();
