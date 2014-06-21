"use strict"

window.onload = function() {
	var comms = document.getElementById("stuff");
	var idfld = document.getElementById("idfld");
	var idbtn = document.getElementById("idbtn");

	if (!'WebSocket' in window) {
		console.log("No websocket support y'all");
		return;
	}

	idbtn.onclick = function() {
		var connection = new WebSocket('ws://localhost:8080/leapsocket');
		var intervalID;
		connection.onmessage = function(e) {
			var server_message = e.data;
			comms.innerHTML += "<p>" + server_message + "</p>";

			var obj = JSON.parse(server_message);
			if (obj.response_type === "document") {
				intervalID = setInterval(function() {
					var data = JSON.stringify({
						command: "submit",
						transforms: [
							{
								position: 0,
								num_delete: 0,
								insert: "hello world",
								version: 2,
								DocID: obj.leap_document.id
							},
							{
								position: 0,
								num_delete: 0,
								insert: "hello world",
								version: 3,
								DocID: obj.leap_document.id
							}
						]
					});
					console.log("sending " + data);
					connection.send(data);
				}, 3000);
			}
		};
		connection.onclose = function(){
			console.log('Connection closed');
			if (intervalID !== undefined) {
				clearInterval(intervalID);
			}
		};
		connection.onopen = function(){
			console.log('Connection opened');
			if (idfld.value !== undefined && idfld.value.length > 0) {
				connection.send(JSON.stringify({
					command: "find",
					document_id: idfld.value
				}));
			} else {
				connection.send(JSON.stringify({
					command: "create",
					leap_document: {
						title: "test",
						description: "test doc",
						content: "hello world"
					}
				}));
			}
		};
	}
};
