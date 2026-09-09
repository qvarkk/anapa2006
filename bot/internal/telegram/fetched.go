package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type listedPost struct {
	ID          int64
	Channel     string
	PublishedAt time.Time
	Snippet     string
	Status      string
	MediaCounts map[string]int
}

const (
	PostsPerPage    = 5
	ChannelsPerPage = 5
)

func handleListOpen(ctx context.Context, b *bot.Bot, update *models.Update) {
	ackCallback(ctx, b, update)

	lang := langFromContext(ctx)
	chatID, msgID := callbackTarget(update)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: i18n.T(lang, i18n.KeyBtnFetchedByChannel), CallbackData: encodeCallback(callbackListFetchedByGroups, 0)}},
		{{Text: i18n.T(lang, i18n.KeyBtnFetchedLatest), CallbackData: encodeCallback(callbackListFetchedLatest, 0)}},
		{{Text: i18n.T(lang, i18n.KeyBtnBack), CallbackData: string(callbackOpenMenu)}},
	}}
	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID,
		Text: i18n.T(lang, i18n.KeyListFetchedMenuTitle), ReplyMarkup: kb,
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to list fetched menu failed",
			slog.Int64("chat_id", chatID),
			slog.Int("message_id", msgID),
			slog.String("error", err.Error()),
		)
	}
}

func handleGroupByChannels(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		ackCallback(ctx, b, update)

		lang := langFromContext(ctx)
		page := parseIntCallbackPart(update.CallbackQuery.Data, 2) // callbackListFetchedByGroups = "grp:p:<page>"

		total, err := st.CountSources(ctx)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"count sources",
				slog.String("error", err.Error()),
			)
			return
		}
		tp := totalPages(int(total), ChannelsPerPage)
		page = clampPage(page, tp)

		sources, err := st.ListSourcesWithNewCount(ctx, db.ListSourcesWithNewCountParams{
			Limit: ChannelsPerPage, Offset: int64(page * ChannelsPerPage),
		})
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"list sources",
				slog.String("error", err.Error()),
			)
			return
		}

		var rows [][]models.InlineKeyboardButton
		for _, s := range sources {
			label := i18n.TN(lang, i18n.KeyChannelNewCount, int(s.NewCount), s.ChannelHandle)
			rows = append(rows, []models.InlineKeyboardButton{
				{Text: label, CallbackData: fmt.Sprintf("grp:c:%d:0:%d", s.ID, page)},
			})
		}

		prev, next := string(callbackNoop), string(callbackNoop)
		if page > 0 {
			prev = fmt.Sprintf(string(callbackListFetchedByGroups), page-1)
		}
		if page < tp-1 {
			next = fmt.Sprintf(string(callbackListFetchedByGroups), page+1)
		}

		rows = append(rows, buildPaginationRow(page, tp, prev, next))
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: i18n.T(lang, i18n.KeyBtnBack), CallbackData: string(callbackListFetched)},
		})

		chatID, msgID := callbackTarget(update)
		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID,
			Text:        i18n.T(lang, i18n.KeyListFetchedChannelTitle),
			ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to list fetched channel failed",
				slog.Int64("chat_id", chatID),
				slog.Int("message_id", msgID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func handleLatest(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		ackCallback(ctx, b, update)

		lang := langFromContext(ctx)
		page := parseIntCallbackPart(update.CallbackQuery.Data, 1) // callbackListFetchedLatest = "lat:<page>"

		total, err := st.CountPosts(ctx)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"count posts",
				slog.String("error", err.Error()),
			)
			return
		}
		tp := totalPages(int(total), PostsPerPage)
		page = clampPage(page, tp)

		rows, err := st.ListPostsLatest(ctx, db.ListPostsLatestParams{
			Limit: PostsPerPage, Offset: int64(page * PostsPerPage),
		})
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"list latest posts",
				slog.String("error", err.Error()),
			)
			return
		}

		posts := make([]listedPost, 0, len(rows))
		for _, r := range rows {
			posts = append(posts, listedPost{
				ID: r.ID, Channel: r.ChannelHandle, PublishedAt: r.PublishedAt.Time,
				Snippet: r.RawText, Status: r.Status, MediaCounts: mediaCounts(ctx, st, r.ID),
			})
		}

		prev, next := string(callbackNoop), string(callbackNoop)
		if page > 0 {
			prev = encodeCallback(callbackListFetchedLatest, page-1)
		}
		if page < tp-1 {
			next = encodeCallback(callbackListFetchedLatest, page+1)
		}

		renderPostListPage(ctx, b, update, lang, posts, page, tp, int(total),
			func(id int64) string {
				return encodeCallback(
					callbackLatestPostDetail, id,
					encodeCallback(callbackListFetchedLatest, page),
				)
			},
			prev, next, string(callbackListFetched))
	}
}

