window.onload = function() {
	"use strict";

	var client = new leap_client();
	var textarea = document.getElementById("test");
	var idinput = document.getElementById("idfield");
	var createbtn = document.getElementById("createbtn");
	var joinbtn = document.getElementById("joinbtn");

	var boundarea = new leap_bind_textarea(client, textarea);

	var connected = false;

	createbtn.onclick = function() {
		if (!connected) {
			console.error("Tried to create document without connection");
			return;
		}
		var err = client.create_document("test", "test_document", "this is a new test document");
		if ( err !== undefined ) {
			console.error(err);
		}
	};

	joinbtn.onclick = function() {
		if (!connected) {
			console.error("Tried to join document without connection");
			return;
		}
		if (idinput.value.length === 0) {
			console.error("Tried to join document without an id");
			return;
		}
		var err = client.join_document(idinput.value);
		if ( err !== undefined ) {
			console.error(err);
		}
	};

	client.on("document", function(doc) {
		idinput.value = doc.id;
	});

	client.on("error", function(err) {
		console.log(JSON.stringify(err));
	});

	client.on("connect", function() {
		connected = true;
	});

	var protocol = (window.location.protocol == 'https:') ? "wss://" : "ws://";

	var err = client.connect("ws://" + window.location.host + "/socket");
	if ( err !== undefined ) {
		console.error(err);
	}
};
