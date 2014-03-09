// Copyright 2014 Daniel Pupius

// A very simple PubSubHubbub subscriber client that watches a feed that can be
// easily updated by visiting http://push-pub.appspot.com/.
//
// Usage:
// $ go run example/example.go --host=[host or ip] --port=[port number]
//
// The Hub will need to be able to communicate with your server.  If your dev
// machine isn't publically accessible consider setting up reverse port
// forwarding from EC2 or a VPS (see https://medium.com/dev-tricks/220030f3c84a)
//
// To gracefully shutdown the server and unsubscribe from the hub, press enter.
//
package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/dpup/gohubbub"
)

type Feed struct {
	Status  string  `xml:"status>http"`
	Entries []Entry `xml:"entry"`
}

type Entry struct {
	URL       string `xml:"id"`
	Published string `xml:"published"`
	Title     string `xml:"title"`
	Content   string `xml:"content"`
}

var host = flag.String("host", "", "Host or IP to serve from")
var port = flag.Int("port", 10000, "The port to serve from")

func main() {
	flag.Parse()

	log.Println("PubSubHubbub Subscriber Started")

	client := gohubbub.NewClient("http://pubsubhubbub.superfeedr.com", *host, *port, "Test App")
	client.Subscribe("http://push-pub.appspot.com/feed", func(contentType string, body []byte) {
		var feed Feed
		xmlError := xml.Unmarshal(body, &feed)

		if xmlError != nil {
			log.Printf("XML Parse Error %v", xmlError)

		} else {
			log.Println(feed.Status)
			for _, entry := range feed.Entries {
				log.Printf("%s - %s (%s)", entry.Title, entry.Content, entry.URL)
			}
		}
	})

	go client.StartAndServe()

	time.Sleep(time.Second * 5)
	log.Println("Press Enter for graceful shutdown...")

	var input string
	fmt.Scanln(&input)

	client.Unsubscribe("http://push-pub.appspot.com/feed")

	time.Sleep(time.Second * 5)
}
