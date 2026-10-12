package targets

import (
	"context"
	"database/sql"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

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
		targetID := data[0]

		target, err := st.GetTargetChannelByID(ctx, targetID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get target channel by id",
				slog.Int64("target_id", targetID),
				slog.String("error", err.Error()),
			)
			return
		}

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)
		backText := i18n.T(lang, i18n.BtnBack)

		// origin is the list callback; recover the page for the post-delete return.
		page := callback.ParseIntPart(origin, 2)

		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID,
			Text: i18n.T(lang, i18n.TargetDeletePrompt, targetLabel(target)),
			ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
				{{
					Text:         i18n.T(lang, i18n.BtnDelete),
					CallbackData: callback.Format(callback.TargetsDelete, targetID, page),
				}},
				{{Text: backText, CallbackData: origin}},
			}},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to target delete prompt failed",
				slog.Int64("target_id", targetID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func HandleDelete(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		targetID := int64(callback.ParseIntPart(update.CallbackQuery.Data, 2))
		page := callback.ParseIntPart(update.CallbackQuery.Data, 3)

		// Deleting a target also removes its schedules (cascade) and puts any
		// affected scheduled posts back to "new".
		err := st.WithTx(ctx, func(q *db.Queries) error {
			ids, err := q.ListScheduledDraftsByTarget(ctx, sql.NullInt64{Int64: targetID, Valid: true})
			if err != nil {
				return err
			}
			for _, postID := range ids {
				if err := q.Deschedule(ctx, postID); err != nil {
					return err
				}
				if err := q.UnhidePost(ctx, postID); err != nil {
					return err
				}
			}
			return q.DeleteTargetChannel(ctx, targetID)
		})
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"delete target channel",
				slog.Int64("target_id", targetID),
				slog.String("error", err.Error()),
			)
			return
		}

		slog.LogAttrs(ctx, slog.LevelInfo, "target channel removed", slog.Int64("target_id", targetID))
		renderList(ctx, b, st, update, page)
	}
}
