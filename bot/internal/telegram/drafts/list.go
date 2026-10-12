package drafts

import (
	"context"
	"database/sql"
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

const RecordsPerPage = 5

type draftRow struct {
	PostID          int64
	FinalText       string
	TargetChannelID sql.NullInt64
	Status          string
	ScheduledAt     sql.NullTime
	SentAt          sql.NullTime
	SentMessageID   sql.NullInt64
	LastError       sql.NullString
	TargetHandle    sql.NullString
}

func HandleList(st *store.Store, kind render.DraftListKind) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		page := callback.ParseIntPart(update.CallbackQuery.Data, 2)
		renderList(ctx, b, st, update, kind, page)
	}
}

func renderList(ctx context.Context, b *bot.Bot, st *store.Store, update *models.Update, kind render.DraftListKind, page int) {
	lang := extract.Lang(ctx)

	total, err := countForKind(ctx, st, kind)
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "count drafts", slog.String("error", err.Error()))
		return
	}
	tp := pagination.TotalPages(int(total), RecordsPerPage)
	page = pagination.ClampPage(page, tp)

	rows, err := listForKind(ctx, st, kind, page)
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "list drafts", slog.String("error", err.Error()))
		return
	}

	drafts := make([]render.RenderedDraft, 0, len(rows))
	for _, r := range rows {
		drafts = append(drafts, render.RenderedDraft{
			PostID:      r.PostID,
			PostStatus:  r.Status,
			ScheduledAt: r.ScheduledAt,
			SentAt:      r.SentAt,
			Link:        render.MessageLink(r.TargetHandle.String, r.SentMessageID.Int64),
			Error:       r.LastError.String,
			Text:        r.FinalText,
			MediaCounts: draftMediaCounts(ctx, st, r.PostID),
		})
	}

	listCallback := callback.Format(listFormat(kind), page)

	prev, next := callback.FirstPage, callback.LastPage
	if page > 0 {
		prev = callback.Format(listFormat(kind), page-1)
	}
	if page < tp-1 {
		next = callback.Format(listFormat(kind), page+1)
	}

	payload := render.DraftListPayload{
		Bot: b, Update: update, Lang: lang, Page: page, TotalPages: tp, Total: int(total),
		Kind: kind, PrevCallback: prev, NextCallback: next, Drafts: drafts,
		BackCallback: callback.DraftsMenu,
		SelectCallback: func(id int64) string {
			return callback.Format(callback.DraftDetail, id, listCallback)
		},
	}

	render.DraftListPage(ctx, payload)
}

func listFormat(kind render.DraftListKind) string {
	switch kind {
	case render.DraftListPosted:
		return callback.DraftsListPosted
	case render.DraftListFailed:
		return callback.DraftsListFailed
	default:
		return callback.DraftsListActive
	}
}

func countForKind(ctx context.Context, st *store.Store, kind render.DraftListKind) (int64, error) {
	switch kind {
	case render.DraftListPosted:
		return st.CountPostedDrafts(ctx)
	case render.DraftListFailed:
		return st.CountFailedDrafts(ctx)
	default:
		return st.CountActiveDrafts(ctx)
	}
}

func listForKind(ctx context.Context, st *store.Store, kind render.DraftListKind, page int) ([]draftRow, error) {
	limit := int64(RecordsPerPage)
	offset := int64(page * RecordsPerPage)

	switch kind {
	case render.DraftListPosted:
		rows, err := st.ListPostedDrafts(ctx, db.ListPostedDraftsParams{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		out := make([]draftRow, len(rows))
		for i, r := range rows {
			out[i] = draftRow{
				PostID: r.PostID, FinalText: r.FinalText, TargetChannelID: r.TargetChannelID,
				Status: r.Status, ScheduledAt: r.ScheduledAt, SentAt: r.SentAt,
				SentMessageID: r.SentMessageID, LastError: r.LastError, TargetHandle: r.TargetHandle,
			}
		}
		return out, nil
	case render.DraftListFailed:
		rows, err := st.ListFailedDrafts(ctx, db.ListFailedDraftsParams{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		out := make([]draftRow, len(rows))
		for i, r := range rows {
			out[i] = draftRow{
				PostID: r.PostID, FinalText: r.FinalText, TargetChannelID: r.TargetChannelID,
				Status: r.Status, ScheduledAt: r.ScheduledAt, SentAt: r.SentAt,
				SentMessageID: r.SentMessageID, LastError: r.LastError, TargetHandle: r.TargetHandle,
			}
		}
		return out, nil
	default:
		rows, err := st.ListActiveDrafts(ctx, db.ListActiveDraftsParams{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		out := make([]draftRow, len(rows))
		for i, r := range rows {
			out[i] = draftRow{
				PostID: r.PostID, FinalText: r.FinalText, TargetChannelID: r.TargetChannelID,
				Status: r.Status, ScheduledAt: r.ScheduledAt, SentAt: r.SentAt,
				SentMessageID: r.SentMessageID, LastError: r.LastError, TargetHandle: r.TargetHandle,
			}
		}
		return out, nil
	}
}

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
