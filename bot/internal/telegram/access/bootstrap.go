package access

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

// HandleBootstrap adds the configured owner to allowed_users. It is the only
// command that works before the owner has access.
func HandleBootstrap(st *store.Store, ownerID int64) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		lang := extract.Lang(ctx)
		chatID := update.Message.Chat.ID

		if ownerID == 0 {
			sendMessage(ctx, b, chatID, i18n.T(lang, i18n.BootstrapNoOwner))
			return
		}

		userID, _, ok := extract.Identity(update)
		if !ok || userID != ownerID {
			sendMessage(ctx, b, chatID, i18n.T(lang, i18n.OwnerOnly))
			return
		}

		_, err := st.GetAllowedUser(ctx, ownerID)
		switch {
		case err == nil:
			sendMessage(ctx, b, chatID, i18n.T(lang, i18n.BootstrapAlready))
			return
		case !errors.Is(err, sql.ErrNoRows):
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get owner",
				slog.Int64("user_id", ownerID),
				slog.String("error", err.Error()),
			)
			return
		}

		var username sql.NullString
		if update.Message.From != nil && update.Message.From.Username != "" {
			username = sql.NullString{String: update.Message.From.Username, Valid: true}
		}

		if err := st.AddAllowedUser(ctx, db.AddAllowedUserParams{
			UserID: ownerID, Username: username,
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"add owner to allowed users",
				slog.Int64("user_id", ownerID),
				slog.String("error", err.Error()),
			)
			return
		}

		slog.LogAttrs(ctx, slog.LevelInfo, "owner bootstrapped", slog.Int64("user_id", ownerID))
		sendMessage(ctx, b, chatID, i18n.T(lang, i18n.BootstrapSuccess))
	}
}