func handleChannelPosts(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		ackCallback(ctx, b, update)
		lang := langFromContext(ctx)
		data := update.CallbackQuery.Data //  "grp:c:<sourceID>:<postPage>:<grpPage>"
		sourceID := int64(parseIntCallbackPart(data, 2))
		postPage := parseIntCallbackPart(data, 3)
		grpPage := parseIntCallbackPart(data, 4)

		total, err := st.CountPostsBySource(ctx, sourceID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"count posts by source",
				slog.Int64("source_id", sourceID),
				slog.String("error", err.Error()),
			)
			return
		}
		tp := totalPages(int(total), PostsPerPage)
		postPage = clampPage(postPage, tp)

		rows, err := st.ListPostsBySource(ctx, db.ListPostsBySourceParams{
			SourceID: sourceID, Limit: PostsPerPage, Offset: int64(postPage * PostsPerPage),
		})
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"list posts by source",
				slog.Int64("source_id", sourceID),
				slog.String("error", err.Error()),
			)
			return
		}

		posts := make([]listedPost, 0, len(rows))
		for _, r := range rows {
			posts = append(posts, listedPost{
				ID: r.ID, Channel: r.ChannelHandle, PublishedAt: r.PublishedAt.Time,
				Snippet: r.RawText, Status: r.Status, MediaCounts: mediaCounts(ctx, st, r.ID),
			})
		}

		prev, next := string(callbackNoop), string(callbackNoop)
		if postPage > 0 {
			prev = encodeCallback(callbackListFetchedGroupPosts, sourceID, postPage-1, grpPage)
		}
		if postPage < tp-1 {
			next = encodeCallback(callbackListFetchedGroupPosts, sourceID, postPage+1, grpPage)
		}

		renderPostListPage(ctx, b, update, lang, posts, postPage, tp, int(total),
			func(id int64) string {
				return encodeCallback(
					callbackLatestPostDetail, id,
					encodeCallback(callbackListFetchedGroupPosts, sourceID, postPage, grpPage),
				)
			},
			prev, next, encodeCallback(callbackListFetchedGroups, grpPage))
	}
}

func handlePostDetail(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		ackCallback(ctx, b, update)
		lang := langFromContext(ctx)

		parts := strings.SplitN(update.CallbackQuery.Data, ":", 3) // "post:<id>:<origin>"
		if len(parts) != 3 {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"malformed post detail callback",
				slog.String("data", update.CallbackQuery.Data),
			)
			return
		}
		postID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"malformed post id",
				slog.String("data", update.CallbackQuery.Data),
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

		chatID, _ := callbackTarget(update)
		sendPostMedia(ctx, b, chatID, media)

		counts := map[string]int{}
		for _, m := range media {
			counts[m.Kind]++
		}

		text := i18n.T(lang, i18n.KeyPostDetail,
			post.ChannelHandle, post.PublishedAt.Time.Format("02.01.2006 15:04"),
			statusLabel(lang, post.Status), attachmentsSummary(counts), post.RawText,
		)

		kb := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: i18n.T(lang, i18n.KeyBtnUse), CallbackData: "noop"}},
			{{Text: i18n.T(lang, i18n.KeyBtnEdit), CallbackData: "noop"}},
			{{Text: i18n.T(lang, i18n.KeyBtnSkip), CallbackData: "noop"}},
			{{Text: i18n.T(lang, i18n.KeyBtnBack), CallbackData: origin}},
		}}

		// TODO: think, mark, think
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID, Text: text, ReplyMarkup: kb,
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"send post detail",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func mediaCounts(ctx context.Context, st *store.Store, postID int64) map[string]int {
	rows, err := st.CountMediaKindsByPost(ctx, postID)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"count media kinds",
			slog.Int64("post_id", postID),
			slog.String("error", err.Error()),
		)
		return nil
	}
	m := make(map[string]int, len(rows))
	for _, r := range rows {
		m[r.Kind] = int(r.Cnt)
	}
	return m
}

