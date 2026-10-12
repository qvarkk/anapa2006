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

	// Channel management
	ChannelsMenu = "ch:menu"
	// ch:list:<page>
	ChannelsListPrefix = "ch:list:"
	ChannelsList       = ChannelsListPrefix + "%d"
	// ch:add:<prev_callback>
	ChannelsAddPrefix = "ch:add:"
	ChannelsAdd       = ChannelsAddPrefix + "%s"
	// ch:v:<source_id>:<prev_callback>
	ChannelsDetailPrefix = "ch:v:"
	ChannelsDetail       = ChannelsDetailPrefix + "%d:%s"
	// ch:tg:<source_id>:<prev_callback>
	ChannelsTogglePrefix = "ch:tg:"
	ChannelsToggle       = ChannelsTogglePrefix + "%d:%s"
	// ch:rmp:<source_id>:<prev_callback>
	ChannelsDeletePromptPrefix = "ch:rmp:"
	ChannelsDeletePrompt       = ChannelsDeletePromptPrefix + "%d:%s"
	// ch:rm:<source_id>:<page>
	ChannelsDeletePrefix = "ch:rm:"
	ChannelsDelete       = ChannelsDeletePrefix + "%d:%d"

	// Target channels (publishing destinations)
	TargetsMenu = "tg:menu"
	// tg:list:<page>
	TargetsListPrefix = "tg:list:"
	TargetsList       = TargetsListPrefix + "%d"
	// tg:add:<prev_callback>
	TargetsAddPrefix = "tg:add:"
	TargetsAdd       = TargetsAddPrefix + "%s"
	// tg:rmp:<target_id>:<prev_callback>
	TargetsDeletePromptPrefix = "tg:rmp:"
	TargetsDeletePrompt       = TargetsDeletePromptPrefix + "%d:%s"
	// tg:rm:<target_id>:<page>
	TargetsDeletePrefix = "tg:rm:"
	TargetsDelete       = TargetsDeletePrefix + "%d:%d"

	// Access management
	AccessMenu = "acc:menu"
	// acc:list:<page>
	AccessListPrefix = "acc:list:"
	AccessList       = AccessListPrefix + "%d"
	// acc:add:<prev_callback>
	AccessAddPrefix = "acc:add:"
	AccessAdd       = AccessAddPrefix + "%s"
	// acc:v:<user_id>:<prev_callback>
	AccessUserPrefix = "acc:v:"
	AccessUser       = AccessUserPrefix + "%d:%s"
	// acc:rmp:<user_id>:<prev_callback>
	AccessDeletePromptPrefix = "acc:rmp:"
	AccessDeletePrompt       = AccessDeletePromptPrefix + "%d:%s"
	// acc:rm:<user_id>:<page>
	AccessDeletePrefix = "acc:rm:"
	AccessDelete       = AccessDeletePrefix + "%d:%d"

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
	PostShowPrefix      = "post:s:"
	PostShow            = PostShowPrefix + "%d:%s"
	PostHidePrefix      = "post:h:"
	PostHide            = PostHidePrefix + "%d:%s"
	PostArchivePrefix   = "post:a:"
	PostArchive         = PostArchivePrefix + "%d:%s"
	PostUnarchivePrefix = "post:u:"
	PostUnarchive       = PostUnarchivePrefix + "%d:%s"
	PostDeletePrefix    = "post:d:"
	PostDelete          = PostDeletePrefix + "%d:%s"

	// draft
	DraftsMenu = "draft:menu"

	// draft:la:<page> | draft:lp:<page> | draft:lf:<page>
	DraftsListActivePrefix = "draft:la:"
	DraftsListActive       = DraftsListActivePrefix + "%d"
	DraftsListPostedPrefix = "draft:lp:"
	DraftsListPosted       = DraftsListPostedPrefix + "%d"
	DraftsListFailedPrefix = "draft:lf:"
	DraftsListFailed       = DraftsListFailedPrefix + "%d"

	// draft:v:<post_id>:<prev_callback>
	DraftDetailPrefix = "draft:v:"
	DraftDetail       = DraftDetailPrefix + "%d:%s"

	// draft:t:<post_id>:<page>:<prev_callback> (target picker)
	DraftTargetListPrefix = "draft:t:"
	DraftTargetList       = DraftTargetListPrefix + "%d:%d:%s"
	// draft:tp:<post_id>:<target_id>:<prev_callback>
	DraftTargetPickPrefix = "draft:tp:"
	DraftTargetPick       = DraftTargetPickPrefix + "%d:%d:%s"
	// draft:tc:<post_id>:<prev_callback>
	DraftTargetClearPrefix = "draft:tc:"
	DraftTargetClear       = DraftTargetClearPrefix + "%d:%s"

	// draft:p:e:t:<post_id>:<prev_callback>
	DraftPromptEditTextPrefix = "draft:p:e:t:"
	DraftPromptEditText       = DraftPromptEditTextPrefix + "%d:%s"
	// draft:p:e:m:<post_id>:<prev_callback>
	DraftPromptEditMediaPrefix = "draft:p:e:m:"
	DraftPromptEditMedia       = DraftPromptEditMediaPrefix + "%d:%s"
	// draft:p:q:<post_id>:<prev_callback>
	DraftPromptQueuePrefix = "draft:p:q:"
	DraftPromptQueue       = DraftPromptQueuePrefix + "%d:%s"
	// draft:p:dq:<post_id>:<prev_callback>
	DraftPromptDequeuePrefix = "draft:p:dq:"
	DraftPromptDequeue       = DraftPromptDequeuePrefix + "%d:%s"
	// draft:p:d:<post_id>:<prev_callback>
	DraftPromptDeletePrefix = "draft:p:d:"
	DraftPromptDelete       = DraftPromptDeletePrefix + "%d:%s"
	// draft:p:c:<post_id>:<prev_callback>
	DraftPromptCustomPrefix = "draft:p:c:"
	DraftPromptCustom       = DraftPromptCustomPrefix + "%d:%s"

	// draft:c:<post_id>:<prev_callback>
	DraftCreatePrefix = "draft:c:"
	DraftCreate       = DraftCreatePrefix + "%d:%s"

	// draft:q:<post_id>:<duration>:<prev_callback>
	DraftQueuePrefix = "draft:q:"
	DraftQueue       = DraftQueuePrefix + "%d:%s:%s"
	// draft:dq:<post_id>:<prev_callback>
	DraftDequeuePrefix = "draft:dq:"
	DraftDequeue       = DraftDequeuePrefix + "%d:%s"
	// draft:d:<post_id>:<prev_callback>
	DraftDeletePrefix = "draft:d:"
	DraftDelete       = DraftDeletePrefix + "%d:%s"
	// draft:show:<post_id>:<prev_callback>
	DraftShowPrefix = "draft:show:"
	DraftShow       = DraftShowPrefix + "%d:%s"

	// media editor
	// NOTE: the list prefix must NOT be "draft:m:" because that is also a prefix
	// of draft:m:v:/draft:m:d:/draft:m:s:/draft:m:a:, which would shadow them in
	// prefix-matched handler registration.
	// draft:ml:<post_id>:<prev_callback>
	DraftMediaListPrefix = "draft:ml:"
	DraftMediaList       = DraftMediaListPrefix + "%d:%s"
	// draft:m:v:<post_id>:<media_id>:<prev_callback>
	DraftMediaDetailPrefix = "draft:m:v:"
	DraftMediaDetail       = DraftMediaDetailPrefix + "%d:%d:%s"
	// draft:m:d:<post_id>:<media_id>:<prev_callback>
	DraftMediaDeletePrefix = "draft:m:d:"
	DraftMediaDelete       = DraftMediaDeletePrefix + "%d:%d:%s"
	// draft:m:s:<post_id>:<media_id>:<prev_callback>
	DraftMediaSubstitutePromptPrefix = "draft:m:s:"
	DraftMediaSubstitutePrompt       = DraftMediaSubstitutePromptPrefix + "%d:%d:%s"
	// draft:m:a:<post_id>:<prev_callback>
	DraftMediaAddPromptPrefix = "draft:m:a:"
	DraftMediaAddPrompt       = DraftMediaAddPromptPrefix + "%d:%s"
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
