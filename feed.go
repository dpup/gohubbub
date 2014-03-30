// Copyright 2014 Daniel Pupius

package gohubbub

// Multipurpose struct used for deserializing both RSS and Atom feeds.
type Feed struct {
	Link    []Link  `xml:"link"`
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Link []Link `xml:"http://www.w3.org/2005/Atom link"`
}

type Link struct {
	Rel  string `xml:"rel,attr,omitempty"`
	Href string `xml:"href,attr"`
}
