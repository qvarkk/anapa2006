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

	// post:v:<post_id>:<prev_callback>
	FetchedPostPrefix = "post:v:"
	FetchedPost       = FetchedPostPrefix + "%d:%s"

	// post:s:<post_id>:<prev_callback>
	PostShowPrefix    = "post:s:"
	PostShow          = PostShowPrefix + "%d:%s"
	PostHidePrefix    = "post:h:"
	PostHide          = PostHidePrefix + "%d:%s"
	PostArchivePrefix = "post:a:"
	PostArchive       = PostArchivePrefix + "%d:%s"
	PostDeletePrefix  = "post:d:"
	PostDelete        = PostDeletePrefix + "%d:%s"

	// q::<page>
	QueueListPrefix = "q:list:"
	QueueList       = QueueListPrefix + "%d"
	QueueSentPrefix = "q:sent:"
	QueueSent       = QueueSentPrefix + "%d"

	// q:<sched_id>:<prev_callback>
	QueueSchedulePrefix = "sched:"
	QueueSchedule       = QueueSchedulePrefix + "%d:%s"

	// draft
	DraftsMenu = "draft:menu"
	// draft:list:<page>
	DraftsListPrefix = "draft:list:"
	DraftsList       = DraftsListPrefix + "%d"
	// draft:v:<post_id>:<prev_callback>
	DraftDetailPrefix = "draft:v:"
	DraftDetail       = DraftDetailPrefix + "%d:%s"
	// draft:p::<post_id>:<prev_callback>
	DraftPromptEditTextPrefix  = "draft:p:e:t:"
	DraftPromptEditText        = DraftPromptEditTextPrefix + "%d:%s"
	DraftPromptEditMediaPrefix = "draft:p:e:m"
	DraftPromptEditMedia       = DraftPromptEditMediaPrefix + "%d:%s"
	DraftPromptQueuePrefix     = "draft:p:q:"
	DraftPromptQueue           = DraftPromptQueuePrefix + "%d:%s"
	DraftPromptDequeuePrefix   = "draft:p:dq:"
	DraftPromptDequeue         = DraftPromptDequeuePrefix + "%d:%s"
	DraftPromptDeletePrefix    = "draft:p:d:"
	DraftPromptDelete          = DraftPromptDeletePrefix + "%d:%s"
	// draft:c:<post_id>:<prev_callback>
	DraftCreatePrefix = "draft:c:"
	DraftCreate       = DraftCreatePrefix + "%d:%s"
	// draft:e::<post_id>:<prev_callback>
	DraftEditTextPrefix  = "draft:e:t:"
	DraftEditText        = DraftEditTextPrefix + "%d:%s"
	DraftEditMediaPrefix = "draft:e:m:"
	DraftEditMedia       = DraftEditMediaPrefix + "%d:%s"
	// draft:q:<post_id>:<prev_callback>
	DraftQueuePrefix = "draft:q:"
	DraftQueue       = DraftQueuePrefix + "%d:%s:%s"
	// draft:dq:<post_id>:<prev_callback>
	DraftDequeuePrefix = "draft:dq:"
	DraftDequeue       = DraftDequeuePrefix + "%d:%s"
	// draft:d:<post_id>:<prev_callback>
	DraftDeletePrefix = "draft:d:"
	DraftDelete       = DraftDeletePrefix + "%d:%s"
	// draft:show:<post_id>
	DraftShowPrefix = "draft:show:"
	DraftShow       = DraftShowPrefix + "%d:%s"
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
