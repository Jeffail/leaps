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
		client.join_document("username", "ignored_auth_token", "test_document");
	});

	client.connect("ws://" + window.location.host + "/socket");

}
```

The path to use for connecting to a leaps service will depend on your service
configuration. Once a document is joined by the leaps client it will begin
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
leaps_client.on("error", function(err) {
	// Do important stuff
	console.error(err);
});

leaps_client.on("disconnect", function() {
	// Do important stuff
	console.log("we are disconnected and stuff");
});
```

When leaps emits an error it will also disconnect, depending on the error
severity this may not result in the disconnect event being emitted. If a
disconnect is unexpected, or an error occurs, then you need to construct a new
instance of leap_client in order to reconnect to the document.

## Other Events

The leaps client emits a variety of events that can also be used to further
improve your interface, here is a full list of all event types.

### connect

```JavaScript
leaps_client.on("connect", function() {
```

Occurs when the leaps client is connected to the target server.

### disconnect

```JavaScript
leaps_client.on("disconnect", function() {
```

Occurs when the leaps client is no longer connected to the target server.

### document

```JavaScript
leaps_client.on("document", function(doc) {
```

A full document has been received from the leaps server, this is normally used
to initially populate the editor but is done automatically for you when the
leaps client is bound to an editor.

Arguments:

- document doc, a document object (outlined below)

### transforms

```JavaScript
leaps_client.on("transforms", function(transforms) {
```

Some transforms have been received from the server and should be applied to the
editor, this is done automatically when the leaps client is bound to an editor.

Arguments:

- transform[] transforms, an array of transform objects (outlined below)

### user

```JavaScript
leaps_client.on("user", function(user_update) {
```

The user event is used to distribute metadata updates coming from other users
currently connected to your document. These updates reveal the user id (the
token used to authenticate with the server, or otherwise a random GUID) and some
potential snippets of info such as a cursor position or a text message.

Arguments:

- user\_object user\_update, a user\_object containing updates about a specific
  user (outlined below)

### error

```JavaScript
leaps_client.on("error", function(err) {
```

An error has occurred, all errors are treated as fatal and the leaps client
 should be replaced.

Arguments:

- string err, a string containing the error message

## User Authentication

There are multiple solutions for adding user authorization when connecting to
documents through leaps. When an authentication method is being used you must
provide a user token when joining or creating documents. This is given along
with your username as a second argument to the join\_document and
create\_document methods like so:

```JavaScript
leaps_client.on("connect", function() {
	leaps_client.join_document("username", "auth_token", "test_document");
});
```

The origin of user_token will vary depending on the solution you have, but
generally this will come from another service you have that checks document
access to a user and generates a one use access token.

## Sending meta data

It's possible to send metadata from one user to all other users connected to a
document through leaps. Messages that are dispatched will be received by all
other users through the `user` event. To do this you call
`leaps_client.send_message` with a string argument. You can use `JSON.stringify`
to send objects, but make sure that you remember to parse the output.

```JavaScript
leaps_client.on("user", function(user_update) {
	if ( 'string' === typeof user_update.message.content ) {
		var metadata = JSON.parse(user_update.message.content);
		console.log("Tag: " + metadata.tag + ", Message: " + metadata.text);
	}
});

// Dispatched to all other users
leaps_client.send_message(JSON.stringify({
	tag: "edgyteenager",
	text: "Life is so pointless..."
});
```

## Types

### document

A document is an object of the following format:
```JavaScript
{
	id: "4h5g4jh6ghj3456ghg45hjg6",    // string, unique ID for this document
	content: "hello world"             // string, the current content
}
```

### transform

A transform is an object describing a specific change of a document. The object
will differ depending on the type of the document, but for text documents it
will be of the format:
```JavaScript
{
	position: 0,           // int, index of the change
	num_delete: 5,         // int, number of characters to delete
	insert: "hello world", // string, text to insert
	version: 2             // int, the version number of this change
}
```

### user_object

A user object always contains a 'user_id' field and the boolean field 'active',
all other fields are optional and should be validated as existing before using.
```JavaScript
{
	client : {
		user_id: "jeffail",    // string, the id/username of this user
		session_id: "<UUID>"   // string, unique per user per connection
	},
	message : {
		active: true,          // bool, whether this user is currently connected
		position: 5,           // int, the latest cursor position of this user
		content: "hello world" // string, a message sent
	}
}
```
