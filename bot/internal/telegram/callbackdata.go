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

// TODO: remove, just string everywhere, whatever
type callbackAction string

const (
	callbackNoop callbackAction = "noop"

	callbackOpenMenu callbackAction = "menu:open"

	callbackListFetched   callbackAction = "list:fetched"
	callbackListScheduled callbackAction = "list:scheduled"

	callbackListFetchedByGroups callbackAction = "grp:p:%d"
	callbackListFetchedLatest   callbackAction = "lat:%d"

	callbackLatestPostDetail callbackAction = "post:%d:%s"

	callbackListFetchedGroups     callbackAction = "grp:p:%d"
	callbackListFetchedGroupPosts callbackAction = "grp:c:%d:%d:%d"
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

func encodeCallback(action callbackAction, args ...any) string {
	if len(args) == 0 {
		return string(action)
	}
	return fmt.Sprintf(string(action), args...)
}
