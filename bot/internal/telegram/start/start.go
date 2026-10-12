package start

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func startMenuKeyboard(lang i18n.Lang, owner bool) *models.InlineKeyboardMarkup {
	rows := [][]models.InlineKeyboardButton{
		{{Text: i18n.T(lang, i18n.BtnNewPosts), CallbackData: callback.Fetched}},
		{{Text: i18n.T(lang, i18n.BtnDraftsMenu), CallbackData: callback.DraftsMenu}},
		{{Text: i18n.T(lang, i18n.BtnManageChannels), CallbackData: callback.ChannelsMenu}},
	}

	// Only the owner may manage access; the handlers check this as well.
	if owner {
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: i18n.T(lang, i18n.BtnManageAccess), CallbackData: callback.AccessMenu},
		})
	}

	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func HandleCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	lang := extract.Lang(ctx)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        i18n.T(lang, i18n.Start),
		ReplyMarkup: startMenuKeyboard(lang, extract.IsOwner(ctx)),
	})
}

func HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback.Ack(ctx, b, update)

	lang := extract.Lang(ctx)
	chatID, msgID := extract.CallbackTarget(update)

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   msgID,
		Text:        i18n.T(lang, i18n.Start),
		ReplyMarkup: startMenuKeyboard(lang, extract.IsOwner(ctx)),
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
