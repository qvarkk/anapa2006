package telegram

import (
	"context"
	"database/sql"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type MiddlewareStore struct {
	store *store.Store
}

func NewMiddlewareStore(store *store.Store) *MiddlewareStore {
	return &MiddlewareStore{
		store: store,
	}
}

func (m *MiddlewareStore) RequireAllowedMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		// TODO: remove!
		if update.Message != nil && strings.HasPrefix(update.Message.Text, "/tmp") {
			next(ctx, b, update)
			return
		}

		userID, chatID, ok := extractIdentity(update)
		if !ok {
			return
		}

		user, err := m.store.GetAllowedUser(ctx, userID)
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
