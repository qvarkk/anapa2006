package reply

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/extract"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type operationMessagePayload struct {
	ChatID      int64
	MessageText string

	ReturnText     string
	ReturnCallback string
}

func IsReplyToBot(update *models.Update) bool {
	return update.Message != nil && update.Message.ReplyToMessage != nil
}

// Draft edit text and custom schedule
func HandlePendingReply(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		lang := extract.Lang(ctx)
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
			formatted := entitiesToHTML(update.Message.Text, update.Message.Entities)

			if err := st.UpdatePostDraftText(ctx, db.UpdatePostDraftTextParams{
				PostID: pending.PostID, FinalText: formatted,
			}); err != nil {
				slog.LogAttrs(
					ctx, slog.LevelError,
					"update draft text",
					slog.Int64("post_id", pending.PostID),
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

			payload := operationMessagePayload{
				ChatID:         chatID,
				MessageText:    i18n.T(lang, i18n.OperationSuccess),
				ReturnText:     i18n.T(lang, i18n.BtnBack),
				ReturnCallback: pending.Origin,
			}

			sendOperationSuccessMessage(ctx, b, payload)

		case "awaiting_schedule":
			// TODO:
			return
		}
	}
}

func sendOperationSuccessMessage(ctx context.Context, b *bot.Bot, payload operationMessagePayload) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: payload.ChatID,
		Text:   payload.MessageText,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{
						Text:         payload.ReturnText,
						CallbackData: payload.ReturnCallback,
					},
				},
			},
		},
	})
}
