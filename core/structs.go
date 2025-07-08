package core

import (
	"encoding/xml"
	"time"
)

type Description struct {
	Data string `xml:",cdata"`
}

type AtomEntry struct {
	Title       string      `xml:"title"`
	Link        string      `xml:"link"`
	Description Description `xml:"description"`
	Date        string      `xml:"pubDate"`
	Category    []string    `xml:"category"`
	Enclosure   string      `xml:"enclosure"`
	Id          string      `xml:"guid"`
}

type Image struct {
	Title  string `xml:"title"`
	URL    string `xml:"url"`
	Link   string `xml:"link"`
	Width  int32  `xml:"width"`
	Height int32  `xml:"height"`
}

type AtomChannel struct {
	Title       string      `xml:"title"`
	Link        string      `xml:"link"`
	Description Description `xml:"description"`
	Date        string      `xml:"pubDate"`
	Language    string      `xml:"language"`
	Generator   string      `xml:"generator"`
	TTL         int32       `xml:"ttl"`
	Image       Image       `xml:"image"`

	AtomEntries []AtomEntry `xml:"item"`
}

type AtomFormat struct {
	XMLName     xml.Name    `xml:"rss"`
	AtomChannel AtomChannel `xml:"channel"`
}

type News struct {
	Title       string    `json:"Title"`
	Date        time.Time `json:"Date"`
	Description string    `json:"Description"`
}
