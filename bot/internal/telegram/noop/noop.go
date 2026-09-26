package noop

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/telegram/extract"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	lang := extract.Lang(ctx)
	if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            i18n.T(lang, i18n.FeatureUnavailable),
		ShowAlert:       true,
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"answer noop callback failed",
			slog.String("error", err.Error()),
		)
	}
}
