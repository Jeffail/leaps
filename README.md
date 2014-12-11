![Leaps](leaps_logo.png "Leaps")

Leaps is a service for hosting collaborative, live web editors for text documents that can be shared by multiple users. The library uses a method called operational transforms to allow multiple people to contribute and view each others changes simultaneously in real time whilst ensuring that each user also has the same content.

Leaps is ready to be deployed as a service, or alternatively you can use it as a library and write your own personalised service around it. The client is designed to be simple enough to drop into an existing website with ease, but also to be highly customizable when required.

To read more, check out the wiki: [leaps wiki](https://github.com/Jeffail/leaps/wiki)

##How to run:

Just download a release package and do the following:

```bash
tar -xvf ./leaps-linux_amd64-v0.1.1.tar.gz
cd leaps
./leaps -c ./config/leaps_example.yaml
```

and then visit: http://localhost:8080 to play with an example server.
A leaps service by default will run and host a statistics page to view event counts and uptime, for the example this is hosted at: http://localhost:4040.

To generate a configuration file of all default values:

```bash
# for a JSON file
./leaps --print-json

# for a YAML file
./leaps --print-yaml
```

##Your own service

To learn how to customize your leaps service read here:
[leaps service wiki](https://github.com/Jeffail/leaps/wiki/Service)

##Leaps clients

The leaps client is written in JavaScript and is ready to simply drop into your website. You can read about it here:
[leaps client wiki](https://github.com/Jeffail/leaps/wiki/Clients)

The files to include can be found in the release packages at ./js, or in a built repository at ./bin/js. Here's a short example of using leaps to turn a textarea into a shared leaps editor:

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

Then at some point you might want to close it:

```javascript
client.close();
```

If you bound leaps to a textarea or ace document then the document becomes readonly when the connection is lost/closed.

##How to build:

Dependencies:

- Golang 1.2+
- [golint](https://github.com/golang/lint "golint")
- nodejs
- npm (uglifyjs, jshint, nodeunit)

To build and then start up an example server do the following:

```bash
go get github.com/jeffail/leaps
cd $GOPATH/src/github.com/jeffail/leaps
make
./bin/leaps -c ./config/leaps_example.yaml
```

##Contributing and customizing

Documentation for the main service library can be found here, for those interested in writing custom servers look at the Curator structure:

https://godoc.org/github.com/Jeffail/leaps/lib
