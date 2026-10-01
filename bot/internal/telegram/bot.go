package telegram

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/commands"
	"qq/anapa2006/internal/telegram/def"
	"qq/anapa2006/internal/telegram/drafts"
	"qq/anapa2006/internal/telegram/fetched"
	"qq/anapa2006/internal/telegram/middleware"
	"qq/anapa2006/internal/telegram/noop"
	"qq/anapa2006/internal/telegram/pagination"
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
	registerExactCallback(b, callback.Fetched, fetched.HandleFetched)
	registerPrefixCallback(b, callback.FetchedChannelsPrefix, fetched.HandleFetchedChannels(st))
	registerPrefixCallback(b, callback.FetchedChannelPostsPrefix, fetched.HandleFetchedChannelPosts(st))
	registerPrefixCallback(b, callback.FetchedLatestPrefix, fetched.HandleFetchedLatest(st))
	registerPrefixCallback(b, callback.FetchedPostPrefix, fetched.HandlePostDetail(st))

	// Queue
	registerExactCallback(b, callback.Queued, queue.HandleQueued)
	registerPrefixCallback(b, callback.QueueListPrefix, queue.HandleQueueList(st))
	registerPrefixCallback(b, callback.QueueSchedulePrefix, queue.HandleScheduleDetail(st))

	// Draft TODO: add media edit
	registerExactCallback(b, callback.DraftsMenu, drafts.HandleDraftsMenu)
	registerPrefixCallback(b, callback.DraftsListPrefix, drafts.HandleList(st))
	registerPrefixCallback(b, callback.DraftDetailPrefix, drafts.HandleDetail(st))
	registerPrefixCallback(b, callback.DraftCreatePrefix, drafts.HandleCreate(st))
	registerPrefixCallback(b, callback.DraftQueuePrefix, drafts.HandleQueue(st, channelID))
	registerPrefixCallback(b, callback.DraftDequeuePrefix, drafts.HandleDequeue(st))
	registerPrefixCallback(b, callback.DraftDeletePrefix, drafts.HandleDelete(st))
	registerPrefixCallback(b, callback.DraftShowPrefix, drafts.HandleShow(st))

	// Draft prompts TODO: add media edit
	registerPrefixCallback(b, callback.DraftPromptDeletePrefix, drafts.HandleDeletePrompt)
	registerPrefixCallback(b, callback.DraftPromptDequeuePrefix, drafts.HandleDequeuePrompt)
	registerPrefixCallback(b, callback.DraftPromptEditTextPrefix, drafts.HandleEditTextPrompt(st))
	registerPrefixCallback(b, callback.DraftPromptQueuePrefix, drafts.HandleQueuePrompt(st))
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
