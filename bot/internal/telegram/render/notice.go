package render

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Notice sends a standalone message with a single way back.
func Notice(ctx context.Context, b *bot.Bot, chatID int64, text, backText, backCallback string) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: backKeyboard(backText, backCallback),
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"send notice failed",
			slog.String("error", err.Error()),
		)
	}
}

// EditNotice turns the message the callback came from into a notice with a
// single way back.
func EditNotice(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	messageID int,
	text, backText, backCallback string,
) {
	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ReplyMarkup: backKeyboard(backText, backCallback),
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"edit message to notice failed",
			slog.String("error", err.Error()),
		)
	}
}

func backKeyboard(backText, backCallback string) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: backText, CallbackData: backCallback}},
		},
	}
}
