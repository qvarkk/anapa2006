package reply

import (
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/go-telegram/bot/models"
	"golang.org/x/net/html"
)

func entitiesToHTML(text string, entities []models.MessageEntity) string {
	if len(entities) == 0 {
		return html.EscapeString(text)
	}

	type boundary struct {
		pos  int
		open bool
		e    models.MessageEntity
	}

	sorted := make([]models.MessageEntity, len(entities))
	copy(sorted, entities)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Offset != sorted[j].Offset {
			return sorted[i].Offset < sorted[j].Offset
		}
		return sorted[i].Length > sorted[j].Length
	})

	var bounds []boundary
	for _, e := range sorted {
		bounds = append(bounds,
			boundary{pos: int(e.Offset), open: true, e: e},
			boundary{pos: int(e.Offset + e.Length), open: false, e: e},
		)
	}

	sort.SliceStable(bounds, func(i, j int) bool { return bounds[i].pos < bounds[j].pos })

	byPos := map[int][]boundary{}
	for _, b := range bounds {
		byPos[b.pos] = append(byPos[b.pos], b)
	}

	var sb strings.Builder
	var stack []models.MessageEntity
	units := utf16.Encode([]rune(text))

	for i := 0; i <= len(units); i++ {
		for _, b := range byPos[i] {
			if !b.open {
				var toReopen []models.MessageEntity

				for len(stack) > 0 {
					top := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					sb.WriteString(closeTag(top))

					if sameEntity(top, b.e) {
						break
					}

					toReopen = append(toReopen, top)
				}

				for j := len(toReopen) - 1; j >= 0; j-- {
					sb.WriteString(openTag(toReopen[j]))
					stack = append(stack, toReopen[j])
				}
			}
		}

		for _, b := range byPos[i] {
			if b.open {
				sb.WriteString(openTag(b.e))
				stack = append(stack, b.e)
			}
		}

		if i == len(units) {
			break
		}

		u := units[i]
		if utf16.IsSurrogate(rune(u)) && i+1 < len(units) {
			r := utf16.DecodeRune(rune(u), rune(units[i+1]))
			sb.WriteString(html.EscapeString(string(r)))
			i++
		} else {
			sb.WriteString(html.EscapeString(string(rune(u))))
		}
	}

	return sb.String()
}

func sameEntity(a, b models.MessageEntity) bool {
	return a.Offset == b.Offset && a.Length == b.Length && a.Type == b.Type
}

func openTag(e models.MessageEntity) string {
	switch e.Type {
	case models.MessageEntityTypeBold:
		return "<b>"
	case models.MessageEntityTypeItalic:
		return "<i>"
	case models.MessageEntityTypeUnderline:
		return "<u>"
	case models.MessageEntityTypeStrikethrough:
		return "<s>"
	case models.MessageEntityTypeCode:
		return "<code>"
	case models.MessageEntityTypePre:
		return "<pre>"
	case models.MessageEntityTypeSpoiler:
		return `<span class="tg-spoiler">`
	case models.MessageEntityTypeTextLink:
		return `<a href="` + html.EscapeString(e.URL) + `">`
	case models.MessageEntityTypeBlockquote:
		return "<blockquote>"
	default:
		return ""
	}
}

func closeTag(e models.MessageEntity) string {
	switch e.Type {
	case models.MessageEntityTypeBold:
		return "</b>"
	case models.MessageEntityTypeItalic:
		return "</i>"
	case models.MessageEntityTypeUnderline:
		return "</u>"
	case models.MessageEntityTypeStrikethrough:
		return "</s>"
	case models.MessageEntityTypeCode:
		return "</code>"
	case models.MessageEntityTypePre:
		return "</pre>"
	case models.MessageEntityTypeSpoiler:
		return "</span>"
	case models.MessageEntityTypeTextLink:
		return "</a>"
	case models.MessageEntityTypeBlockquote:
		return "</blockquote>"
	default:
		return ""
	}
}
