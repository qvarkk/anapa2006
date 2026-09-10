package telegram

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	PostsPerPage    = 5
	ChannelsPerPage = 5
)

func handleFetch(ctx context.Context, b *bot.Bot, update *models.Update) {
	ackCallback(ctx, b, update)

	lang := langFromContext(ctx)
	chatID, msgID := callbackTarget(update)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: i18n.T(lang, i18n.BtnFetchChannels), CallbackData: format(fetchChannels, 0)}},
		{{Text: i18n.T(lang, i18n.BtnFetchLatest), CallbackData: format(fetchLatest, 0)}},
		{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: menu}},
	}}
	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID,
		Text: i18n.T(lang, i18n.FetchMenu), ReplyMarkup: kb,
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to fetch list menu failed",
			slog.Int64("chat_id", chatID),
			slog.Int("message_id", msgID),
			slog.String("error", err.Error()),
		)
	}
}

func handleFetchChannels(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		ackCallback(ctx, b, update)

		lang := langFromContext(ctx)
		page := parseIntCallbackPart(update.CallbackQuery.Data, 2)

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
			label := i18n.TN(lang, i18n.ChannelNewCount, int(s.NewCount), s.ChannelHandle)
			rows = append(rows, []models.InlineKeyboardButton{
				{Text: label, CallbackData: format(fetchChannelPosts, s.ID, 0, page)},
			})
		}

		prev, next := firstPage, lastPage
		if page > 0 {
			prev = format(fetchChannels, page-1)
		}
		if page < tp-1 {
			next = format(fetchChannels, page+1)
		}

		rows = append(rows, buildPaginationRow(page, tp, prev, next))
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: i18n.T(lang, i18n.BtnBack), CallbackData: fetch},
		})

		chatID, msgID := callbackTarget(update)
		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID,
			Text:        i18n.T(lang, i18n.FetchChannelsMenu),
			ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to fetch channel failed",
				slog.Int64("chat_id", chatID),
				slog.Int("message_id", msgID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func handleFetchLatest(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		ackCallback(ctx, b, update)

		lang := langFromContext(ctx)
		page := parseIntCallbackPart(update.CallbackQuery.Data, 1)

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
				"list posts latest",
				slog.String("error", err.Error()),
			)
			return
		}

		posts := make([]listedPost, 0, len(rows))
		for _, r := range rows {
			posts = append(posts, listedPost{
				ID: r.ID, Channel: r.ChannelHandle, Link: r.ExternalID, PublishedAt: r.PublishedAt.Time,
				Snippet: r.RawText, Status: r.Status, MediaCounts: mediaCounts(ctx, st, r.ID),
			})
		}

		prev, next := firstPage, lastPage
		if page > 0 {
			prev = format(fetchLatest, page-1)
		}
		if page < tp-1 {
			next = format(fetchLatest, page+1)
		}

		renderPostListPage(ctx, b, update, lang, posts, page, tp, int(total),
			func(id int64) string {
				return format(
					fetchPost, id,
					format(fetchLatest, page),
				)
			},
			prev, next, fetch)
	}
}

func handleFetchChannelPosts(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		ackCallback(ctx, b, update)
		lang := langFromContext(ctx)
		data := update.CallbackQuery.Data
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
				ID: r.ID, Channel: r.ChannelHandle, Link: r.ExternalID, PublishedAt: r.PublishedAt.Time,
				Snippet: r.RawText, Status: r.Status, MediaCounts: mediaCounts(ctx, st, r.ID),
			})
		}

		prev, next := firstPage, lastPage
		if postPage > 0 {
			prev = format(fetchChannelPosts, sourceID, postPage-1, grpPage)
		}
		if postPage < tp-1 {
			next = format(fetchChannelPosts, sourceID, postPage+1, grpPage)
		}

		renderPostListPage(ctx, b, update, lang, posts, postPage, tp, int(total),
			func(id int64) string {
				return format(
					fetchPost, id,
					format(fetchChannelPosts, sourceID, postPage, grpPage),
				)
			},
			prev, next, format(fetchChannels, grpPage))
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
