package reply

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/domain"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/moscow"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/extract"
	"strings"
	"time"

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
func HandlePendingReply(st *store.Store, channelID int64) bot.HandlerFunc {
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
		case domain.PendingActionAwaitingText:
			handleTextReply(ctx, b, st, pending, update, chatID, promptID, lang)

		case domain.PendingActionAwaitingSchedule:
			handleScheduleReply(ctx, b, st, channelID, pending, update, chatID, promptID, lang)
		}
	}
}

func handleTextReply(
	ctx context.Context,
	b *bot.Bot,
	st *store.Store,
	pending db.PendingReply,
	update *models.Update,
	chatID int64,
	promptID int64,
	lang i18n.Lang,
) {
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
	deletePendingReply(ctx, st, chatID, promptID)

	payload := operationMessagePayload{
		ChatID:         chatID,
		MessageText:    i18n.T(lang, i18n.OperationSuccess),
		ReturnText:     i18n.T(lang, i18n.BtnBack),
		ReturnCallback: pending.Origin,
	}

	sendOperationSuccessMessage(ctx, b, payload)
}

// handleScheduleReply parses a Moscow date/time from the user's reply and
// schedules the draft. Invalid input answers with a retry-or-back prompt and
// keeps the pending reply, so replying to the original prompt again works.
func handleScheduleReply(
	ctx context.Context,
	b *bot.Bot,
	st *store.Store,
	channelID int64,
	pending db.PendingReply,
	update *models.Update,
	chatID int64,
	promptID int64,
	lang i18n.Lang,
) {
	input := strings.TrimSpace(update.Message.Text)

	scheduledAt, err := moscow.Parse(input, time.Now())
	if err != nil {
		sendScheduleInputError(ctx, b, chatID, pending.Origin, lang, i18n.ScheduleCustomInvalid)
		return
	}
	if !scheduledAt.After(time.Now()) {
		sendScheduleInputError(ctx, b, chatID, pending.Origin, lang, i18n.ScheduleCustomPast)
		return
	}

	if err := st.Deschedule(ctx, pending.PostID); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"deschedule draft before rescheduling",
			slog.Int64("post_id", pending.PostID),
			slog.String("error", err.Error()),
		)
	}

	if err := st.CreateSchedule(ctx, db.CreateScheduleParams{
		PostID: pending.PostID, TargetChatID: channelID,
		// The driver stores timestamps with their own offset, and the scheduler
		// compares them as text against time.Now(), so keep this in the process
		// timezone like every other scheduled_at value.
		ScheduledAt: scheduledAt.In(time.Local),
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"create custom schedule",
			slog.Int64("post_id", pending.PostID),
			slog.String("error", err.Error()),
		)
		return
	}

	if err := st.MarkPostScheduled(ctx, pending.PostID); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"mark post scheduled",
			slog.Int64("post_id", pending.PostID),
			slog.String("error", err.Error()),
		)
	}

	deletePendingReply(ctx, st, chatID, promptID)

	slog.LogAttrs(
		ctx, slog.LevelInfo,
		"post was scheduled with custom time",
		slog.Int64("post_id", pending.PostID),
		slog.String("scheduled_at", scheduledAt.Format(time.RFC3339)),
	)

	sendOperationSuccessMessage(ctx, b, operationMessagePayload{
		ChatID:         chatID,
		MessageText:    i18n.T(lang, i18n.ScheduledAt, moscow.Format(scheduledAt)),
		ReturnText:     i18n.T(lang, i18n.BtnBack),
		ReturnCallback: pending.Origin,
	})
}

func sendScheduleInputError(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	origin string,
	lang i18n.Lang,
	key i18n.Key,
) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   i18n.T(lang, key),
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{
						Text:         i18n.T(lang, i18n.BtnBack),
						CallbackData: origin,
					},
				},
			},
		},
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"send schedule input error",
			slog.String("error", err.Error()),
		)
	}
}

func deletePendingReply(ctx context.Context, st *store.Store, chatID, promptID int64) {
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
