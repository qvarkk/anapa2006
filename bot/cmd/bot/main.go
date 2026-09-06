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

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"couldn't create store",
			slog.Any("error", err.Error()))
		os.Exit(1)
	}
	defer st.Close()

	slog.Info("starting bot...")
	b, err := telegram.New(ctx, cfg.TelegramBotToken, st)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"couldn't create bot",
			slog.Any("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("bot started")
	b.Start(ctx)
	slog.Info("bot stopped")
}
