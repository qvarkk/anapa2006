package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"qq/anapa2006/internal/config"
	"qq/anapa2006/internal/logger"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram"

	"github.com/go-telegram/bot"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := logger.NewSlogger(cfg.Debug)
	slog.SetDefault(logger.Logger)

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"couldn't create store",
			slog.Any("error", err.Error()))
		os.Exit(1)
	}
	defer st.Close()

	ms := telegram.NewMiddlewareStore(st)
	hs := telegram.NewHandlerStore(st)

	opts := []bot.Option{
		bot.WithMiddlewares(ms.RequireAllowedMiddleware),
		bot.WithDefaultHandler(hs.DefaultHandler),
	}

	slog.Info("starting bot...")
	b, err := bot.New(cfg.TelegramBotToken, opts...)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"couldn't start bot",
			slog.Any("error", err.Error()))
		os.Exit(1)
	}

	// TODO: remove
	b.RegisterHandler(
		bot.HandlerTypeMessageText, "/tmp",
		bot.MatchTypeExact, hs.GiveAdminHandler,
	)

	slog.Info("bot started")
	b.Start(ctx)
}
