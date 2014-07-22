var ace_editor;

window.onload = function() {
	"use strict";

	ace_editor = ace.edit("editor");
	ace_editor.setTheme("ace/theme/monokai");
	ace_editor.getSession().setMode("ace/mode/javascript");

	var client = new leap_client();
	client.bind_ace_editor(ace_editor);

	client.on("error", function(err) {
		console.log(JSON.stringify(err));
	});

	client.on("connect", function() {
		client.join_document("test_document");
	});

	client.connect("ws://" + window.location.host + "/socket");
};
