package queue

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

// TODO: delete.
// resolve possible DRY with telegram/fetched/post.go:HandlePostDetail()
func HandleScheduleDetail(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)
		callbackData := update.CallbackQuery.Data

		parts := strings.SplitN(callbackData, ":", 3)
		if len(parts) != 3 {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"malformed schedule detail callback",
				slog.String("data", callbackData),
			)
			return
		}
		scheduleID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"malformed schedule id",
				slog.String("data", callbackData),
			)
			return
		}
		origin := parts[2]

		schedule, err := st.GetScheduleWithDraftData(ctx, scheduleID)
		if err != nil {
			return
		}

		media, err := st.ListDraftMedia(ctx, schedule.PostID)
		if err != nil {
			return
		}

		counts := map[string]int{}
		for _, m := range media {
			counts[m.Kind]++
		}

		text := i18n.T(lang, i18n.ScheduledDetail,
			schedule.ScheduledAt.Format("02.01.2006 15:04"), render.ScheduleStatusLabel(lang, ""),
			render.AttachmentsSummary(counts), schedule.ExternalID, schedule.FinalText,
		)

		kb := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
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
				"edit message to schedule detail failed",
				slog.Int("message_id", msgID),
				slog.Int64("chat_id", chatID),
				slog.Int64("schedule_id", scheduleID),
				slog.String("error", err.Error()),
			)
		}
	}
}
