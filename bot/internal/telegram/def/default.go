// default is a reserved keyword so I fallback to something shorter
package def

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID, ok := extract.ChatID(update)
	if !ok {
		slog.LogAttrs(
			ctx, slog.LevelDebug,
			"default handler got update with no resolvable chat",
			slog.Int64("update_id", update.ID),
		)
		return
	}

	lang := extract.Lang(ctx)
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: i18n.T(lang, i18n.BtnOpenMenu), CallbackData: callback.Start}},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        i18n.T(lang, i18n.Unrecognized),
		ReplyMarkup: kb,
	})
}
