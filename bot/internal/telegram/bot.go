package telegram

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/commands"
	"qq/anapa2006/internal/telegram/def"
	"qq/anapa2006/internal/telegram/fetched"
	"qq/anapa2006/internal/telegram/middleware"
	"qq/anapa2006/internal/telegram/noop"
	"qq/anapa2006/internal/telegram/pagination"
	"qq/anapa2006/internal/telegram/planner"
	"qq/anapa2006/internal/telegram/post"
	"qq/anapa2006/internal/telegram/queue"
	"qq/anapa2006/internal/telegram/reply"
	"qq/anapa2006/internal/telegram/start"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Config struct {
	Token     string
	ChannelID int64
}

func New(ctx context.Context, cfg Config, st *store.Store) (*bot.Bot, error) {
	b, err := bot.New(cfg.Token,
		bot.WithMiddlewares(middleware.RequireAllowed(st)),
		bot.WithDefaultHandler(def.Handle),
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
	b.RegisterHandlerMatchFunc(reply.IsReplyToBot, reply.HandlePendingReply(st))

	// Start
	b.RegisterHandler(bot.HandlerTypeMessageText, commands.Start, bot.MatchTypeExact, start.HandleCommand)
	registerExactCallback(b, callback.Start, start.HandleCallback)

	// Common
	registerExactCallback(b, callback.Noop, noop.Handle)
	registerExactCallback(b, callback.FirstPage, pagination.HandleFirstPage)
	registerExactCallback(b, callback.LastPage, pagination.HandleLastPage)

	// Fetch
	registerExactCallback(b, callback.Fetch, fetched.HandleFetch)
	registerPrefixCallback(b, callback.FetchChannelsMatch, fetched.HandleFetchChannels(st))
	registerPrefixCallback(b, callback.FetchChannelPostsMatch, fetched.HandleFetchChannelPosts(st))
	registerPrefixCallback(b, callback.FetchLatestMatch, fetched.HandleFetchLatest(st))
	registerPrefixCallback(b, callback.FetchPostMatch, post.HandlePostDetail(st))

	// Queue
	registerExactCallback(b, callback.Scheduled, queue.HandleScheduled)
	registerPrefixCallback(b, callback.ScheduledListMatch, queue.HandleScheduledList(st))

	// Planner
	registerPrefixCallback(b, callback.ScheduleUseMatch, planner.HandleScheduleUse(st))
	registerPrefixCallback(b, callback.ScheduleEditMatch, planner.HandleScheduleEdit(st))
	registerPrefixCallback(b, callback.ScheduleSkipMatch, planner.HandleScheduleSkip(st))
	registerPrefixCallback(b, callback.ScheduleCreateMatch, planner.HandleScheduleCreate(st, channelID))
}

func registerExactCallback(b *bot.Bot, callback string, f bot.HandlerFunc) {
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, callback, bot.MatchTypeExact, f)
}

func registerPrefixCallback(b *bot.Bot, prefix string, f bot.HandlerFunc) {
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, prefix, bot.MatchTypePrefix, f)
}

func setCommandsMenu(ctx context.Context, b *bot.Bot) error {
	// TODO: localize for multiple languages (at least RU, EN)
	_, err := b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: string(commands.Start), Description: i18n.T(i18n.DefaultLang, i18n.CommandStart)},
		},
	})
	return err
}
