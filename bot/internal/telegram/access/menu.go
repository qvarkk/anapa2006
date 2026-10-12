package access

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/commands"
	"qq/anapa2006/internal/telegram/extract"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// IsBootstrapCommand matches the /bootstrap command, which is the only thing a
// not-yet-allowed owner may do.
func IsBootstrapCommand(update *models.Update) bool {
	return update.Message != nil && commands.IsBootstrap(update.Message.Text)
}

func HandleMenu(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !requireOwner(ctx, b, update) {
		return
	}
	callback.Ack(ctx, b, update)

	lang := extract.Lang(ctx)
	chatID, msgID := extract.CallbackTarget(update)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: i18n.T(lang, i18n.BtnAccessList), CallbackData: callback.Format(callback.AccessList, 0)}},
		{{Text: i18n.T(lang, i18n.BtnUserAdd), CallbackData: callback.Format(callback.AccessAdd, callback.AccessMenu)}},
		{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: callback.Start}},
	}}

	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID,
		Text: i18n.T(lang, i18n.AccessMenu), ReplyMarkup: kb,
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to access menu failed",
			slog.String("error", err.Error()),
		)
	}
}

// requireOwner reports whether the acting user is the owner. When they are not,
// the callback is answered with an alert and the caller must stop.
func requireOwner(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if extract.IsOwner(ctx) {
		return true
	}

	lang := extract.Lang(ctx)
	if update.CallbackQuery != nil {
		if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            i18n.T(lang, i18n.OwnerOnly),
			ShowAlert:       true,
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"answer owner-only callback failed",
				slog.String("error", err.Error()),
			)
		}
	}

	return false
}

func sendMessage(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"send access message failed",
			slog.String("error", err.Error()),
		)
	}
}
