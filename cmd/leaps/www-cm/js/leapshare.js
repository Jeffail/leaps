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
Cookies.set("username", username, { path: '', expires: 7 });

var users = {};
var file_paths = {
	next_path: '',
	opened: '',
	root: true,
	name: "Files",
	path: "Files",
	children: [],
	subscriptions: {}
};
var collapsed_dirs = {};

var messages_obj = {
	messages: []
};

var cmds_obj = {
	selected: 0,
	options: []
};

// Configuration options
var config = {
	theme: "Dark",
	binding: "None",
	use_tabs: true,
	indent_unit: 4,
	line_guide: 0,
	wrap_lines: false,
	show_space: true,
	auto_bracket: true,
	show_bracket: true,
	hide_numbers: false
};

// Load config from cookies
(function() {
	var conf_str = Cookies.get("config");
	if ( conf_str !== undefined && conf_str.length > 0 ) {
		try {
			var loaded_config = JSON.parse(conf_str);
			for ( var field in loaded_config ) {
				if ( loaded_config.hasOwnProperty(field) ) {
					config[field] = loaded_config[field];
				}
			}
		} catch(e) {
			console.error(err);
		}
	}
})();

function save_config() {
	Cookies.set("config", JSON.stringify(config), { path: '', expires: 7 });
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
		cm_editor.setOption("showTrailingSpace", config.show_space);
		cm_editor.setOption("autoCloseBrackets", config.auto_bracket);
		cm_editor.setOption("matchBrackets", config.show_bracket);
		if ( config.line_guide > 0 ) {
			cm_editor.setOption("rulers", [
				{color: "#777", column: config.line_guide, lineStyle: "dashed"}
			]);
		} else {
			cm_editor.setOption("rulers", []);
		}
	}
}

function restart_leaps() {
	if ( file_paths.opened.length > 0 ) {
		var reopen_path = file_paths.opened;
		leaps_client.on_next("disconnect", function() {
			join_document(reopen_path);
		});
	} else {
		leaps_client.on_next("disconnect", function() {
			init_leaps(function() {});
		});
	}
	leaps_client.close();
}

