package render

import (
	"context"
	"fmt"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/pagination"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	truncateLength = 50
)

type RenderedPost struct {
	ID          int64
	Channel     string
	Link        string
	PublishedAt time.Time
	Snippet     string
	Status      string
	MediaCounts map[string]int
}

type RenderedSchedule struct {
	ID          int64
	ScheduledAt time.Time
	Status      string
	Text        string
	MediaCounts map[string]int
}

type CallbackBuilder func(id int64) string

type renderPayload struct {
	Bot    *bot.Bot
	Update *models.Update
	Lang   i18n.Lang

	Page       int
	TotalPages int
	Total      int

	SelectCallback CallbackBuilder
	PrevCallback   string
	NextCallback   string
	BackCallback   string
}

type PostListPayload struct {
	renderPayload
	Posts []RenderedPost
}

type ScheduledListPayload struct {
	renderPayload
	Scheduled []RenderedSchedule
}

func PostListPage(ctx context.Context, payload PostListPayload) {
	var sb strings.Builder
	sb.WriteString(i18n.T(payload.Lang, i18n.PostListHeader, payload.Total))
	sb.WriteString("\n\n")

	var rows [][]models.InlineKeyboardButton
	for _, p := range payload.Posts {
		sb.WriteString(i18n.T(payload.Lang, i18n.PostListEntry,
			p.ID, p.Channel, p.PublishedAt.Format("02.01.2006 15:04"), PostStatusLabel(payload.Lang, p.Status),
			truncate(stripTagsForPreview(p.Snippet), truncateLength), AttachmentsSummary(p.MediaCounts),
		))
		sb.WriteString("\n\n")
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: i18n.T(payload.Lang, i18n.BtnSelectPost, p.ID), CallbackData: payload.SelectCallback(p.ID)},
		})
	}
	rows = append(rows, pagination.BuildPaginationRow(payload.Page, payload.TotalPages, payload.PrevCallback, payload.NextCallback))
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: i18n.T(payload.Lang, i18n.BtnBack), CallbackData: payload.BackCallback},
	})

	chatID, msgID := extract.CallbackTarget(payload.Update)
	if _, err := payload.Bot.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID, Text: sb.String(),
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
		ParseMode:   models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to render post list failed",
			slog.String("error", err.Error()),
		)
	}
}

func ScheduleListPage(ctx context.Context, payload ScheduledListPayload) {
	var sb strings.Builder
	sb.WriteString(i18n.T(payload.Lang, i18n.ScheduledListHeader, payload.Total))
	sb.WriteString("\n\n")

	var rows [][]models.InlineKeyboardButton
	for _, s := range payload.Scheduled {
		sb.WriteString(i18n.T(payload.Lang, i18n.ScheduledListEntry,
			s.ID, s.ScheduledAt.Format("02.01.2006 15:04"), ScheduleStatusLabel(payload.Lang, s.Status),
			truncate(stripTagsForPreview(s.Text), truncateLength), AttachmentsSummary(s.MediaCounts),
		))
		sb.WriteString("\n\n")
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: i18n.T(payload.Lang, i18n.BtnSelectSchedule, s.ID), CallbackData: payload.SelectCallback(s.ID)},
		})
	}
	rows = append(rows, pagination.BuildPaginationRow(payload.Page, payload.TotalPages, payload.PrevCallback, payload.NextCallback))
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: i18n.T(payload.Lang, i18n.BtnBack), CallbackData: payload.BackCallback},
	})

	chatID, msgID := extract.CallbackTarget(payload.Update)
	if _, err := payload.Bot.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID, Text: sb.String(),
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
		ParseMode:   models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to render scheduled list failed",
			slog.String("error", err.Error()),
		)
	}
}

func AttachmentsSummary(counts map[string]int) string {
	order := []struct{ kind, emoji string }{
		{"photo", "📷"}, {"video", "🎥"}, {"document", "📄"}, {"animation", "🎞"},
	}
	var parts []string
	for _, o := range order {
		if c := counts[o.kind]; c > 0 {
			parts = append(parts, fmt.Sprintf("%s ×%d", o.emoji, c))
		}
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, ", ")
}

func stripTagsForPreview(s string) string {
	nodes, err := html.ParseFragment(strings.NewReader(s), &html.Node{
		Type: html.ElementNode, Data: "body", DataAtom: atom.Body,
	})
	if err != nil {
		return s
	}
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	return sb.String()
}

// Cuts by rune, so Cyrillic text never gets sliced mid-character.
// Finished early if encounters newline
func truncate(s string, max int) string {
	r := []rune(s)

	nlIdx := -1
	for i, c := range r {
		if c == '\n' {
			nlIdx = i
			break
		}
	}

	cut := max
	if nlIdx != -1 && nlIdx < max {
		cut = nlIdx
	}

	if cut >= len(r) {
		return s
	}
	return strings.TrimSpace(string(r[:cut])) + "…"
}
