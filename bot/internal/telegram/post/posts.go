package post

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/render"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func HandlePostDetail(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)
		callbackData := update.CallbackQuery.Data

		parts := strings.SplitN(callbackData, ":", 3)
		if len(parts) != 3 {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"malformed post detail callback",
				slog.String("data", callbackData),
			)
			return
		}
		postID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"malformed post id",
				slog.String("data", callbackData),
			)
			return
		}
		origin := parts[2]

		post, err := st.GetPostWithSource(ctx, postID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get post with source",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}
		media, err := st.ListPostMedia(ctx, postID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"list post media",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
		}

		counts := map[string]int{}
		for _, m := range media {
			counts[m.Kind]++
		}

		text := i18n.T(lang, i18n.PostDetail,
			post.ChannelHandle, post.PublishedAt.Time.Format("02.01.2006 15:04"),
			render.PostStatusLabel(lang, post.Status), post.ExternalID, render.AttachmentsSummary(counts), post.RawText,
		)

		kb := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: i18n.T(lang, i18n.BtnUse), CallbackData: callback.Format(callback.ScheduleUse, postID, callbackData)}},
			{{Text: i18n.T(lang, i18n.BtnEdit), CallbackData: callback.Format(callback.ScheduleEdit, postID, callbackData)}},
			{{Text: i18n.T(lang, i18n.BtnSkip), CallbackData: callback.Format(callback.ScheduleSkip, postID, callbackData)}},
			{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: origin}},
		}}

		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID: msgID, ChatID: chatID, Text: text,
			ReplyMarkup: kb, ParseMode: models.ParseModeHTML,
			LinkPreviewOptions: &models.LinkPreviewOptions{
				IsDisabled: bot.True(),
			},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to post detail failed",
				slog.Int("message_id", msgID),
				slog.Int64("chat_id", chatID),
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
		}
	}
}
