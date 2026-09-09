package fetcher

import (
	"strings"

	"golang.org/x/net/html"
)

type MediaType string

const (
	MediaTypePhoto    MediaType = "photo"
	MediaTypeVideo    MediaType = "video"
	MediaTypeDocument MediaType = "document"
)

type MediaItem struct {
	URL       string
	MediaType MediaType
}

func ParseDescription(raw string) (caption string, media []MediaItem, err error) {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return "", nil, err
	}

	var textParts []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "img":
				if src := attr(n, "src"); src != "" {
					media = append(media, MediaItem{MediaType: MediaTypePhoto, URL: src})
				}
				return
			case "video":
				if src := attr(n, "src"); src != "" {
					media = append(media, MediaItem{MediaType: MediaTypeVideo, URL: src})
				}
				return
			case "blockquote":
				media = append(media, MediaItem{MediaType: MediaTypeDocument, URL: ""})
				return
			}
		}

		if n.Type == html.TextNode {
			if t := strings.TrimSpace(n.Data); t != "" {
				textParts = append(textParts, t)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return strings.Join(textParts, " "), media, nil
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
