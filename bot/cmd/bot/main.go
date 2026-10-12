package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"qq/anapa2006/internal/config"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/fetcher"
	"qq/anapa2006/internal/logger"
	"qq/anapa2006/internal/scheduler"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram"
	"qq/anapa2006/internal/telegram/cache"
	"qq/anapa2006/internal/telegram/sender"
	"syscall"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := logger.NewSlogger(cfg.Debug)
	slog.SetDefault(logger.Logger)

	if cfg.TelegramOwnerID == 0 {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"TELEGRAM_OWNER_ID is not set: /bootstrap will refuse, so nobody can get access",
		)
	} else {
		slog.LogAttrs(
			ctx, slog.LevelInfo,
			"owner configured, waiting for /bootstrap if they are not in the database yet",
			slog.Int64("owner_id", cfg.TelegramOwnerID),
		)
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"couldn't create store",
			slog.Any("error", err.Error()))
		os.Exit(1)
	}
	defer st.Close()

	if cfg.TelegramChannelChatID != 0 {
		if err := seedDefaultTarget(ctx, st, cfg.TelegramChannelChatID); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"seed default target channel",
				slog.String("error", err.Error()),
			)
		}
	}

	tgCfg := telegram.Config{
		Token:   cfg.TelegramBotToken,
		OwnerID: cfg.TelegramOwnerID,
	}

	slog.LogAttrs(ctx, slog.LevelInfo, "starting bot...")
	b, err := telegram.New(ctx, tgCfg, st)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"couldn't create bot",
			slog.Any("error", err.Error()))
		os.Exit(1)
	}

	cacher := cache.NewMediaCacher(b, cfg.TelegramCacheChannelChatID)
	fetcher := fetcher.NewFetcher(st, cfg.RsshubBaseUrl, cacher, nil)
	go fetcher.Run(ctx, cfg.FetchInterval)

	sender := sender.New(b)
	scheduler := scheduler.NewScheduler(st, sender)
	go scheduler.Run(ctx, cfg.PostInterval)

	slog.LogAttrs(ctx, slog.LevelInfo, "bot started")
	b.Start(ctx)
	slog.LogAttrs(ctx, slog.LevelInfo, "bot stopped")
}

// seedDefaultTarget inserts the legacy TELEGRAM_CHANNEL_CHAT_ID as the first
// target channel, so existing deployments keep working after the target moved
// into the database. Users can then add/remove targets from the bot UI.
func seedDefaultTarget(ctx context.Context, st *store.Store, chatID int64) error {
	count, err := st.CountTargetChannels(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	_, err = st.CreateTargetChannel(ctx, db.CreateTargetChannelParams{
		ChatID: chatID,
		Handle: sql.NullString{},
		Title:  "Канал по умолчанию",
	})
	return err
}
