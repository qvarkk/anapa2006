package callback

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
	Noop      = "noop"
	FirstPage = "page:first"
	LastPage  = "page:last"

	Start = "start"

	Fetch     = "list:fetched"
	Scheduled = "list:scheduled"

	// lat:<page>
	FetchLatestMatch = "lat:"
	FetchLatest      = FetchLatestMatch + "%d"

	// grp:p:<page>
	FetchChannelsMatch = "grp:p:"
	FetchChannels      = FetchChannelsMatch + "%d"
	// grp:c:<source_id>:<post_page>:<group_page>
	FetchChannelPostsMatch = "grp:c:"
	FetchChannelPosts      = FetchChannelPostsMatch + "%d:%d:%d"

	// post:<post_id>:<prev_callback>
	FetchPostMatch = "post:"
	FetchPost      = FetchPostMatch + "%d:%s"

	// sched::<page>
	ScheduledListMatch = "sched:list:"
	ScheduledList      = ScheduledListMatch + "%d"
	ScheduledSentMatch = "sched:sent:"
	ScheduledSent      = ScheduledSentMatch + "%d"

	// sched::<post_id>:<prev_callback>
	ScheduleUseMatch  = "sched:use:"
	ScheduleUse       = ScheduleUseMatch + "%d:%s"
	ScheduleEditMatch = "sched:edit:"
	ScheduleEdit      = ScheduleEditMatch + "%d:%s"
	ScheduleSkipMatch = "sched:skip:"
	ScheduleSkip      = ScheduleSkipMatch + "%d:%s"

	// sched:get:<draft_id>:<prev_callback>
	ScheduleInputMatch = "sched:get:"
	ScheduleInput      = ScheduleInputMatch + "%d:%s"
	// sched:custom:<draft_id>:<prev_callback>
	ScheduleCreateCustomMatch = "sched:custom:"
	ScheduleCreateCustom      = ScheduleCreateCustomMatch + "%d:%s"
	// sched:create:<draft_id>:<dur>:<prev_callback>
	ScheduleCreateMatch = "sched:create:"
	ScheduleCreate      = ScheduleCreateMatch + "%d:%s:%s"
)

const (
	In30m  = "30m"
	In1hr  = "1h"
	In3hr  = "3h"
	In6hr  = "6h"
	In12hr = "12h"
)

func Ack(ctx context.Context, b *bot.Bot, update *models.Update) {
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

func ParseIntPart(data string, idx int) int {
	parts := strings.Split(data, ":")
	if idx >= len(parts) {
		return 0
	}
	n, _ := strconv.Atoi(parts[idx])
	return n
}

// Little helper to get callback w/ format
func Format(cb string, args ...any) string {
	if len(args) == 0 {
		return cb
	}
	return fmt.Sprintf(cb, args...)
}
