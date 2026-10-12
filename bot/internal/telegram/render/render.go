package render

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/moscow"
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

type DraftListKind string

const (
	DraftListActive DraftListKind = "active"
	DraftListPosted DraftListKind = "posted"
	DraftListFailed DraftListKind = "failed"
)

type RenderedDraft struct {
	PostID      int64
	PostStatus  string
	ScheduledAt sql.NullTime
	SentAt      sql.NullTime
	Link        string
	Error       string
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

type DraftListPayload struct {
	renderPayload
	Kind   DraftListKind
	Drafts []RenderedDraft
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
	if row := pagination.MaybeRow(payload.Page, payload.TotalPages, payload.PrevCallback, payload.NextCallback); row != nil {
		rows = append(rows, row)
	}
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

func DraftListPage(ctx context.Context, payload DraftListPayload) {
	var sb strings.Builder
	sb.WriteString(draftListHeader(payload.Lang, payload.Kind, payload.Total))
	sb.WriteString("\n\n")

	var rows [][]models.InlineKeyboardButton
	for _, d := range payload.Drafts {
		sb.WriteString(draftListEntry(payload.Lang, payload.Kind, d))
		sb.WriteString("\n\n")

		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         i18n.T(payload.Lang, i18n.BtnDraftDetail, d.PostID),
				CallbackData: payload.SelectCallback(d.PostID),
			},
		})
	}

	if payload.TotalPages > 1 {
		rows = append(
			rows,
			pagination.BuildPaginationRow(
				payload.Page,
				payload.TotalPages,
				payload.PrevCallback,
				payload.NextCallback,
			),
		)
	}

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
			"edit message to render drafts list failed",
			slog.String("error", err.Error()),
		)
	}
}

func draftListHeader(lang i18n.Lang, kind DraftListKind, total int) string {
	switch kind {
	case DraftListPosted:
		return i18n.T(lang, i18n.DraftsListPostedHeader, total)
	case DraftListFailed:
		return i18n.T(lang, i18n.DraftsListFailedHeader, total)
	default:
		return i18n.T(lang, i18n.DraftsListActiveHeader, total)
	}
}

func draftListEntry(lang i18n.Lang, kind DraftListKind, d RenderedDraft) string {
	switch kind {
	case DraftListPosted:
		return i18n.T(lang, i18n.DraftListPostedEntry,
			d.PostID, d.SentAt.Time.Format("02.01.2006 15:04"),
			PostedLinkHTML(d.Link),
			truncate(stripTagsForPreview(d.Text), truncateLength),
			AttachmentsSummary(d.MediaCounts),
		)
	case DraftListFailed:
		err := d.Error
		if err == "" {
			err = i18n.T(lang, i18n.DraftErrorUnknown)
		}
		return i18n.T(lang, i18n.DraftListFailedEntry,
			d.PostID, err,
			truncate(stripTagsForPreview(d.Text), truncateLength),
			AttachmentsSummary(d.MediaCounts),
		)
	default:
		scheduledStr := i18n.T(lang, i18n.Unscheduled)
		if d.ScheduledAt.Valid {
			scheduledStr = moscow.Format(d.ScheduledAt.Time)
		}
		return i18n.T(lang, i18n.DraftDetail,
			d.PostID, scheduledStr, PostStatusLabel(lang, d.PostStatus),
			truncate(stripTagsForPreview(d.Text), truncateLength), AttachmentsSummary(d.MediaCounts),
		)
	}
}

func PostedLinkHTML(link string) string {
	if link == "" {
		return "—"
	}
	return `<a href="` + link + `">` + link + `</a>`
}

// MessageLink builds a t.me deep link to a message in a public channel. It
// returns an empty string when no link can be built (private target or no id).
func MessageLink(handle string, messageID int64) string {
	if handle == "" || messageID == 0 {
		return ""
	}
	return fmt.Sprintf("https://t.me/%s/%d", handle, messageID)
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
