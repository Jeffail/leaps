window.onload = function() {
	"use strict";

	var client = new leap_client();
	var textarea = document.getElementById("test");

	var boundarea = new leap_bind_textarea(client, textarea);

	client.subscribe_event("on_error", function(err) {
		console.log(JSON.stringify(err));
	});

	client.subscribe_event("on_connect", function() {
		var err = client.join_document("test_document");
		if ( err !== undefined ) {
			console.error(err);
			return;
		}
	});

	var err = client.connect("ws://localhost:8080/leapsocket");
	if ( err !== undefined ) {
		console.error(err);
		return;
	}
};
