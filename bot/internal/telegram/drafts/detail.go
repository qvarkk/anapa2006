package drafts

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/render"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

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
		postID := data[0]

		draft, err := st.GetPostDraftByID(ctx, postID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get draft fetails by id",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}

		scheduled := true
		schedule, err := st.GetPostScheduleByID(ctx, postID)
		if errors.Is(err, sql.ErrNoRows) {
			scheduled = false
		} else if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get schedule",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}

		post, err := st.GetPost(ctx, postID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get post",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}

		lang := extract.Lang(ctx)
		thisCallback := update.CallbackQuery.Data

		var scheduleTime string
		var queueButtons []models.InlineKeyboardButton
		if scheduled {
			queueButtons = append(queueButtons, models.InlineKeyboardButton{
				Text:         i18n.T(lang, i18n.BtnDraftRequeue),
				CallbackData: callback.Format(callback.DraftPromptQueue, draft.PostID, thisCallback),
			})
			queueButtons = append(queueButtons, models.InlineKeyboardButton{
				Text:         i18n.T(lang, i18n.BtnDraftDequeue),
				CallbackData: callback.Format(callback.DraftDequeue, draft.PostID, thisCallback),
			})

			scheduleTime = schedule.ScheduledAt.Format("02.01.2006 15:04")
		} else {
			queueButtons = append(queueButtons, models.InlineKeyboardButton{
				Text:         i18n.T(lang, i18n.BtnDraftQueue),
				CallbackData: callback.Format(callback.DraftPromptQueue, draft.PostID, thisCallback),
			})

			scheduleTime = i18n.T(lang, i18n.Unscheduled)
		}

		chatID, msgID := extract.CallbackTarget(update)

		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID: msgID,
			ChatID:    chatID,
			Text: i18n.T(
				lang,
				i18n.DraftDetail,
				draft.PostID,
				scheduleTime,
				render.PostStatusLabel(lang, post.Status),
				draft.FinalText,
				// TODO: attachments counts
				"TODO",
			),
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{{
						Text:         i18n.T(lang, i18n.BtnDraftEditText),
						CallbackData: callback.Format(callback.DraftPromptEditText, draft.PostID, thisCallback),
					}},
					{{
						// TODO: implement
						Text:         i18n.T(lang, i18n.BtnDraftEditMedia),
						CallbackData: callback.Noop,
					}},
					queueButtons,
					{{
						Text:         i18n.T(lang, i18n.BtnDraftDelete),
						CallbackData: callback.Format(callback.DraftPromptDelete, draft.PostID, thisCallback),
					}},
					{{
						Text:         i18n.T(lang, i18n.BtnDraftShow),
						CallbackData: callback.Format(callback.DraftShow, draft.PostID, thisCallback),
					}},
					{{
						Text:         i18n.T(lang, i18n.BtnBack),
						CallbackData: origin,
					}},
				},
			},
			ParseMode: models.ParseModeHTML,
			LinkPreviewOptions: &models.LinkPreviewOptions{
				IsDisabled: bot.True(),
			},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to draft detail failed",
				slog.Int64("post_id", draft.PostID),
				slog.String("error", err.Error()),
			)
		}
	}
}
