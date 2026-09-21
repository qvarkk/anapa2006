package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot/models"
)

func extractIdentity(update *models.Update) (userID int64, chatID int64, ok bool) {
	switch {
	case update.Message != nil && update.Message.From != nil:
		return update.Message.From.ID, update.Message.Chat.ID, true
	case update.CallbackQuery != nil:
		msg := update.CallbackQuery.Message.Message
		if msg == nil {
			slog.LogAttrs(
				context.Background(),
				slog.LevelWarn,
				"callback query with inaccessible message",
				slog.Int64("update_id", update.ID),
				slog.Int64("callback_from", update.CallbackQuery.From.ID),
				slog.String("callback_data", update.CallbackQuery.Data),
			)
			return 0, 0, false
		}
		return update.CallbackQuery.From.ID, msg.Chat.ID, true
	default:
		slog.LogAttrs(
			context.Background(),
			slog.LevelDebug,
			"update did not match any known identity shape",
			slog.Int64("update_id", update.ID),
			slog.Bool("has_message", update.Message != nil),
			slog.Bool("has_callback_query", update.CallbackQuery != nil),
			slog.Bool("has_edited_message", update.EditedMessage != nil),
			slog.Bool("has_channel_post", update.ChannelPost != nil),
			slog.Bool("has_my_chat_member", update.MyChatMember != nil),
		)
		return 0, 0, false
	}
}

// extract [nArgs]int64 data from a callback with a destination callback for a previous page
func parseBackNavigation(
	update *models.Update,
	nPrefix int,
	nArgs int,
) (data []int64, origin string, err error) {
	str := update.CallbackQuery.Data

	totalSplits := nPrefix + nArgs + 1
	parts := strings.SplitN(str, ":", totalSplits)

	if len(parts) != totalSplits {
		return nil, "", fmt.Errorf("malformed callback %q: expected at least %d components, got %d",
			str, totalSplits, len(parts))
	}

	data = make([]int64, nArgs)
	for i := range nArgs {
		rawArg := parts[nPrefix+i]
		parsedData, err := strconv.ParseInt(rawArg, 10, 64)
		if err != nil {
			return nil, "", fmt.Errorf("malformed data %q: failed to parse argument %q at index %d: %w",
				str, rawArg, nPrefix+i, err)
		}
		data[i] = parsedData
	}

	origin = parts[totalSplits-1]
	if origin == "" {
		return nil, "", fmt.Errorf("malformed callback %q: trailing origin menu is empty", str)
	}

	return data, origin, nil
}

func chatIDFromUpdate(update *models.Update) (int64, bool) {
	if update.Message != nil {
		return update.Message.Chat.ID, true
	}
	if update.CallbackQuery != nil && update.CallbackQuery.Message.Message != nil {
		return update.CallbackQuery.Message.Message.Chat.ID, true
	}
	return 0, false
}

func callbackTarget(update *models.Update) (chatID int64, messageID int) {
	msg := update.CallbackQuery.Message.Message
	return msg.Chat.ID, msg.ID
}
