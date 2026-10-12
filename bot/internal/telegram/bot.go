package telegram

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/access"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/channels"
	"qq/anapa2006/internal/telegram/commands"
	"qq/anapa2006/internal/telegram/def"
	"qq/anapa2006/internal/telegram/drafts"
	"qq/anapa2006/internal/telegram/fetched"
	"qq/anapa2006/internal/telegram/middleware"
	"qq/anapa2006/internal/telegram/noop"
	"qq/anapa2006/internal/telegram/pagination"
	"qq/anapa2006/internal/telegram/render"
	"qq/anapa2006/internal/telegram/reply"
	"qq/anapa2006/internal/telegram/start"
	"qq/anapa2006/internal/telegram/targets"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Config struct {
	Token   string
	OwnerID int64
}

func New(ctx context.Context, cfg Config, st *store.Store) (*bot.Bot, error) {
	b, err := bot.New(cfg.Token,
		bot.WithMiddlewares(middleware.RequireAllowed(st, cfg.OwnerID)),
		bot.WithDefaultHandler(def.Handle),
	)
	if err != nil {
		return nil, err
	}

	registerHandlers(b, cfg, st)

	if err := setCommandsMenu(ctx, b); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"failed to set command menu",
			slog.String("error", err.Error()),
		)
	}

	return b, nil
}

func registerHandlers(b *bot.Bot, cfg Config, st *store.Store) {
	// Single entry point for answers to bot prompts (drafts, schedule, channel
	// handle, user reference) and for forwarded messages.
	b.RegisterHandlerMatchFunc(reply.IsReplyOrForward, reply.HandlePendingReply(st))

	// Bootstrap: the only handler reachable by the owner before they are allowed
	b.RegisterHandlerMatchFunc(access.IsBootstrapCommand, access.HandleBootstrap(st, cfg.OwnerID))

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

	// Post actions
	registerPrefixCallback(b, callback.PostShowPrefix, fetched.HandleShow(st))
	registerPrefixCallback(b, callback.PostHidePrefix, fetched.HandleHide(st))
	registerPrefixCallback(b, callback.PostArchivePrefix, fetched.HandleArchive(st))
	registerPrefixCallback(b, callback.PostUnarchivePrefix, fetched.HandleUnarchive(st))
	registerPrefixCallback(b, callback.PostDeletePrefix, fetched.HandleDelete(st))

	// Drafts
	registerExactCallback(b, callback.DraftsMenu, drafts.HandleDraftsMenu)
	registerPrefixCallback(b, callback.DraftsListActivePrefix, drafts.HandleList(st, render.DraftListActive))
	registerPrefixCallback(b, callback.DraftsListPostedPrefix, drafts.HandleList(st, render.DraftListPosted))
	registerPrefixCallback(b, callback.DraftsListFailedPrefix, drafts.HandleList(st, render.DraftListFailed))
	registerPrefixCallback(b, callback.DraftDetailPrefix, drafts.HandleDetail(st))
	registerPrefixCallback(b, callback.DraftCreatePrefix, drafts.HandleCreate(st))
	registerPrefixCallback(b, callback.DraftQueuePrefix, drafts.HandleQueue(st))
	registerPrefixCallback(b, callback.DraftDequeuePrefix, drafts.HandleDequeue(st))
	registerPrefixCallback(b, callback.DraftDeletePrefix, drafts.HandleDelete(st))
	registerPrefixCallback(b, callback.DraftShowPrefix, drafts.HandleShow(st))

	// Draft target channel selection
	registerPrefixCallback(b, callback.DraftTargetListPrefix, drafts.HandleTargetList(st))
	registerPrefixCallback(b, callback.DraftTargetPickPrefix, drafts.HandleTargetPick(st))
	registerPrefixCallback(b, callback.DraftTargetClearPrefix, drafts.HandleTargetClear(st))

	// Draft prompts
	registerPrefixCallback(b, callback.DraftPromptDeletePrefix, drafts.HandleDeletePrompt)
	registerPrefixCallback(b, callback.DraftPromptDequeuePrefix, drafts.HandleDequeuePrompt)
	registerPrefixCallback(b, callback.DraftPromptEditTextPrefix, drafts.HandleEditTextPrompt(st))
	registerPrefixCallback(b, callback.DraftPromptEditMediaPrefix, drafts.HandleEditMediaPrompt(st))
	registerPrefixCallback(b, callback.DraftPromptQueuePrefix, drafts.HandleQueuePrompt(st))
	registerPrefixCallback(b, callback.DraftPromptCustomPrefix, drafts.HandleCustomSchedulePrompt(st))

	// Draft media editor
	registerPrefixCallback(b, callback.DraftMediaListPrefix, drafts.HandleMediaList(st))
	registerPrefixCallback(b, callback.DraftMediaDetailPrefix, drafts.HandleMediaDetail(st))
	registerPrefixCallback(b, callback.DraftMediaDeletePrefix, drafts.HandleMediaDelete(st))
	registerPrefixCallback(b, callback.DraftMediaSubstitutePromptPrefix, drafts.HandleMediaSubstitutePrompt(st))
	registerPrefixCallback(b, callback.DraftMediaAddPromptPrefix, drafts.HandleMediaAddPrompt(st))

	// Channels
	registerExactCallback(b, callback.ChannelsMenu, channels.HandleMenu)
	registerPrefixCallback(b, callback.ChannelsListPrefix, channels.HandleList(st))
	registerPrefixCallback(b, callback.ChannelsDetailPrefix, channels.HandleDetail(st))
	registerPrefixCallback(b, callback.ChannelsAddPrefix, channels.HandleAddPrompt(st))
	registerPrefixCallback(b, callback.ChannelsTogglePrefix, channels.HandleToggle(st))
	registerPrefixCallback(b, callback.ChannelsDeletePromptPrefix, channels.HandleDeletePrompt(st))
	registerPrefixCallback(b, callback.ChannelsDeletePrefix, channels.HandleDelete(st))

	// Target channels (publishing destinations)
	registerExactCallback(b, callback.TargetsMenu, targets.HandleMenu)
	registerPrefixCallback(b, callback.TargetsListPrefix, targets.HandleList(st))
	registerPrefixCallback(b, callback.TargetsAddPrefix, targets.HandleAddPrompt(st))
	registerPrefixCallback(b, callback.TargetsDeletePromptPrefix, targets.HandleDeletePrompt(st))
	registerPrefixCallback(b, callback.TargetsDeletePrefix, targets.HandleDelete(st))

	// Access (owner only, re-checked inside every handler)
	registerExactCallback(b, callback.AccessMenu, access.HandleMenu)
	registerPrefixCallback(b, callback.AccessListPrefix, access.HandleList(st, cfg.OwnerID))
	registerPrefixCallback(b, callback.AccessUserPrefix, access.HandleUser(st, cfg.OwnerID))
	registerPrefixCallback(b, callback.AccessAddPrefix, access.HandleAddPrompt(st))
	registerPrefixCallback(b, callback.AccessDeletePromptPrefix, access.HandleDeletePrompt(st, cfg.OwnerID))
	registerPrefixCallback(b, callback.AccessDeletePrefix, access.HandleDelete(st, cfg.OwnerID))
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
