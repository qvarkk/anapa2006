package fetched

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

// TODO: resolve possible DRY with telegram/queue/schedule.go:HandleScheduleDetail()
func HandlePostDetail(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)
		callbackData := update.CallbackQuery.Data

		parts := strings.SplitN(callbackData, ":", 4)
		if len(parts) != 4 {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"malformed post detail callback",
				slog.String("data", callbackData),
			)
			return
		}
		postID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"malformed post id",
				slog.String("data", callbackData),
			)
			return
		}
		origin := parts[3]

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

		draftExists, err := st.CheckPostDraftExists(ctx, postID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"check draft exists",
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

		var draftButton models.InlineKeyboardButton
		if draftExists {
			draftButton = models.InlineKeyboardButton{
				Text:         i18n.T(lang, i18n.BtnDraftView),
				CallbackData: callback.Format(callback.DraftDetail, postID, callbackData),
			}
		} else {
			draftButton = models.InlineKeyboardButton{
				Text:         i18n.T(lang, i18n.BtnDraftCreate),
				CallbackData: callback.Format(callback.DraftCreate, postID, callbackData),
			}
		}

		var postVisibilityButton models.InlineKeyboardButton
		if post.Status != "skipped" {
			postVisibilityButton = models.InlineKeyboardButton{
				Text:         i18n.T(lang, i18n.BtnPostHide),
				CallbackData: callback.Format(callback.PostHide, postID, callbackData),
			}
		} else {
			postVisibilityButton = models.InlineKeyboardButton{
				Text:         i18n.T(lang, i18n.BtnPostShow),
				CallbackData: callback.Format(callback.PostShow, postID, callbackData),
			}
		}

		kb := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{draftButton},
			{postVisibilityButton},
			{
				{
					Text:         i18n.T(lang, i18n.BtnPostDelete),
					CallbackData: callback.Format(callback.PostDelete, postID, callbackData),
				},
			},
			{
				{
					Text:         i18n.T(lang, i18n.BtnBack),
					CallbackData: origin,
				},
			},
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
