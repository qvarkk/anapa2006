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
			{{Text: i18n.T(lang, i18n.KeyBtnNewPosts), CallbackData: string(callbackListNew)}},
			{{Text: i18n.T(lang, i18n.KeyBtnScheduled), CallbackData: string(callbackListScheduled)}},
		},
	}
}

func handleStartCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	lang := LangFromContext(ctx)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        i18n.T(lang, i18n.KeyStart),
		ReplyMarkup: startMenuKeyboard(lang),
	})
}

func handleOpenMenuCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"answer callback query failed",
			slog.String("error", err.Error()),
		)
	}

	lang := LangFromContext(ctx)
	msg := update.CallbackQuery.Message.Message

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        i18n.T(lang, i18n.KeyStart),
		ReplyMarkup: startMenuKeyboard(lang),
	})
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to start menu failed",
			slog.Int64("chat_id", msg.Chat.ID),
			slog.Int("message_id", msg.ID),
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

	lang := LangFromContext(ctx)
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

func chatIDFromUpdate(update *models.Update) (int64, bool) {
	if update.Message != nil {
		return update.Message.Chat.ID, true
	}
	if update.CallbackQuery != nil && update.CallbackQuery.Message.Message != nil {
		return update.CallbackQuery.Message.Message.Chat.ID, true
	}
	return 0, false
}
