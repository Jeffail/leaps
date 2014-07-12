![Leaps](http://jeffail.uk/images/leaps_logo.png "Leaps")

Leaps is a service for hosting collaborative, live web editors for text documents that can be shared by multiple users. The library uses a method called operational transforms to allow multiple people to contribute and view each others changes simultaneously in real time.

Leaps is ready to be deployed as a service, or alternatively you can use it as a library and write your own personalised service around it. The client library is designed to be highly customizable, with more basic helper functions that simply wrap around textarea elements or other popular web editors in your website.

[leaps wiki](https://github.com/Jeffail/leaps/wiki/Service).

##How to run:

To start up an example server do the following:

```bash
go get github.com/jeffail/leaps
cd $GOPATH/src/github.com/jeffail/leaps
make
./bin/leaps -c ./config/leaps_example.js
```

Or just download a release package and do the following:

```bash
tar -xvf ./leaps-linux_amd64-v0.0.2.tar.gz
cd leaps
./leaps -c ./config/leaps_example.js
```

and then visit: http://localhost:8080 to play with an example server.

A leaps service by default will run and host a statistics page to view event counts and uptime, for the example this is hosted at: http://localhost:4040.

##Your own service

Running a customized leaps service is as simple as:

```bash
leaps -c ./leaps_config.js
```

To learn how to set up your leaps service read here: [leaps service wiki](https://github.com/Jeffail/leaps/wiki/Service).

##Leaps clients

The leaps client is written in JavaScript and is ready to simply drop into your website. You can read about it [here](https://github.com/Jeffail/leaps/wiki/Clients), and the files to include can be found in the release packages at ./js, or in a built repository at ./bin/js.

Here's a short example of using leaps to turn a textarea into a shared leaps editor:

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

##Contributing and customizing

Documentation for the main service library can be found here, for those interested in writing custom servers look at the Curator structure:

https://godoc.org/github.com/Jeffail/leaps/leaplib
