// Copyright 2014 Daniel Pupius

// A PubSubHubbub subscriber client that watches Medium's latest feed.
//
// Usage:
// $ go run example/medium.go --host=[host or ip] --port=[port number]
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
	URL     string `xml:"id"`
	Title   string `xml:"title"`
	Summary string `xml:"summary"`
	Author  Author `xml:"author"`
}

type Author struct {
	Name string `xml:"name"`
}

var host = flag.String("host", "", "Host or IP to serve from")
var port = flag.Int("port", 10000, "The port to serve from")

func main() {
	flag.Parse()

	log.Println("Medium Story Watcher Started...")

	client := gohubbub.NewClient(*host, *port, "MediumStoryWatcher")
	err := client.DiscoverAndSubscribe("https://medium.com/feed/latest", func(contentType string, body []byte) {
		var feed Feed
		xmlError := xml.Unmarshal(body, &feed)

		if xmlError != nil {
			log.Printf("XML Parse Error %v", xmlError)

		} else {
			for _, entry := range feed.Entries {
				log.Printf("%s by %s (%s)", entry.Title, entry.Author.Name, entry.URL)
			}
		}
	})

	if err != nil {
		log.Fatal(err)
	}

	go client.StartAndServe()

	time.Sleep(time.Second * 5)
	log.Println("Press Enter for graceful shutdown...")

	var input string
	fmt.Scanln(&input)

	client.Unsubscribe("https://medium.com/feed/latest")

	time.Sleep(time.Second * 5)
}
