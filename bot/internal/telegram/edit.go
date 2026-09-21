package telegram

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func isReplyToBot(update *models.Update) bool {
	return update.Message != nil && update.Message.ReplyToMessage != nil
}

func handlePendingReply(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		lang := langFromContext(ctx)
		chatID := update.Message.Chat.ID
		promptID := int64(update.Message.ReplyToMessage.ID)

		pending, err := st.GetPendingReply(ctx, db.GetPendingReplyParams{
			ChatID: chatID, PromptMessageID: promptID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return // reply to some other message
		}
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get pending reply",
				slog.String("error", err.Error()),
			)
			return
		}

		switch pending.Action {
		case "awaiting_text":
			if err := st.UpdateDraftText(ctx, db.UpdateDraftTextParams{
				ID: pending.DraftID, FinalText: update.Message.Text,
			}); err != nil {
				slog.LogAttrs(
					ctx, slog.LevelError,
					"update draft text",
					slog.Int64("draft_id", pending.DraftID),
					slog.String("error", err.Error()),
				)
				return
			}
			if err := st.DeletePendingReply(ctx, db.DeletePendingReplyParams{
				ChatID: chatID, PromptMessageID: promptID,
			}); err != nil {
				slog.LogAttrs(
					ctx, slog.LevelWarn,
					"delete pending reply",
					slog.Int64("chat_id", chatID),
					slog.Int64("prompt_id", promptID),
					slog.String("error", err.Error()),
				)
			}

			kb := scheduleKeyboard(pending.DraftID, pending.Origin, lang)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatID,
				Text:        i18n.T(lang, i18n.SchedulePrompt, update.Message.Text),
				ReplyMarkup: kb, ParseMode: models.ParseModeHTML,
				LinkPreviewOptions: &models.LinkPreviewOptions{IsDisabled: bot.True()},
			})

		case "awaiting_schedule":
			// TODO:
			return
		}
	}
}
