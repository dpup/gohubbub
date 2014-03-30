GoHubbub
========

A [PubSubHubbub](https://pubsubhubbub.googlecode.com/) subscriber client library
for Go (golang).

Example
-------

gohubbub can start its own HTTP server:

```go
import "github.com/dpup/gohubbub"

// ...

client := gohubbub.NewClient(hostname, port, "Testing")
client.DiscoverAndSubscribe(topicURL, func(contentType string, body []byte) {
  // Handle update notification.
})
client.StartAndServe()
```

Or if you have your own server, you can register the gohubbub request handler on
your own mux and then call `client.Run()` when your server is running:

```go
import "github.com/dpup/gohubbub"

// ...

client := gohubbub.NewClient(hostname, port, "Testing")
client.DiscoverAndSubscribe(topicURL, func(contentType string, body []byte) {
  // Handle update notification.
})
client.RegisterHandler(myMux)

// ...

client.Start()

```

See this simple [program](./example/example.go) for a fully functioning example.
You will need to run it from a machine that is publicly accessible or use
[reverse port forwarding](https://medium.com/dev-tricks/220030f3c84a) from one
that is.


Contributing
------------
Questions, comments, bug reports, and pull requests are all welcome.  Submit
them [on the project issue tracker](https://github.com/dpup/gohubbub/issues/new).

License
-------
Copyright 2014 [Daniel Pupius](http://pupius.co.uk). Licensed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0).
