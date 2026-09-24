package telegram

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type listedSchedule struct {
	ID          int64
	ScheduledAt time.Time
	Status      string
	Text        string
	MediaCounts map[string]int
}

func handleScheduled(ctx context.Context, b *bot.Bot, update *models.Update) {
	ackCallback(ctx, b, update)
	lang := langFromContext(ctx)
	chatID, msgID := callbackTarget(update)

	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID, Text: i18n.T(lang, i18n.Scheduled),
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: i18n.T(lang, i18n.BtnScheduledList), CallbackData: format(scheduledList, 0)}},
				{{Text: i18n.T(lang, i18n.BtnScheduledSent), CallbackData: format(scheduledSent, 0)}},
				{{Text: i18n.T(lang, i18n.BtnScheduledSettings), CallbackData: noop}},
				{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: menu}},
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

func handleScheduledList(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		ackCallback(ctx, b, update)

		data := update.CallbackQuery.Data
		page := parseIntCallbackPart(data, 2)

		total, err := st.CountScheduled(ctx)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"count scheduled",
				slog.String("error", err.Error()),
			)
			return
		}
		tp := totalPages(int(total), PostsPerPage)
		page = clampPage(page, tp)

		rows, err := st.ListScheduledLatest(ctx, db.ListScheduledLatestParams{
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

		schedules := make([]listedSchedule, 0, len(rows))
		for _, r := range rows {
			schedules = append(schedules, listedSchedule{
				ID: r.ID, ScheduledAt: r.ScheduledAt, Status: r.Status,
				Text: r.FinalText, MediaCounts: scheduledMediaCounts(ctx, st, r.ID),
			})
		}

		prev, next := firstPage, lastPage
		if page > 0 {
			prev = format(scheduledList, page-1)
		}
		if page < tp-1 {
			next = format(scheduledList, page+1)
		}

		lang := langFromContext(ctx)

		payload := renderScheduledPayload{
			Bot: b, Update: update, Lang: lang, Page: page, TotalPages: tp, Total: int(total),
			PrevCallback: prev, NextCallback: next, Scheduled: schedules,
			BackCallback: scheduled,
			SelectCallback: func(id int64) string {
				// TODO: add read select callbacks
				return noop
			},
		}

		renderScheduleListPage(ctx, payload)
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
