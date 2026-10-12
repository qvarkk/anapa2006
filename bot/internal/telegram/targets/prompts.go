package targets

import (
	"context"
	"database/sql"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/domain"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// HandleAddPrompt asks for a target channel handle and remembers the answer as
// a pending reply.
func HandleAddPrompt(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		_, origin, err := extract.BackNavigation(update, 2, 0)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"parse back navagation",
				slog.String("error", err.Error()),
			)
			return
		}

		lang := extract.Lang(ctx)
		chatID, _ := extract.CallbackTarget(update)

		sent, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   i18n.T(lang, i18n.TargetAddPrompt),
			ReplyMarkup: &models.ForceReply{
				ForceReply:            true,
				Selective:             true,
				InputFieldPlaceholder: i18n.T(lang, i18n.TargetAddHolder),
			},
		})
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"send target add prompt",
				slog.String("error", err.Error()),
			)
			return
		}

		if err := st.CreatePendingReply(ctx, db.CreatePendingReplyParams{
			ChatID:          chatID,
			PromptMessageID: int64(sent.ID),
			Action:          domain.PendingActionAwaitingTargetHandle,
			PostID:          sql.NullInt64{},
			Origin:          origin,
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"create pending reply",
				slog.String("error", err.Error()),
			)
		}
	}
}
