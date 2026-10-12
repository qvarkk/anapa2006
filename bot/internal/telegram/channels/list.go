package channels

import (
	"context"
	"fmt"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/pagination"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func HandleList(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		page := callback.ParseIntPart(update.CallbackQuery.Data, 2)
		renderList(ctx, b, st, update, page)
	}
}

// renderList shows the channel page, editing the message the callback came from.
func renderList(ctx context.Context, b *bot.Bot, st *store.Store, update *models.Update, page int) {
	lang := extract.Lang(ctx)

	total, err := st.CountSources(ctx)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"count sources",
			slog.String("error", err.Error()),
		)
		return
	}

	totalPages := pagination.TotalPages(int(total), RecordsPerPage)
	page = pagination.ClampPage(page, totalPages)

	sources, err := st.ListSourcesWithStats(ctx, db.ListSourcesWithStatsParams{
		Limit: RecordsPerPage, Offset: int64(page * RecordsPerPage),
	})
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"list sources with stats",
			slog.String("error", err.Error()),
		)
		return
	}

	listCallback := callback.Format(callback.ChannelsList, page)

	var sb strings.Builder
	sb.WriteString(i18n.T(lang, i18n.ChannelsListHeader, total))
	sb.WriteString("\n\n")

	rows := make([][]models.InlineKeyboardButton, 0, len(sources)+3)
	for _, source := range sources {
		status := i18n.T(lang, i18n.ChannelStatusOff)
		if source.Active == 1 {
			status = i18n.T(lang, i18n.ChannelStatusActive)
		}

		sb.WriteString(i18n.T(lang, i18n.ChannelsListEntry, source.ID, source.ChannelHandle, status))
		sb.WriteString(i18n.T(lang, i18n.ChannelStats, source.Fetched, source.Failed, source.Sent))
		sb.WriteString("\n\n")

		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("📡 @%s", source.ChannelHandle),
				CallbackData: callback.Format(callback.ChannelsDetail, source.ID, listCallback),
			},
		})
	}

	if totalPages > 1 {
		prev, next := callback.FirstPage, callback.LastPage
		if page > 0 {
			prev = callback.Format(callback.ChannelsList, page-1)
		}
		if page < totalPages-1 {
			next = callback.Format(callback.ChannelsList, page+1)
		}
		rows = append(rows, pagination.BuildPaginationRow(page, totalPages, prev, next))
	}

	rows = append(rows, []models.InlineKeyboardButton{
		{Text: i18n.T(lang, i18n.BtnChannelAdd), CallbackData: callback.Format(callback.ChannelsAdd, listCallback)},
	})
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: i18n.T(lang, i18n.BtnBack), CallbackData: callback.ChannelsMenu},
	})

	chatID, msgID := extract.CallbackTarget(update)
	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID,
		Text:        sb.String(),
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
		ParseMode:   models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to channels list failed",
			slog.String("error", err.Error()),
		)
	}
}

func HandleDetail(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 2, 1)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"parse back navagation",
				slog.String("error", err.Error()),
			)
			return
		}
		sourceID := data[0]
		renderDetail(ctx, b, st, update, sourceID, origin)
	}
}

// renderDetail shows one channel with its stats and toggle/delete actions. The
// listCallback is the callback to return to from the detail screen.
func renderDetail(ctx context.Context, b *bot.Bot, st *store.Store, update *models.Update, sourceID int64, listCallback string) {
	lang := extract.Lang(ctx)
	chatID, msgID := extract.CallbackTarget(update)
	backText := i18n.T(lang, i18n.BtnBack)

	source, err := st.GetSourceByID(ctx, sourceID)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"get source by id",
			slog.Int64("source_id", sourceID),
			slog.String("error", err.Error()),
		)
		return
	}

	stats, err := st.GetSourceStats(ctx, sourceID)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"get source stats",
			slog.Int64("source_id", sourceID),
			slog.String("error", err.Error()),
		)
		return
	}

	thisCallback := callback.Format(callback.ChannelsDetail, sourceID, listCallback)

	status := i18n.T(lang, i18n.ChannelStatusOff)
	toggleText := i18n.T(lang, i18n.BtnChannelEnable)
	if source.Active == 1 {
		status = i18n.T(lang, i18n.ChannelStatusActive)
		toggleText = i18n.T(lang, i18n.BtnChannelDisable)
	}

	text := i18n.T(lang, i18n.ChannelDetail,
		source.ChannelHandle, status, stats.Fetched, stats.Failed, stats.Sent)

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: toggleText, CallbackData: callback.Format(callback.ChannelsToggle, sourceID, thisCallback)}},
		{{Text: i18n.T(lang, i18n.BtnDelete), CallbackData: callback.Format(callback.ChannelsDeletePrompt, sourceID, thisCallback)}},
		{{Text: backText, CallbackData: listCallback}},
	}}

	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID,
		Text:        text,
		ReplyMarkup: kb,
		ParseMode:   models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to channel detail failed",
			slog.Int64("source_id", sourceID),
			slog.String("error", err.Error()),
		)
	}
}
