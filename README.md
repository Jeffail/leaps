![Leaps](http://jeffail.uk/images/leaps_logo.png "Leaps")

Leaps is a service for hosting collaborative, live web editors for text documents that can be shared by multiple users. The library uses a method called operational transforms to allow multiple people to contribute and view each others changes simultaneously in real time.

Leaps is ready to be deployed as a service, or alternatively you can use it as a library and write your own personalised service around it. The client library is designed to be highly customizable, with more basic helper functions that simply wrap around textarea elements or other popular web editors in your website.

The service library is designed to be heavily modular and configurable, allowing it to be broken down into a scalable solution of individual components, with both redundancy and parallelism at each component level.

Documentation for the service library can be found here, for those interested in writing custom servers look at the Curator structure:

https://godoc.org/github.com/Jeffail/leaps/leaplib

##How to run:

To start up an example server do the following:

```bash
go get github.com/jeffail/leaps
cd $GOPATH/src/github.com/jeffail/leaps
make example
```

and then visit: http://localhost:8080 to play with an example server.

A leaps service by default will run and host a statistics page to view event counts and uptime, for the example this is hosted at: http://localhost:4040.

##Your own service

Running a customized leaps service is as simple as:

```bash
leaps -c ./leaps_config.js
```

To build up your own service look up the configuration files in ./config. The simple configuration file shows you a minimal config, to see all configuration options check out the default configuration file, and the example file shows you how the leaps example is configured.

##Leaps clients

After building leaps the files you need for the client side can be found in ./bin/js. The files in ./bin/js are simply concatenated and minified versions of the files in ./leapclient, and include the core library as well as some javascript libraries for binding leaps to textareas, Ace editors and CodeMirror editors.

For more guidance on how to use the client look up the examples in ./static, good documentation is soon to come, probably, but for now here's a quick sample:

```javascript
window.onload = function() {
	var client = new leap_client();
	client.bind_textarea(document.getElementById("document"));

	client.on("connect", function() {
		client.join_document("test_document");
	});

	client.connect("ws://" + window.location.host + "/socket");
};
```
