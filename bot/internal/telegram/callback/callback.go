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

	Fetched = "menu:fetched"
	Queued  = "menu:queued"

	// lat:<page>
	FetchedLatestPrefix = "lat:"
	FetchedLatest       = FetchedLatestPrefix + "%d"

	// grp:p:<page>
	FetchedChannelsPrefix = "grp:p:"
	FetchedChannels       = FetchedChannelsPrefix + "%d"
	// grp:c:<source_id>:<post_page>:<group_page>
	FetchedChannelPostsPrefix = "grp:c:"
	FetchedChannelPosts       = FetchedChannelPostsPrefix + "%d:%d:%d"

	// post:<post_id>:<prev_callback>
	FetchedPostPrefix = "post:"
	FetchedPost       = FetchedPostPrefix + "%d:%s"

	// q::<page>
	QueueListPrefix = "q:list:"
	QueueList       = QueueListPrefix + "%d"
	QueueSentPrefix = "q:sent:"
	QueueSent       = QueueSentPrefix + "%d"

	// q:<sched_id>:<prev_callback>
	QueueSchedulePrefix = "sched:"
	QueueSchedule       = QueueSchedulePrefix + "%d:%s"

	// plan::<post_id>:<prev_callback>
	PlannerUsePrefix  = "plan:use:"
	PlannerUse        = PlannerUsePrefix + "%d:%s"
	PlannerEditPrefix = "plan:edit:"
	PlannerEdit       = PlannerEditPrefix + "%d:%s"
	PlannerSkipPrefix = "plan:skip:"
	PlannerSkip       = PlannerSkipPrefix + "%d:%s"

	// sched:get:<draft_id>:<prev_callback>
	PlannerInputPrefix = "plan:get:"
	PlannerInput       = PlannerInputPrefix + "%d:%s"
	// sched:custom:<draft_id>:<prev_callback>
	PlannerCreateCustomPrefix = "plan:custom:"
	PlannerCreateCustom       = PlannerCreateCustomPrefix + "%d:%s"
	// sched:create:<draft_id>:<dur>:<prev_callback>
	PlannerCreatePrefix = "plan:create:"
	PlannerCreate       = PlannerCreatePrefix + "%d:%s:%s"
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
