package pagination

import (
	"context"
	"fmt"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func HandleFirstPage(ctx context.Context, b *bot.Bot, update *models.Update) {
	HandleIncorrectPage(ctx, b, update, i18n.FirstPage)
}

func HandleLastPage(ctx context.Context, b *bot.Bot, update *models.Update) {
	HandleIncorrectPage(ctx, b, update, i18n.LastPage)
}

func HandleIncorrectPage(ctx context.Context, b *bot.Bot, update *models.Update, message i18n.Key) {
	lang := extract.Lang(ctx)
	if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            i18n.T(lang, message),
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"answer incorrect page callback failed",
			slog.String("error", err.Error()),
		)
	}
}

func TotalPages(total, pageSize int) int {
	if total == 0 {
		return 1
	}
	return (total + pageSize - 1) / pageSize
}

func ClampPage(page, totalPages int) int {
	if page < 0 {
		return 0
	}
	if page >= totalPages {
		return totalPages - 1
	}
	return page
}

func BuildPaginationRow(page, totalPages int, prevCallback, nextCallback string) []models.InlineKeyboardButton {
	return []models.InlineKeyboardButton{
		{Text: "⏪", CallbackData: prevCallback},
		{Text: fmt.Sprintf("%d/%d", page+1, totalPages), CallbackData: callback.Noop},
		{Text: "⏩", CallbackData: nextCallback},
	}
}
