package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type listedPost struct {
	ID          int64
	Channel     string
	Link        string
	PublishedAt time.Time
	Snippet     string
	Status      string
	MediaCounts map[string]int
}

const (
	truncateLength = 50
)

func handlePostDetail(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		ackCallback(ctx, b, update)

		lang := langFromContext(ctx)
		chatID, msgID := callbackTarget(update)
		callbackData := update.CallbackQuery.Data

		parts := strings.SplitN(callbackData, ":", 3)
		if len(parts) != 3 {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"malformed post detail callback",
				slog.String("data", callbackData),
			)
			return
		}
		postID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"malformed post id",
				slog.String("data", callbackData),
			)
			return
		}
		origin := parts[2]

		post, err := st.GetPostWithSource(ctx, postID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get post with source",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}
		media, err := st.ListPostMedia(ctx, postID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"list post media",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
		}

		counts := map[string]int{}
		for _, m := range media {
			counts[m.Kind]++
		}

		text := i18n.T(lang, i18n.PostDetail,
			post.ChannelHandle, post.PublishedAt.Time.Format("02.01.2006 15:04"),
			statusLabel(lang, post.Status), post.ExternalID, attachmentsSummary(counts), post.RawText,
		)

		kb := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: i18n.T(lang, i18n.BtnUse), CallbackData: format(scheduleUse, postID, callbackData)}},
			{{Text: i18n.T(lang, i18n.BtnEdit), CallbackData: format(scheduleEdit, postID, callbackData)}},
			{{Text: i18n.T(lang, i18n.BtnSkip), CallbackData: format(scheduleSkip, postID, callbackData)}},
			{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: origin}},
		}}

		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID: msgID, ChatID: chatID, Text: text,
			ReplyMarkup: kb, ParseMode: models.ParseModeHTML,
			LinkPreviewOptions: &models.LinkPreviewOptions{
				IsDisabled: bot.True(),
			},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to post detail failed",
				slog.Int("message_id", msgID),
				slog.Int64("chat_id", chatID),
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func renderPostListPage(
	ctx context.Context, b *bot.Bot, update *models.Update, lang i18n.Lang,
	posts []listedPost, page, tp, total int,
	selectCallback func(id int64) string, prev, next, back string,
) {
	var sb strings.Builder
	sb.WriteString(i18n.T(lang, i18n.PostListHeader, total))
	sb.WriteString("\n\n")

	var rows [][]models.InlineKeyboardButton
	for _, p := range posts {
		sb.WriteString(i18n.T(lang, i18n.PostListEntry,
			p.ID, p.Channel, p.PublishedAt.Format("02.01.2006 15:04"), statusLabel(lang, p.Status),
			truncate(stripTagsForPreview(p.Snippet), truncateLength), attachmentsSummary(p.MediaCounts),
		))
		sb.WriteString("\n\n")
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: i18n.T(lang, i18n.BtnSelectPost, p.ID), CallbackData: selectCallback(p.ID)},
		})
	}
	rows = append(rows, buildPaginationRow(page, tp, prev, next))
	rows = append(rows, []models.InlineKeyboardButton{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: back}})

	chatID, msgID := callbackTarget(update)
	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
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
			slog.Int64("chat_id", chatID),
			slog.Int("message_id", msgID),
			slog.String("error", err.Error()),
		)
	}
}

func statusLabel(lang i18n.Lang, status string) string {
	switch status {
	case "new":
		return i18n.T(lang, i18n.PostStatusNew)
	case "skipped":
		return i18n.T(lang, i18n.PostStatusSkipped)
	case "scheduled":
		return i18n.T(lang, i18n.PostStatusScheduled)
	case "sent":
		return i18n.T(lang, i18n.PostStatusSent)
	default:
		return status
	}
}

func attachmentsSummary(counts map[string]int) string {
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
