![Leaps](http://jeffail.uk/images/leaps_logo.png "Leaps")

Leaps is a standalone service, a front end JavaScript library and back end Golang library for hosting collaborative, live web editors for text documents that can be shared by multiple users. The library uses a method called operational transforms to allow multiple editors of a document hosted online to contribute and view each others changes simultaneously in real time.

The service library is designed to be heavily modular and configurable, allowing it to be broken down into a scaleable solution of individual components, with both redundancy and parallelism at each component level.

For writing custom services in Golang using the library look at the Curator structure:

https://godoc.org/github.com/Jeffail/leaps/leaplib

For writing clients in JavaScript look at:

http://jeffail.uk/leapclient

##How to run:

Currently there are no releases, so to run an example service you need to check out and build it:

```bash
go get github.com/jeffail/leaps
go run github.com/jeffail/leaps/leapexample/example1.go
```

and then visit: http://localhost:8080 to open an example document.