func renderPostListPage(
	ctx context.Context, b *bot.Bot, update *models.Update, lang i18n.Lang,
	posts []listedPost, page, tp, total int,
	selectCallback func(id int64) string, prev, next, back string,
) {
	var sb strings.Builder
	from := page*PostsPerPage + 1
	to := from + len(posts) - 1
	sb.WriteString(i18n.T(lang, i18n.KeyPostListHeader, from, to, total))
	sb.WriteString("\n\n")

	var rows [][]models.InlineKeyboardButton
	for i, p := range posts {
		ordinal := page*PostsPerPage + i + 1
		sb.WriteString(i18n.T(lang, i18n.KeyPostListEntry,
			p.ID, p.Channel, p.PublishedAt.Format("02.01.2006 15:04"),
			statusLabel(lang, p.Status), truncate(p.Snippet, 40), attachmentsSummary(p.MediaCounts),
		))
		sb.WriteString("\n\n")
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: i18n.T(lang, i18n.KeyBtnSelectPost, ordinal, p.ID), CallbackData: selectCallback(p.ID)},
		})
	}
	rows = append(rows, buildPaginationRow(page, tp, prev, next))
	rows = append(rows, []models.InlineKeyboardButton{{Text: i18n.T(lang, i18n.KeyBtnBack), CallbackData: back}})

	chatID, msgID := callbackTarget(update)
	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID,
		Text: sb.String(), ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
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
		return i18n.T(lang, i18n.KeyPostStatusNew)
	case "skipped":
		return i18n.T(lang, i18n.KeyPostStatusSkipped)
	case "reviewing":
		return i18n.T(lang, i18n.KeyPostStatusReviewing)
	case "archived":
		return i18n.T(lang, i18n.KeyPostStatusArchived)
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
			parts = append(parts, fmt.Sprintf("%s×%d", o.emoji, c))
		}
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, " ")
}

// cuts by rune, so Cyrillic text never gets sliced mid-character
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func sendPostMedia(ctx context.Context, b *bot.Bot, chatID int64, media []db.PostMedium) {
	var group []models.InputMedia
	for _, m := range media {
		switch m.Kind { // TODO: add a type for media
		case "photo":
			group = append(group, &models.InputMediaPhoto{Media: m.Url})
		case "video":
			group = append(group, &models.InputMediaVideo{Media: m.Url})
		}
	}
	switch len(group) {
	case 0:
		return
	case 1:
		switch v := group[0].(type) {
		case *models.InputMediaPhoto:
			if _, err := b.SendPhoto(ctx, &bot.SendPhotoParams{ChatID: chatID, Photo: &models.InputFileString{Data: v.Media}}); err != nil {
				slog.LogAttrs(
					ctx, slog.LevelWarn,
					"send photo",
					slog.String("error", err.Error()),
				)
			}
		case *models.InputMediaVideo:
			if _, err := b.SendVideo(ctx, &bot.SendVideoParams{ChatID: chatID, Video: &models.InputFileString{Data: v.Media}}); err != nil {
				slog.LogAttrs(
					ctx, slog.LevelWarn,
					"send video",
					slog.String("error", err.Error()),
				)
			}
		}
	default:
		if _, err := b.SendMediaGroup(ctx, &bot.SendMediaGroupParams{ChatID: chatID, Media: group}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"send media group",
				slog.String("error", err.Error()),
			)
		}
	}
}