function init_leaps(after) {
	if ( cm_editor === null ) {
		var default_options = CodeMirror.defaults;
		default_options.readOnly = true;
		default_options.viewPortMargin = "Infinity";

		cm_editor = CodeMirror(document.getElementById("editor"), default_options);

		configure_codemirror();
	}

	if ( leaps_client !== null ) {
		after();
		return;
	}

	leaps_client = new leap_client();
	leaps_client.bind_codemirror(cm_editor);

	leaps_client.on("error", function(body) {
		if ( body.error.type === "ERR_SYNC" ) {
			show_err_message(body.error.message);
			if ( cm_editor !== null ) {
				cm_editor.options.readOnly = true;
			}
			if ( leaps_client !== null ) {
				console.error(body.error);
				leaps_client.close();
				leaps_client = null;
			}
		}
	});

	leaps_client.on("unsubscribe", function(body) {
		file_paths.opened = '';
		show_sys_message("Closed " + body.document.id);
		if ( cm_editor !== null ) {
			cm_editor.options.readOnly = true;
		}
		if ( file_paths.next_path.length > 0 ) {
			let err = leaps_client.subscribe(file_paths.next_path);
			if ( err ) {
				console.error(err);
			}
			file_paths.next_path = '';
		}
		calc_sub_counts();
	});

	leaps_client.on("connect", function() {
		show_sys_message("Connected");
		after();
	});

	leaps_client.on("disconnect", function() {
		file_paths.opened = '';
		show_sys_message("Lost connection");
		if ( cm_editor !== null ) {
			cm_editor.options.readOnly = true;
			cm_editor.getWrapperElement().parentNode.removeChild(cm_editor.getWrapperElement());
			cm_editor = null;
		}
		if ( leaps_client !== null ) {
			leaps_client.close();
			leaps_client = null;
		}
	});

	leaps_client.on("subscribe", function(body) {
		file_paths.opened = body.document.id;
		cm_editor.options.readOnly = false;
		show_sys_message("Opened " + body.document.id);

		// Set the hash of our URL to the path
		window.location.hash = "path:" + body.document.id;

		calc_sub_counts();
	});

	leaps_client.on("global_metadata", function(body) {
		if ( body.metadata.type === "user_info" ) {
			// This also gives us our session_id in body.client.session_id
			for ( var old_user in users ) {
				if ( users.hasOwnProperty(old_user) ) {
					Vue.delete(users, old_user);
				}
			}
			for ( var new_user in body.metadata.body.users ) {
				if ( body.metadata.body.users.hasOwnProperty(new_user) &&
					new_user !== body.client.session_id ) {
					Vue.set(users, new_user, body.metadata.body.users[new_user]);
				}
			}
		}
		if ( body.metadata.type === "cursor_update" ) {
			users[body.client.session_id].position = body.metadata.body.position;
		}
		if ( body.metadata.type === "message" ) {
			show_user_message(body.client.username, body.client.session_id, body.metadata.body.message.content);
		}
		if ( body.metadata.type === "user_subscribe" ) {
			show_sys_message("User " + body.client.username + " opened " + body.metadata.body.document.id);
			users[body.client.session_id].subscriptions = [
				body.metadata.body.document.id
			];
			calc_sub_counts();
		}
		if ( body.metadata.type === "user_unsubscribe" ) {
			if ( users.hasOwnProperty(body.client.session_id) ) {
				show_sys_message("User " + body.client.username + " closed " + body.metadata.body.document.id);
				users[body.client.session_id].subscriptions = [];
				calc_sub_counts();
			}
		}
		if ( body.metadata.type === "user_connect" ) {
			Vue.set(users, body.client.session_id, {
				username: body.client.username,
				position: 0,
				subscriptions: []
			});
			show_sys_message("User " + body.client.username + " has connected");
		}
		if ( body.metadata.type === "user_disconnect" ) {
			Vue.delete(users, body.client.session_id);
			show_sys_message("User " + body.client.username + " has disconnected");
			calc_sub_counts();
		}
		if ( body.metadata.type === "cmd_list" ) {
			cmds_obj.options = []; // Clear old cmds
			if ( body.metadata.body.cmds instanceof Array ) {
				for ( var i = 0; i < body.metadata.body.cmds.length; i++ ) {
					cmds_obj.options.push({
						index: i,
						text: body.metadata.body.cmds[i]
					});
				}
			}
		}
		if ( body.metadata.type === "cmd_output" ) {
			show_cmd_output(body.metadata.body.cmd);
		}
	});

	var protocol = window.location.protocol === "http:" ? "ws:" : "wss:";
	leaps_client.connect(
		protocol + "//" + window.location.host + window.location.pathname + "leaps/ws" +
			"?username=" + encodeURIComponent(username)
	);
}

function join_document(document_id) {
	if ( leaps_client === null ) {
		init_leaps(function() {
			join_document(document_id);
		});
		return;
	}

	try {
		var ext = document_id.substr(document_id.lastIndexOf(".") + 1);
		var info = CodeMirror.findModeByExtension(ext);
		if ( ext.length !== document_id.length && info.mode ) {
			cm_editor.setOption("mode", info.mime);
			CodeMirror.autoLoadMode(cm_editor, info.mode);
		} else {
			cm_editor.setOption("mode", null);
		}
	} catch (e) {
		console.error(e);
	}

	if ( file_paths.opened.length > 0 ) {
		file_paths.next_path = document_id;
		let err = leaps_client.unsubscribe(file_paths.opened);
		if ( err ) {
			console.error(err);
		}
	} else {
		let err = leaps_client.subscribe(document_id);
		if ( err ) {
			console.error(err);
		}
	}
}

function navigate_to_user(user) {
	if ( cm_editor !== null ) {
		if ( (user.subscriptions || []).length > 0 &&
			file_paths.opened !== user.subscriptions[0] ) {
			leaps_client.on_next("subscribe", function() {
				var pos = leap_client.pos_from_u_index(cm_editor.getDoc(), user.position);
				cm_editor.setCursor(pos);
				cm_editor.focus();
			});
			join_document(user.subscriptions[0]);
		} else {
			var pos = leap_client.pos_from_u_index(cm_editor.getDoc(), user.position);
			cm_editor.setCursor(pos);
			cm_editor.focus();
		}
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
		content: content
	});
	clip_messages();
}

