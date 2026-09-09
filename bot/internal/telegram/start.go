package telegram

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func startMenuKeyboard(lang i18n.Lang) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: i18n.T(lang, i18n.KeyBtnNewPosts), CallbackData: string(callbackListFetched)}},
			{{Text: i18n.T(lang, i18n.KeyBtnScheduled), CallbackData: string(callbackListScheduled)}},
		},
	}
}

func handleStartCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	lang := langFromContext(ctx)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        i18n.T(lang, i18n.KeyStart),
		ReplyMarkup: startMenuKeyboard(lang),
	})
}

func handleOpenMenuCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	ackCallback(ctx, b, update)

	lang := langFromContext(ctx)
	chatID, msgID := callbackTarget(update)

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   msgID,
		Text:        i18n.T(lang, i18n.KeyStart),
		ReplyMarkup: startMenuKeyboard(lang),
	})
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to start menu failed",
			slog.Int64("chat_id", chatID),
			slog.Int("message_id", msgID),
			slog.String("error", err.Error()),
		)
	}
}

func handleDefault(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID, ok := chatIDFromUpdate(update)
	if !ok {
		slog.LogAttrs(
			ctx, slog.LevelDebug,
			"default handler got update with no resolvable chat",
			slog.Int64("update_id", update.ID),
		)
		return
	}

	lang := langFromContext(ctx)
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: i18n.T(lang, i18n.KeyBtnOpenMenu), CallbackData: string(callbackOpenMenu)}},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        i18n.T(lang, i18n.KeyUnrecognized),
		ReplyMarkup: kb,
	})
}

func handleNoop(ctx context.Context, b *bot.Bot, update *models.Update) {
	lang := langFromContext(ctx)
	if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            i18n.T(lang, i18n.KeyFeatureUnavailable),
		ShowAlert:       true,
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"answer noop callback failed",
			slog.String("error", err.Error()),
		)
	}
}
