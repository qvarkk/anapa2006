package fetched

import (
	"context"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Post visibility / archival / removal actions, reachable from the post detail screen.

func HandleShow(st *store.Store) bot.HandlerFunc {
	return handlePostAction(func(ctx context.Context, postID int64) error {
		return st.UnhidePost(ctx, postID)
	})
}

func HandleHide(st *store.Store) bot.HandlerFunc {
	return handlePostAction(func(ctx context.Context, postID int64) error {
		return st.HidePost(ctx, postID)
	})
}

func HandleArchive(st *store.Store) bot.HandlerFunc {
	return handlePostAction(func(ctx context.Context, postID int64) error {
		return st.MarkPostArchived(ctx, postID)
	})
}

func HandleUnarchive(st *store.Store) bot.HandlerFunc {
	return handlePostAction(func(ctx context.Context, postID int64) error {
		return st.UnhidePost(ctx, postID)
	})
}

func HandleDelete(st *store.Store) bot.HandlerFunc {
	return handlePostAction(func(ctx context.Context, postID int64) error {
		return st.WithTx(ctx, func(q *db.Queries) error {
			// draft_media.origin_media_id references post_media without a
			// cascade, so the draft has to go before the media it copied.
			if err := q.DeleteDraft(ctx, postID); err != nil {
				return err
			}
			if err := q.DeletePostMedia(ctx, postID); err != nil {
				return err
			}
			return q.DeletePost(ctx, postID)
		})
	})
}

// handlePostAction parses `post:<action>:<post_id>:<origin>`, applies the status
// change, and reports the result with a way back to the previous screen.
func handlePostAction(apply func(ctx context.Context, postID int64) error) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 2, 1)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"parse post action callback",
				slog.String("error", err.Error()),
			)
			return
		}
		postID := data[0]

		if err := apply(ctx, postID); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"apply post action",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)

		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: msgID,
			Text:      i18n.T(lang, i18n.OperationSuccess),
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						{
							Text:         i18n.T(lang, i18n.BtnBack),
							CallbackData: origin,
						},
					},
				},
			},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to post action result failed",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
		}
	}
}
