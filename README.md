![Leaps](leaps_logo.png "Leaps")

Leaps is a service for hosting collaboratively edited documents using operational transforms to ensure zero-collision synchronization across any number of editing clients.

To read more and find examples check out the wiki: [leaps wiki](https://github.com/Jeffail/leaps/wiki)

##How to run

Leaps is a single binary, with no runtime dependencies, everything is set through a single config file. Just download a release package for your OS and do the following to run an example:

```bash
tar -xvf ./leaps-linux_amd64-v0.1.2.tar.gz
cd leaps
./bin/leaps -c ./config/leaps_example.yaml
```

and then visit: http://localhost:8001 to play with an example server.
A leaps service by default will run and host a statistics page to view event counts and uptime, for the example this is hosted at: http://localhost:4040.

To generate a configuration file of all default values:

```bash
# for a JSON file
./bin/leaps --print-json

# for a YAML file
./bin/leaps --print-yaml
```

##Customizing your service

There are lots of example configuration files in ./config to check out for various use cases.

To learn how to customize your leaps service read here:
[leaps service wiki](https://github.com/Jeffail/leaps/wiki/Service)

##Leaps clients

The leaps client is written in JavaScript and is ready to simply drop into a website. You can read about it here:
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

##System compatibility
OS               | Status
---------------- | ------
OSX x86_64       | Supported, tested
Linux x86        | Supported, tested
Linux x86_64     | Supported, tested
Linux ARMv5      | Supported, tested
Linux ARMv7      | Supported, tested
Windows x86      | Supported
Windows x86_64   | Supported

##How to build

Leaps has a Makefile that can lint, run tests, generate the client libraries and package leaps builds into archives. However, if you only wish to get a leaps binary then you can simply use `go get github.com/jeffail/leaps`.

Build dependencies for Makefile:

- Golang 1.2+
- [golint](https://github.com/golang/lint "golint")
- nodejs
- npm (global install of uglifyjs, jshint, nodeunit)

Make sure you have go 1.2+ and nodejs installed and then:

```bash
go get github.com/golang/lint
sudo npm install -g uglifyjs jshint nodeunit
```

To build and then start up an example server do the following:

```bash
go get github.com/jeffail/leaps
cd $GOPATH/src/github.com/jeffail/leaps

# To build the binary and client libraries:
make build
./bin/leaps -c ./config/leaps_example.yaml

# Or, to build only the service binary:
go build
```

For more build options call `make help`.

##Contributing and customizing

Contributions are very welcome, just fork and submit a pull request.

Godocs for the service library can be found [here](https://godoc.org/github.com/Jeffail/leaps/lib).

##Contact

Ashley Jeffs
* Web: [http://jeffs.eu](http://jeffs.eu)
* Twitter: [@Jeffail](https://twitter.com/Jeffail "@jeffail")
* Email: [ash@jeffs.eu](mailto:ash@jeffs.eu)