function show_err_message(content) {
	var now = new Date();
	messages_obj.messages.push({
		timestamp: now.toLocaleTimeString(),
		is_err: true,
		content: content
	});
	clip_messages();
}

function show_cmd_output(output) {
	if ( output.stdout.length === 0 &&
	     output.stderr.length === 0 &&
	     output.error.length === 0 ) {
		output.stdout = "Empty response";
	}
	var cmd_str = "";
	if ( output.id >= 0 && cmd_str < cmds_obj.options.length ) {
		cmd_str = cmds_obj.options[output.id].text;
	}
	messages_obj.messages.push({
		stdout: output.stdout,
		stderr: output.stderr,
		error: output.error,
		cmd: cmd_str
	});
	clip_messages();
}

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
					path: split_path.slice(0, j+1).join('/'),
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

function calc_sub_counts() {
	var subs = {};
	for ( var user in users ) {
		if ( users.hasOwnProperty(user) &&
		   ( users[user].subscriptions instanceof Array ) ) {
			for ( var i = 0; i < users[user].subscriptions.length; i++ ) {
				if ( !subs.hasOwnProperty(users[user].subscriptions[i]) ) {
					subs[users[user].subscriptions[i]] = 1;
				} else {
					subs[users[user].subscriptions[i]]++;
				}
			}
		}
	}
	if ( file_paths.opened.length > 0 ) {
		if ( !subs.hasOwnProperty(file_paths.opened) ) {
			subs[file_paths.opened] = 1;
		} else {
			subs[file_paths.opened]++;
		}
	}
	Vue.set(file_paths, 'subscriptions', subs);
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
}

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
		Cookies.set("username", username, { path: '', expires: 7 });
		restart_leaps();
	});

	var chat_bar = document.getElementById("chat-bar");
	text_input(chat_bar, function(ele, content) {
		if ( content.length > 0 ) {
			if ( leaps_client !== null ) {
				leaps_client.send_global_metadata({
					type: "message",
					body: {
						message: {
							content: content
						}
					}
				});
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
			},
			sub_count: function() {
				if ( typeof file_paths.subscriptions[this.model.path] === 'number' ) {
					return file_paths.subscriptions[this.model.path];
				}
				return 0;
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
					join_document(this.model.path);
				}
			}
		}
	});

	(new Vue({ el: '#file-list', data: { file_data: file_paths } }));
	(new Vue({ el: '#message-list', data: messages_obj }));
	(new Vue({
		el: '#cmd-menu',
		data: cmds_obj,
		methods: {
			run_cmd: function() {
				if ( cmds_obj.selected >= 0 && cmds_obj.selected < cmds_obj.options.length ) {
					if ( leaps_client !== null ) {
						leaps_client.send_global_metadata({
							type: "cmd",
							body: {
								cmd: {
									id: cmds_obj.selected
								}
							}
						});
						show_cmd_output({
							stdout: "Running `" + cmds_obj.options[cmds_obj.selected].text + "`"
						});
					}
				}
			}
		}
	}));
	(new Vue({
		el: '#users-list',
		data: { users: users },
		methods: {
			style: function(id) {
				return {
					color: "#fcfcfc",
					backgroundColor: leap_client.session_id_to_colour(id)
				};
			},
			go_to: navigate_to_user
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

	init_leaps(function() {
		window.onhashchange = function() {
			// You can link directly to a filepath with <URL>#path:/this/is/the/path.go
			if ( window.location.hash.length > 0 &&
				window.location.hash.substr(1, 5) === "path:" ) {
				var new_path = window.location.hash.substr(6);
				if ( new_path !== file_paths.opened ) {
					join_document(window.location.hash.substr(6));
				}
			}
		};

		// Event isn't triggered on page load but we might want to check.
		window.onhashchange();
	});
};

})();
