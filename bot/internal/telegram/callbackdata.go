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

	fetch     string = "list:fetched"
	scheduled string = "list:scheduled"

	// lat:<page>
	fetchLatest string = "lat:%d"

	// grp:p:<page>
	fetchChannels string = "grp:p:%d"
	// grp:c:<source_id>:<post_page>:<group_page>
	fetchChannelPosts string = "grp:c:%d:%d:%d"

	// post:<post_id>:<prev_callback>
	fetchPost string = "post:%d:%s"

	scheduledList string = "later:l:%d"
	scheduledSent string = "later:s:%d"

	// sched::<post_id>:<prev_callback>
	scheduleUse  string = "sched:u:%d:%s"
	scheduleEdit string = "sched:e:%d:%s"
	scheduleSkip string = "sched:s:%d:%s"

	// sched:get:<draft_id>:<prev_callback>
	scheduleInput string = "sched:get:%d:%s"
	// sched:custom:<draft_id>:<prev_callback>
	scheduleCreateCustom string = "sched:custom:%d:%s"
	// sched:set:<draft_id>:<dur>:<prev_callback>
	scheduleCreate string = "sched:set:%d:%s:%s"
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
