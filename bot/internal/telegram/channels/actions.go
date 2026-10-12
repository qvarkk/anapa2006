package channels

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/render"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func HandleToggle(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		sourceID := int64(callback.ParseIntPart(update.CallbackQuery.Data, 2))
		page := callback.ParseIntPart(update.CallbackQuery.Data, 3)

		if err := st.ToggleSourceActive(ctx, sourceID); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"toggle source active",
				slog.Int64("source_id", sourceID),
				slog.String("error", err.Error()),
			)
			return
		}

		renderList(ctx, b, st, update, page)
	}
}

func HandleDeletePrompt(st *store.Store) bot.HandlerFunc {
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

		count, err := st.CountAllPostsBySource(ctx, sourceID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"count posts by source",
				slog.Int64("source_id", sourceID),
				slog.String("error", err.Error()),
			)
			return
		}

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)
		backText := i18n.T(lang, i18n.BtnBack)

		if count > 0 {
			render.EditNotice(
				ctx, b, chatID, msgID,
				i18n.T(lang, i18n.ChannelDeleteBlocked, source.ChannelHandle, count),
				backText, origin,
			)
			return
		}

		// origin is the list callback, so its second component is the page.
		page := callback.ParseIntPart(origin, 2)

		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID,
			Text: i18n.T(lang, i18n.ChannelDeletePrompt, source.ChannelHandle),
			ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
				{{
					Text:         i18n.T(lang, i18n.BtnDelete),
					CallbackData: callback.Format(callback.ChannelsDelete, sourceID, page),
				}},
				{{Text: backText, CallbackData: origin}},
			}},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to channel delete prompt failed",
				slog.Int64("source_id", sourceID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func HandleDelete(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		sourceID := int64(callback.ParseIntPart(update.CallbackQuery.Data, 2))
		page := callback.ParseIntPart(update.CallbackQuery.Data, 3)

		// Posts reference the source, so a channel that still has any (even
		// failed) posts cannot be removed.
		count, err := st.CountAllPostsBySource(ctx, sourceID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"count posts by source",
				slog.Int64("source_id", sourceID),
				slog.String("error", err.Error()),
			)
			return
		}

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)
		backText := i18n.T(lang, i18n.BtnBack)

		if count > 0 {
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

			render.EditNotice(
				ctx, b, chatID, msgID,
				i18n.T(lang, i18n.ChannelDeleteBlocked, source.ChannelHandle, count),
				backText, callback.Format(callback.ChannelsList, page),
			)
			return
		}

		if err := st.DeleteSource(ctx, sourceID); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"delete source",
				slog.Int64("source_id", sourceID),
				slog.String("error", err.Error()),
			)
			return
		}

		renderList(ctx, b, st, update, page)
	}
}
