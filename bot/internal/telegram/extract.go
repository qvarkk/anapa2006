package telegram

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot/models"
)

func extractIdentity(update *models.Update) (userID int64, chatID int64, ok bool) {
	switch {
	case update.Message != nil && update.Message.From != nil:
		return update.Message.From.ID, update.Message.Chat.ID, true
	case update.CallbackQuery != nil:
		msg := update.CallbackQuery.Message.Message
		if msg == nil {
			slog.LogAttrs(
				context.Background(),
				slog.LevelWarn,
				"callback query with inaccessible message",
				slog.Int64("update_id", update.ID),
				slog.Int64("callback_from", update.CallbackQuery.From.ID),
				slog.String("callback_data", update.CallbackQuery.Data),
			)
			return 0, 0, false
		}
		return update.CallbackQuery.From.ID, msg.Chat.ID, true
	default:
		slog.LogAttrs(
			context.Background(),
			slog.LevelDebug,
			"update did not match any known identity shape",
			slog.Int64("update_id", update.ID),
			slog.Bool("has_message", update.Message != nil),
			slog.Bool("has_callback_query", update.CallbackQuery != nil),
			slog.Bool("has_edited_message", update.EditedMessage != nil),
			slog.Bool("has_channel_post", update.ChannelPost != nil),
			slog.Bool("has_my_chat_member", update.MyChatMember != nil),
		)
		return 0, 0, false
	}
}
