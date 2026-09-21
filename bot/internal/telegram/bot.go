package telegram

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Config struct {
	Token     string
	ChannelID int64
}

func New(ctx context.Context, cfg Config, st *store.Store) (*bot.Bot, error) {
	b, err := bot.New(cfg.Token,
		bot.WithMiddlewares(requireAllowed(st)),
		bot.WithDefaultHandler(handleDefault),
	)
	if err != nil {
		return nil, err
	}

	registerHandlers(b, cfg.ChannelID, st)

	if err := setCommandsMenu(ctx, b); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"failed to set command menu",
			slog.String("error", err.Error()),
		)
	}

	return b, nil
}

func registerHandlers(b *bot.Bot, channelID int64, st *store.Store) {
	// Posts edit match func
	b.RegisterHandlerMatchFunc(isReplyToBot, handlePendingReply(st))

	// Start
	b.RegisterHandler(bot.HandlerTypeMessageText, string(commandStart), bot.MatchTypeExact, handleStartCommand)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, string(menu), bot.MatchTypeExact, handleOpenMenuCallback)

	// Common
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, noop, bot.MatchTypeExact, handleNoop)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, firstPage, bot.MatchTypeExact, handleFirstPage)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, lastPage, bot.MatchTypeExact, handleLastPage)

	// Fetch
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, fetch, bot.MatchTypeExact, handleFetch)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "grp:p:", bot.MatchTypePrefix, handleFetchChannels(st))
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "grp:c:", bot.MatchTypePrefix, handleFetchChannelPosts(st))
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "lat:", bot.MatchTypePrefix, handleFetchLatest(st))
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "post:", bot.MatchTypePrefix, handlePostDetail(st))

	// Schedule
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "sched:u:", bot.MatchTypePrefix, handleScheduleUse(st))
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "sched:e", bot.MatchTypePrefix, handleScheduleEdit(st))
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "sched:s:", bot.MatchTypePrefix, handleScheduleSkip(st))
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "sched:set", bot.MatchTypePrefix, handleScheduleCreate(st, channelID))
}

func setCommandsMenu(ctx context.Context, b *bot.Bot) error {
	// TODO: localize for multiple languages (at least RU, EN)
	_, err := b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: string(commandStart), Description: i18n.T(i18n.DefaultLang, i18n.CommandStart)},
		},
	})
	return err
}
