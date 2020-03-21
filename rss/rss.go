// Package rss ...
package rss

// Image ...
type Image struct {
	URL   string `xml:"url"`
	Title string `xml:"title"`
	Link  string `xml:"link"`
}

// Media ...
type Media struct {
	URL    string `xml:"url,attr"`
	Length int    `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// Item ...
type Item struct {
	Title              string `xml:"title"`
	Description        string `xml:"description"`
	EncodedDescription string `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	PubDate            string `xml:"pubDate"`
	GUID               string `xml:"guid"`
	Media              Media  `xml:"enclosure"`
}

// AtomLink ...
type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

// Channel ...
type Channel struct {
	Title       string   `xml:"title"`
	Link        AtomLink `xml:"http://www.w3.org/2005/Atom link"`
	Language    string   `xml:"language"`
	Copyright   string   `xml:"copright"`
	Description string   `xml:"description"`
	Image       Image    `xml:"image"`
	Items       []Item   `xml:"item"`
}

// Feed ...
type Feed struct {
	Channel Channel `xml:"channel"`
}
