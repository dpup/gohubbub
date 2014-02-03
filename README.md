GoHubbub
========

A [PubSubHubbub](https://pubsubhubbub.googlecode.com/) subscriber client library
for Go (golang).

Example
-------

```go
import "github.com/dpup/gohubbub"

// ...

client := gohubbub.NewClient(hubURL, hostname, port, "Testing")
client.Subscribe(topicURL, func(contentType string, body []byte) {
  // Handle update notification.
})
```

See this simple [program](./example/example.go) for a functioning example.
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
