package fetcher

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type MediaType string

const (
	MediaTypePhoto    MediaType = "photo"
	MediaTypeVideo    MediaType = "video"
	MediaTypeDocument MediaType = "document"
)

var telegramInlineTags = map[string]bool{
	"b": true, "strong": true,
	"i": true, "em": true,
	"u": true, "ins": true,
	"s": true, "strike": true, "del": true,
	"code": true, "pre": true, "tg-spoiler": true,
}

type MediaItem struct {
	URL       string
	MediaType MediaType
}

func ParseDescription(raw string) (captionHTML string, media []MediaItem, err error) {
	nodes, err := html.ParseFragment(strings.NewReader(raw), &html.Node{
		Type: html.ElementNode, Data: "body", DataAtom: atom.Body,
	})
	if err != nil {
		return "", nil, err
	}

	var sb strings.Builder
	for _, n := range nodes {
		render(n, &sb, &media)
	}

	return strings.TrimSpace(collapseBlankLines(sb.String())), media, nil
}

func render(n *html.Node, sb *strings.Builder, media *[]MediaItem) {
	switch n.Type {
	case html.TextNode:
		sb.WriteString(html.EscapeString(n.Data))
		return
	case html.ElementNode:
		switch {
		case n.Data == "img":
			if src := attr(n, "src"); src != "" {
				*media = append(*media, MediaItem{MediaType: MediaTypePhoto, URL: src})
			}
		case n.Data == "video":
			if src := attr(n, "src"); src != "" {
				*media = append(*media, MediaItem{MediaType: MediaTypeVideo, URL: src})
			}
		case n.Data == "blockquote":
			*media = append(*media, MediaItem{MediaType: MediaTypeDocument, URL: ""})
		case n.Data == "small":
			// file-size annotations etc.
		case n.Data == "p" || n.Data == "div":
			renderChildren(n, sb, media)
			sb.WriteString("\n\n")
		case n.Data == "br":
			sb.WriteString("\n")
		case n.Data == "a":
			href := attr(n, "href")
			sb.WriteString(`<a href="`)
			sb.WriteString(html.EscapeString(href))
			sb.WriteString(`">`)
			renderChildren(n, sb, media)
			sb.WriteString("</a>")
		case n.Data == "span" && attr(n, "class") == "tg-spoiler":
			sb.WriteString(`<span class="tg-spoiler">`)
			renderChildren(n, sb, media)
			sb.WriteString("</span>")
		case telegramInlineTags[n.Data]:
			sb.WriteString("<")
			sb.WriteString(n.Data)
			sb.WriteString(">")
			renderChildren(n, sb, media)
			sb.WriteString("</")
			sb.WriteString(n.Data)
			sb.WriteString(">")
		default:
			renderChildren(n, sb, media)
		}
	}
}

func renderChildren(n *html.Node, sb *strings.Builder, media *[]MediaItem) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		render(c, sb, media)
	}
}

func collapseBlankLines(s string) string {
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
