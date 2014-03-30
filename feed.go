// Copyright 2014 Daniel Pupius

package gohubbub

// Multipurpose struct used for deserializing both RSS and Atom feeds.
type feed struct {
	Link    []link  `xml:"link"`
	Channel channel `xml:"channel"`
}

type channel struct {
	Link []link `xml:"http://www.w3.org/2005/Atom link"`
}

type link struct {
	Rel  string `xml:"rel,attr,omitempty"`
	Href string `xml:"href,attr"`
}
