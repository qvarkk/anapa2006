package telegram

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

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
