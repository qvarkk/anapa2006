package drafts

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func HandleDraftsMenu(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback.Ack(ctx, b, update)

	lang := extract.Lang(ctx)
	chatID, msgID := extract.CallbackTarget(update)

	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID: msgID, ChatID: chatID, Text: i18n.T(lang, i18n.DraftsMenu),
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: i18n.T(lang, i18n.BtnDraftsList), CallbackData: callback.Format(callback.DraftsList, 0)}},
				{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: callback.Start}},
			},
		},
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to drafts menu failed",
			slog.String("error", err.Error()),
		)
	}
}
