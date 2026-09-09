package fetcher

import (
	"encoding/xml"
	"fmt"
	"time"
)

const RsshubTelegramPath = "/telegram/channel"

type RSSResponse struct {
	XMLName xml.Name `xml:"rss"`

	Title string    `xml:"channel>title"`
	Link  string    `xml:"channel>link"`
	Items []RSSItem `xml:"channel>item"`
}

type RSSItem struct {
	Title       string      `xml:"title"`
	Description string      `xml:"description"`
	GUID        string      `xml:"guid"`
	PublishedAt RSSDateTime `xml:"pubDate"`
}

type RSSDateTime struct {
	time.Time
}

// Custom RFC1123 unmarshal for publish date
func (t *RSSDateTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var content string
	if err := d.DecodeElement(&content, &start); err != nil {
		return err
	}

	parsedTime, err := time.Parse(time.RFC1123, content)
	if err != nil {
		return fmt.Errorf("cannot parse RSS date %q: %w", content, err)
	}

	t.Time = parsedTime
	return nil
}
