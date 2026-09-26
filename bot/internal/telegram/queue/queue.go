package queue

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
	RecordsPerPage = 5
)

func HandleScheduled(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback.Ack(ctx, b, update)
	lang := extract.Lang(ctx)
	chatID, msgID := extract.CallbackTarget(update)

	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID, Text: i18n.T(lang, i18n.Scheduled),
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: i18n.T(lang, i18n.BtnScheduledList), CallbackData: callback.Format(callback.ScheduledList, 0)}},
				{{Text: i18n.T(lang, i18n.BtnScheduledSent), CallbackData: callback.Format(callback.ScheduledSent, 0)}},
				{{Text: i18n.T(lang, i18n.BtnScheduledSettings), CallbackData: callback.Noop}},
				{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: callback.Start}},
			},
		},
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message text scheduled",
			slog.String("error", err.Error()),
		)
	}
}

func HandleScheduledList(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data := update.CallbackQuery.Data
		page := callback.ParseIntPart(data, 2)

		total, err := st.CountScheduled(ctx)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"count scheduled",
				slog.String("error", err.Error()),
			)
			return
		}
		tp := pagination.TotalPages(int(total), RecordsPerPage)
		page = pagination.ClampPage(page, tp)

		rows, err := st.ListScheduledLatest(ctx, db.ListScheduledLatestParams{
			Limit: RecordsPerPage, Offset: int64(page * RecordsPerPage),
		})
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"list posts latest",
				slog.String("error", err.Error()),
			)
			return
		}

		schedules := make([]render.RenderedSchedule, 0, len(rows))
		for _, r := range rows {
			schedules = append(schedules, render.RenderedSchedule{
				ID: r.ID, ScheduledAt: r.ScheduledAt, Status: r.Status,
				Text: r.FinalText, MediaCounts: scheduledMediaCounts(ctx, st, r.ID),
			})
		}

		prev, next := callback.FirstPage, callback.LastPage
		if page > 0 {
			prev = callback.Format(callback.ScheduledList, page-1)
		}
		if page < tp-1 {
			next = callback.Format(callback.ScheduledList, page+1)
		}

		lang := extract.Lang(ctx)

		payload := render.ScheduledListPayload{
			Bot: b, Update: update, Lang: lang, Page: page, TotalPages: tp, Total: int(total),
			PrevCallback: prev, NextCallback: next, Scheduled: schedules,
			BackCallback: callback.Scheduled,
			SelectCallback: func(id int64) string {
				// TODO: add read select callbacks
				return callback.Noop
			},
		}

		render.ScheduleListPage(ctx, payload)
	}
}

// TODO: refactor to unify with postMediaCounts
func scheduledMediaCounts(ctx context.Context, st *store.Store, draftID int64) map[string]int {
	rows, err := st.CountMediaKindsByDraft(ctx, draftID)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"count scheduled media kinds",
			slog.Int64("draft_id", draftID),
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
