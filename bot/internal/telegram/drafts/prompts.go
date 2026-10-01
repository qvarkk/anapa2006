package drafts

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/keyboard"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type genericPromptPayload struct {
	MessageText string

	ProceedText     string
	ProceedCallback string

	ReturnText     string
	ReturnCallback string
}

func HandleEditTextPrompt(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 4, 1)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"parse back navagation",
				slog.String("error", err.Error()),
			)
			return
		}
		postID := data[0]

		draft, err := st.GetPostDraftByID(ctx, postID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get draft by id",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}

		chatID, _ := extract.CallbackTarget(update)
		lang := extract.Lang(ctx)

		sent, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      i18n.T(lang, i18n.EditPrompt, draft.FinalText),
			ParseMode: models.ParseModeHTML,
			ReplyMarkup: &models.ForceReply{
				ForceReply:            true,
				Selective:             true,
				InputFieldPlaceholder: i18n.T(lang, i18n.EditPlaceholder),
			},
		})
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"send edit prompt",
				slog.Int64("draft_id", draft.PostID),
				slog.String("error", err.Error()),
			)
			return
		}

		if err := st.CreatePendingReply(ctx, db.CreatePendingReplyParams{
			ChatID:          chatID,
			PromptMessageID: int64(sent.ID),
			Action:          "awaiting_text",
			PostID:          draft.PostID,
			Origin:          origin,
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"create pending reply",
				slog.Int64("draft_id", draft.PostID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func HandleQueuePrompt(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 3, 1)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"parse back navagation",
				slog.String("error", err.Error()),
			)
			return
		}
		postID := data[0]

		draft, err := st.GetPostDraftByID(ctx, postID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get draft by id",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)

		kb := keyboard.ScheduleKeyboard(draft.PostID, origin, lang)
		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID:   msgID,
			ChatID:      chatID,
			Text:        i18n.T(lang, i18n.PromptDraftQueue),
			ReplyMarkup: kb,
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to draft timing failed",
				slog.Int64("post_id", draft.PostID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func HandleDequeuePrompt(ctx context.Context, b *bot.Bot, update *models.Update) {
	lang := extract.Lang(ctx)

	data, origin, err := extract.BackNavigation(update, 3, 1)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"parse back navagation",
			slog.String("error", err.Error()),
		)
		return
	}
	postID := data[0]

	payload := genericPromptPayload{
		MessageText:     i18n.T(lang, i18n.PromptDraftEditText),
		ProceedText:     i18n.T(lang, i18n.BtnPromptDraftDequeue),
		ProceedCallback: callback.Format(callback.DraftDequeue, postID, origin),
		ReturnText:      i18n.T(lang, i18n.BtnBack),
		ReturnCallback:  origin,
	}

	handleGenericPrompt(ctx, b, update, payload)
}

func HandleDeletePrompt(ctx context.Context, b *bot.Bot, update *models.Update) {
	lang := extract.Lang(ctx)

	data, origin, err := extract.BackNavigation(update, 3, 1)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"parse back navagation",
			slog.String("error", err.Error()),
		)
		return
	}
	postID := data[0]

	payload := genericPromptPayload{
		MessageText:     i18n.T(lang, i18n.PromptDraftDelete),
		ProceedText:     i18n.T(lang, i18n.BtnPromptDraftDelete),
		ProceedCallback: callback.Format(callback.DraftDelete, postID, origin),
		ReturnText:      i18n.T(lang, i18n.BtnBack),
		ReturnCallback:  origin,
	}

	handleGenericPrompt(ctx, b, update, payload)
}

func handleGenericPrompt(ctx context.Context, b *bot.Bot, update *models.Update, payload genericPromptPayload) {
	callback.Ack(ctx, b, update)
	chatID, msgID := extract.CallbackTarget(update)

	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID: msgID,
		ChatID:    chatID,
		Text:      payload.MessageText,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{
						Text:         payload.ProceedText,
						CallbackData: payload.ProceedCallback,
					},
				},
				{
					{
						Text:         payload.ReturnText,
						CallbackData: payload.ReturnCallback,
					},
				},
			},
		},
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to drafts prompt failed",
			slog.String("error", err.Error()),
		)
	}
}
