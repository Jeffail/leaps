window.onload = function() {
	"use strict";

	var ace_editor = ace.edit("editor");

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
