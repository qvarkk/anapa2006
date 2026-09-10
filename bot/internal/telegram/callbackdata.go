package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	noop      string = "noop"
	firstPage string = "page:first"
	lastPage  string = "page:last"

	menu string = "menu:open"

	fetch    string = "list:fetched"
	schedule string = "list:scheduled"

	fetchLatest string = "lat:%d"

	fetchChannels     string = "grp:p:%d"
	fetchChannelPosts string = "grp:c:%d:%d:%d"

	fetchPost string = "post:%d:%s"
)

func ackCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"answer callback query failed",
			slog.String("error", err.Error()),
		)
	}
}

func parseIntCallbackPart(data string, idx int) int {
	parts := strings.Split(data, ":")
	if idx >= len(parts) {
		return 0
	}
	n, _ := strconv.Atoi(parts[idx])
	return n
}

// Little helper to get callback w/ format
func format(cb string, args ...any) string {
	if len(args) == 0 {
		return cb
	}
	return fmt.Sprintf(cb, args...)
}
