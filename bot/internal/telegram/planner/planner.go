package planner

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func HandlePlannerSkip(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)
		lang := extract.Lang(ctx)

		data, origin, err := extract.BackNavigation(update, 2, 1)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"parse back navagation",
				slog.String("error", err.Error()),
			)
			return
		}
		postID := data[0]

		if err := st.HidePost(ctx, postID); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"hide post",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}

		chatID, msgID := extract.CallbackTarget(update)

		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID,
			Text: i18n.T(lang, i18n.ScheduleSkipped),
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: origin}},
				}},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to schedule skip failed",
				slog.Int("message_id", msgID),
				slog.Int64("chat_id", chatID),
				slog.String("error", err.Error()),
			)
			return
		}
	}
}
