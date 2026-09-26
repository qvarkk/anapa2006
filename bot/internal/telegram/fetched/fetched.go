package fetched

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/pagination"
	"qq/anapa2006/internal/telegram/render"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	PostsPerPage    = 5
	ChannelsPerPage = 5
)

func HandleFetch(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback.Ack(ctx, b, update)

	lang := extract.Lang(ctx)
	chatID, msgID := extract.CallbackTarget(update)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: i18n.T(lang, i18n.BtnFetchChannels), CallbackData: callback.Format(callback.FetchChannels, 0)}},
		{{Text: i18n.T(lang, i18n.BtnFetchLatest), CallbackData: callback.Format(callback.FetchLatest, 0)}},
		{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: callback.Start}},
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

func HandleFetchChannels(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		lang := extract.Lang(ctx)
		page := callback.ParseIntPart(update.CallbackQuery.Data, 2)

		total, err := st.CountSources(ctx)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"count sources",
				slog.String("error", err.Error()),
			)
			return
		}
		tp := pagination.TotalPages(int(total), ChannelsPerPage)
		page = pagination.ClampPage(page, tp)

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
				{Text: label, CallbackData: callback.Format(callback.FetchChannelPosts, s.ID, 0, page)},
			})
		}

		prev, next := callback.FirstPage, callback.LastPage
		if page > 0 {
			prev = callback.Format(callback.FetchChannels, page-1)
		}
		if page < tp-1 {
			next = callback.Format(callback.FetchChannels, page+1)
		}

		rows = append(rows, pagination.BuildPaginationRow(page, tp, prev, next))
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: i18n.T(lang, i18n.BtnBack), CallbackData: callback.Fetch},
		})

		chatID, msgID := extract.CallbackTarget(update)
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

func HandleFetchLatest(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		lang := extract.Lang(ctx)
		page := callback.ParseIntPart(update.CallbackQuery.Data, 1)

		total, err := st.CountPosts(ctx)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"count posts",
				slog.String("error", err.Error()),
			)
			return
		}
		tp := pagination.TotalPages(int(total), PostsPerPage)
		page = pagination.ClampPage(page, tp)

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

		posts := make([]render.RenderedPost, 0, len(rows))
		for _, r := range rows {
			posts = append(posts, render.RenderedPost{
				ID: r.ID, Channel: r.ChannelHandle, Link: r.ExternalID, PublishedAt: r.PublishedAt.Time,
				Snippet: r.RawText, Status: r.Status, MediaCounts: postMediaCounts(ctx, st, r.ID),
			})
		}

		prev, next := callback.FirstPage, callback.LastPage
		if page > 0 {
			prev = callback.Format(callback.FetchLatest, page-1)
		}
		if page < tp-1 {
			next = callback.Format(callback.FetchLatest, page+1)
		}

		payload := render.PostListPayload{
			Bot: b, Update: update, Lang: lang, Page: page, TotalPages: tp, Total: int(total),
			PrevCallback: prev, NextCallback: next, Posts: posts, BackCallback: callback.Fetch,
			SelectCallback: func(id int64) string {
				return callback.Format(
					callback.FetchPost, id,
					callback.Format(callback.FetchLatest, page),
				)
			},
		}

		render.PostListPage(ctx, payload)
	}
}

func HandleFetchChannelPosts(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)
		lang := extract.Lang(ctx)

		data := update.CallbackQuery.Data
		sourceID := int64(callback.ParseIntPart(data, 2))
		postPage := callback.ParseIntPart(data, 3)
		grpPage := callback.ParseIntPart(data, 4)

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
		tp := pagination.TotalPages(int(total), PostsPerPage)
		postPage = pagination.ClampPage(postPage, tp)

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

		posts := make([]render.RenderedPost, 0, len(rows))
		for _, r := range rows {
			posts = append(posts, render.RenderedPost{
				ID: r.ID, Channel: r.ChannelHandle, Link: r.ExternalID, PublishedAt: r.PublishedAt.Time,
				Snippet: r.RawText, Status: r.Status, MediaCounts: postMediaCounts(ctx, st, r.ID),
			})
		}

		prev, next := callback.FirstPage, callback.LastPage
		if postPage > 0 {
			prev = callback.Format(callback.FetchChannelPosts, sourceID, postPage-1, grpPage)
		}
		if postPage < tp-1 {
			next = callback.Format(callback.FetchChannelPosts, sourceID, postPage+1, grpPage)
		}

		payload := render.PostListPayload{
			Bot: b, Update: update, Lang: lang, Page: postPage, TotalPages: tp, Total: int(total),
			PrevCallback: prev, NextCallback: next, Posts: posts,
			BackCallback: callback.Format(callback.FetchChannelPosts, sourceID, postPage, grpPage),
			SelectCallback: func(id int64) string {
				return callback.Format(
					callback.FetchPost, id,
					callback.Format(callback.FetchChannelPosts, sourceID, postPage, grpPage),
				)
			},
		}

		render.PostListPage(ctx, payload)
	}
}

// TODO: refactor to unify with scheduledMediaCounts
func postMediaCounts(ctx context.Context, st *store.Store, postID int64) map[string]int {
	rows, err := st.CountMediaKindsByPost(ctx, postID)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"count post media kinds",
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
