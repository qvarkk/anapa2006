package drafts

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/pagination"
	"qq/anapa2006/internal/telegram/render"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	RecordsPerPage = 5
)

func HandleList(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data := update.CallbackQuery.Data
		page := callback.ParseIntPart(data, 2)

		total, err := st.CountDrafts(ctx)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"count drafts",
				slog.String("error", err.Error()),
			)
			return
		}
		tp := pagination.TotalPages(int(total), RecordsPerPage)
		page = pagination.ClampPage(page, tp)

		rows, err := st.ListDraftsLatest(ctx, db.ListDraftsLatestParams{
			Limit: RecordsPerPage, Offset: int64(page * RecordsPerPage),
		})
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"list drafts latest",
				slog.String("error", err.Error()),
			)
			return
		}

		drafts := make([]render.RenderedDraft, 0, len(rows))
		for _, r := range rows {
			drafts = append(drafts, render.RenderedDraft{
				PostID: r.PostID, ScheduledAt: r.ScheduledAt, PostStatus: r.Status,
				Text: r.FinalText, MediaCounts: draftMediaCounts(ctx, st, r.PostID),
			})
		}

		prev, next := callback.FirstPage, callback.LastPage
		if page > 0 {
			prev = callback.Format(callback.QueueList, page-1)
		}
		if page < tp-1 {
			next = callback.Format(callback.QueueList, page+1)
		}

		lang := extract.Lang(ctx)

		payload := render.DraftListPayload{
			Bot: b, Update: update, Lang: lang, Page: page, TotalPages: tp, Total: int(total),
			PrevCallback: prev, NextCallback: next, Drafts: drafts, BackCallback: callback.DraftsMenu,
			SelectCallback: func(id int64) string {
				return callback.Format(callback.DraftDetail, id, data)
			},
		}

		render.DraftListPage(ctx, payload)
	}
}

// TODO: refactor to unify with postMediaCounts
func draftMediaCounts(ctx context.Context, st *store.Store, postID int64) map[string]int {
	rows, err := st.CountMediaKindsByDraft(ctx, postID)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"count draft media kinds",
			slog.Int64("draft_id", postID),
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
