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

and then visit: http://localhost:8080 to play with an example server
