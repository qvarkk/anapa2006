package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"qq/anapa2006/internal/config"
	"qq/anapa2006/internal/logger"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := logger.NewSlogger(config.Debug)

	opts := []bot.Option{
		// TODO: add allowed users middleware; remove default handler
		// bot.WithMiddlewares(nil),
		bot.WithDefaultHandler(handler),
	}

	logger.Logger.Info("starting bot...")
	b, err := bot.New(config.TelegramBotToken, opts...)
	if err != nil {
		logger.Logger.Error("start bot", slog.Any("error", err.Error()))
		os.Exit(1)
	}

	logger.Logger.Info("bot started")
	b.Start(ctx)
}

// TODO: remove temporary handler
func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}
