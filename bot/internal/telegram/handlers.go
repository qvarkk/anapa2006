package telegram

import (
	"context"
	"database/sql"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type HandlerStore struct {
	store *store.Store
}

func NewHandlerStore(store *store.Store) *HandlerStore {
	return &HandlerStore{
		store: store,
	}
}

func (h *HandlerStore) DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}

func (h *HandlerStore) GiveAdminHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	userID, chatID, ok := extractIdentity(update)
	if !ok {
		return
	}

	username := update.Message.From.Username
	firstname := update.Message.From.FirstName
	h.store.AddAllowedUser(ctx, db.AddAllowedUserParams{
		UserID: userID,
		Username: sql.NullString{
			String: username,
			Valid:  true,
		},
	})

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   i18n.T(LangFromContext(ctx), i18n.KeyStart, firstname),
	})
}
