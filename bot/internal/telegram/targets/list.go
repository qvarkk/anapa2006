package targets

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/pagination"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const RecordsPerPage = 5

func HandleList(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		page := callback.ParseIntPart(update.CallbackQuery.Data, 2)
		renderList(ctx, b, st, update, page)
	}
}

func renderList(ctx context.Context, b *bot.Bot, st *store.Store, update *models.Update, page int) {
	lang := extract.Lang(ctx)

	total, err := st.CountTargetChannels(ctx)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"count target channels",
			slog.String("error", err.Error()),
		)
		return
	}

	tp := pagination.TotalPages(int(total), RecordsPerPage)
	page = pagination.ClampPage(page, tp)

	targets, err := st.ListTargetChannels(ctx, db.ListTargetChannelsParams{
		Limit: RecordsPerPage, Offset: int64(page * RecordsPerPage),
	})
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"list target channels",
			slog.String("error", err.Error()),
		)
		return
	}

	listCallback := callback.Format(callback.TargetsList, page)

	var sb strings.Builder
	sb.WriteString(i18n.T(lang, i18n.TargetsListHeader, total))
	sb.WriteString("\n\n")

	rows := make([][]models.InlineKeyboardButton, 0, len(targets)+3)
	for _, t := range targets {
		sb.WriteString(i18n.T(lang, i18n.TargetsListEntry, t.ID, targetLabel(t), t.ChatID))
		sb.WriteString("\n\n")

		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         i18n.T(lang, i18n.BtnTargetDelete, targetLabel(t)),
				CallbackData: callback.Format(callback.TargetsDeletePrompt, t.ID, listCallback),
			},
		})
	}

	if tp > 1 {
		prev, next := callback.FirstPage, callback.LastPage
		if page > 0 {
			prev = callback.Format(callback.TargetsList, page-1)
		}
		if page < tp-1 {
			next = callback.Format(callback.TargetsList, page+1)
		}
		rows = append(rows, pagination.BuildPaginationRow(page, tp, prev, next))
	}

	rows = append(rows, []models.InlineKeyboardButton{
		{Text: i18n.T(lang, i18n.BtnTargetAdd), CallbackData: callback.Format(callback.TargetsAdd, listCallback)},
	})
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: i18n.T(lang, i18n.BtnBack), CallbackData: callback.TargetsMenu},
	})

	chatID, msgID := extract.CallbackTarget(update)
	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID,
		Text:        sb.String(),
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
		ParseMode:   models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to targets list failed",
			slog.String("error", err.Error()),
		)
	}
}

func targetLabel(t db.TargetChannel) string {
	if t.Handle.Valid && t.Handle.String != "" {
		return "@" + t.Handle.String
	}
	if t.Title != "" {
		return fmt.Sprintf("%s (ID %d)", t.Title, t.ChatID)
	}
	return fmt.Sprintf("ID %d", t.ChatID)
}

func nullHandle(handle string) sql.NullString {
	if handle == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: handle, Valid: true}
}
