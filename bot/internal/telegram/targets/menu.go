package targets

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func HandleMenu(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback.Ack(ctx, b, update)

	lang := extract.Lang(ctx)
	chatID, msgID := extract.CallbackTarget(update)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: i18n.T(lang, i18n.BtnTargetsList), CallbackData: callback.Format(callback.TargetsList, 0)}},
		{{Text: i18n.T(lang, i18n.BtnTargetAdd), CallbackData: callback.Format(callback.TargetsAdd, callback.TargetsMenu)}},
		{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: callback.Start}},
	}}

	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID,
		Text: i18n.T(lang, i18n.TargetsMenu), ReplyMarkup: kb,
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to targets menu failed",
			slog.String("error", err.Error()),
		)
	}
}
