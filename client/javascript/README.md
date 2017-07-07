Client
======

This is the JavaScript implementation of a leaps client. It holds an internal
model for processing incoming and outgoing transforms, and tools for
automatically binding this model around various web page UI elements (currently
supports textarea, ACE editor, and CodeMirror).

To test:

```bash
sudo npm install -g nodeunit jshint
jshint ./*.js
nodeunit test_leapclient.js
```

Here is a basic example:

```JavaScript
var client = new leap_client();

window.onload = function() {
	client.on("connect", function() {
		client.subscribe("test_document");
	});

	client.connect("ws://" + window.location.host + "/socket?username=foo");
}
```

The path to use for connecting to a leaps service will depend on your service
configuration. Once a document is subscribed to by the client it will begin
synchronizing with the server, and therefore any other users connected to the
same document. This of course means nothing if you are not presenting the
document in the web page somehow, so that is the next step.

## Editors

Leaps itself is not a text editor solution, it is a service that runs in the
background to keep an existing editor solution synchronized with other users.
Therefore, to present a leaps document in your web page you need to choose an
editor. Currently, you have these options:

- [Ace Editor](http://ace.c9.io/)
- [CodeMirror](http://codemirror.net/)
- A plain old textarea HTML element

Regardless of which editor you choose, the act of binding leaps to that document
is almost the exact same. It looks like this:

### Ace Editor

```JavaScript
var ace_editor = ace.edit("editor");

var leaps_client = new leap_client();
leaps_client.bind_ace_editor(ace_editor);

// Connect leaps and join document, etc
```

### CodeMirror

```JavaScript
var cmirror_editor = CodeMirror.fromTextArea(document.getElementById("editor"));

var leaps_client = new leap_client();
leaps_client.bind_codemirror(cmirror_editor);

// Connect leaps and join document, etc
```

### Textarea

```JavaScript
var leaps_client = new leap_client();
leaps_client.bind_textarea(document.getElementById("leaps-textarea"));

// Connect leaps and join document, etc
```

Technically, that is all you need to have a fully functioning leaps synchronized
editor in your web page. However, it is good practice to handle any errors
and/or disconnects yourself.

## Handling Errors and Disconnections

Leaps will emit events for errors and disconnections, to use these for
recovering your client you can subscribe like so:

```JavaScript
leaps_client.on("error", function(body) {
	// Do important stuff
	console.error(body)
});

leaps_client.on("disconnect", function() {
	// Do important stuff
	console.log("we are disconnected and stuff");
});
```

## Other Events

The leaps client emits a variety of events that can also be used to further
improve your interface, [the full API spec can be found here][0].

[0]: ../../lib/api/README.md
