package middleware

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/commands"
	"qq/anapa2006/internal/telegram/extract"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// RequireAllowed drops updates from users that are not in allowed_users.
// The configured owner is the single exception: they may run /bootstrap before
// they exist in the database, which is the only way to seed the first user.
func RequireAllowed(st *store.Store, ownerID int64) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			userID, chatID, ok := extract.Identity(update)
			if !ok {
				return
			}

			if ownerID != 0 && userID == ownerID && isBootstrap(update) {
				next(extract.WithOwner(ctx, true), b, update)
				return
			}

			user, err := st.GetAllowedUser(ctx, userID)
			if err != nil {
				lang := i18n.DefaultLang

				if !errors.Is(err, sql.ErrNoRows) {
					slog.LogAttrs(
						ctx, slog.LevelError,
						"middleware: DB error",
						slog.String("error", err.Error()),
						slog.Int64("userID", userID),
					)
				}

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   i18n.T(lang, i18n.NoAccess),
				})
				return
			}

			ctx = extract.WithOwner(ctx, ownerID != 0 && userID == ownerID)
			ctx = extract.WithLang(ctx, user.Lang)
			next(ctx, b, update)
		}
	}
}

func isBootstrap(update *models.Update) bool {
	return update.Message != nil && commands.IsBootstrap(update.Message.Text)
}
