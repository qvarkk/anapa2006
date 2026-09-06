package telegram

import (
	"context"
	"database/sql"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func requireAllowed(st *store.Store) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			userID, chatID, ok := extractIdentity(update)
			if !ok {
				return
			}

			user, err := st.GetAllowedUser(ctx, userID)
			if err != nil {
				lang := i18n.DefaultLang

				if err != sql.ErrNoRows {
					slog.LogAttrs(
						ctx, slog.LevelError,
						"middleware: DB error",
						slog.String("error", err.Error()),
						slog.Int64("userID", userID),
					)
				}

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   i18n.T(lang, i18n.KeyNoAccess),
				})
				return
			}

			ctx = context.WithValue(ctx, ctxKeyLang, user.Lang)
			next(ctx, b, update)
		}
	}
}
