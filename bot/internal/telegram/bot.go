package telegram

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func New(ctx context.Context, token string, st *store.Store) (*bot.Bot, error) {
	b, err := bot.New(token,
		bot.WithMiddlewares(requireAllowed(st)),
		bot.WithDefaultHandler(handleDefault),
	)
	if err != nil {
		return nil, err
	}

	registerHandlers(b)

	if err := setCommandsMenu(ctx, b); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"failed to set command menu",
			slog.String("error", err.Error()),
		)
	}

	return b, nil
}

func registerHandlers(b *bot.Bot) {
	b.RegisterHandler(bot.HandlerTypeMessageText, string(commandStart), bot.MatchTypeExact, handleStartCommand)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, string(callbackOpenMenu), bot.MatchTypeExact, handleOpenMenuCallback)
}

func setCommandsMenu(ctx context.Context, b *bot.Bot) error {
	_, err := b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: string(commandStart), Description: i18n.T(i18n.DefaultLang, i18n.KeyCommandStart)},
		},
	})
	return err
}
