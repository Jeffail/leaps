window.onload = function() {
	"use strict";

	var cm_editor = CodeMirror.fromTextArea(document.getElementById("editor"), {
		mode : "text/javascript",
		lineNumbers : true
	});

	var client = new leap_client();
	client.bind_codemirror(cm_editor);

	client.on("error", function(err) {
		console.log(JSON.stringify(err));
	});

	client.on("connect", function() {
		client.join_document("test_document");
	});

	client.connect("ws://" + window.location.host + "/socket");
};
