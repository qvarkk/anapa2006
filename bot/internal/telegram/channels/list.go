package channels

import (
	"context"
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

func HandleList(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		page := callback.ParseIntPart(update.CallbackQuery.Data, 2)
		renderList(ctx, b, st, update, page)
	}
}

// renderList shows the channel page, editing the message the callback came from.
func renderList(ctx context.Context, b *bot.Bot, st *store.Store, update *models.Update, page int) {
	lang := extract.Lang(ctx)

	total, err := st.CountSources(ctx)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"count sources",
			slog.String("error", err.Error()),
		)
		return
	}

	totalPages := pagination.TotalPages(int(total), RecordsPerPage)
	page = pagination.ClampPage(page, totalPages)

	sources, err := st.ListSources(ctx, db.ListSourcesParams{
		Limit: RecordsPerPage, Offset: int64(page * RecordsPerPage),
	})
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"list sources",
			slog.String("error", err.Error()),
		)
		return
	}

	listCallback := callback.Format(callback.ChannelsList, page)

	var sb strings.Builder
	sb.WriteString(i18n.T(lang, i18n.ChannelsListHeader, total))
	sb.WriteString("\n\n")

	rows := make([][]models.InlineKeyboardButton, 0, len(sources)+3)
	for _, source := range sources {
		status, mark := i18n.T(lang, i18n.ChannelStatusOff), "⛔️"
		if source.Active == 1 {
			status, mark = i18n.T(lang, i18n.ChannelStatusActive), "✅"
		}

		sb.WriteString(i18n.T(lang, i18n.ChannelsListEntry, source.ID, source.ChannelHandle, status))
		sb.WriteString("\n\n")

		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("%s @%s", mark, source.ChannelHandle),
				CallbackData: callback.Format(callback.ChannelsToggle, source.ID, page),
			},
			{
				Text:         i18n.T(lang, i18n.BtnDelete),
				CallbackData: callback.Format(callback.ChannelsDeletePrompt, source.ID, listCallback),
			},
		})
	}

	if totalPages > 1 {
		prev, next := callback.FirstPage, callback.LastPage
		if page > 0 {
			prev = callback.Format(callback.ChannelsList, page-1)
		}
		if page < totalPages-1 {
			next = callback.Format(callback.ChannelsList, page+1)
		}
		rows = append(rows, pagination.BuildPaginationRow(page, totalPages, prev, next))
	}

	rows = append(rows, []models.InlineKeyboardButton{
		{Text: i18n.T(lang, i18n.BtnChannelAdd), CallbackData: callback.Format(callback.ChannelsAdd, listCallback)},
	})
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: i18n.T(lang, i18n.BtnBack), CallbackData: callback.ChannelsMenu},
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
			"edit message to channels list failed",
			slog.String("error", err.Error()),
		)
	}
}
