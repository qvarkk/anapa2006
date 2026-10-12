package drafts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/moscow"
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
		renderDetail(ctx, b, st, update, postID, origin)
	}
}

// renderDetail renders a draft detail screen. origin is the callback to return
// to (a drafts list callback such as draft:la:<page>).
func renderDetail(ctx context.Context, b *bot.Bot, st *store.Store, update *models.Update, postID int64, origin string) {
	lang := extract.Lang(ctx)
	chatID, msgID := extract.CallbackTarget(update)
	thisCallback := callback.Format(callback.DraftDetail, postID, origin)

	draft, err := st.GetPostDraftByID(ctx, postID)
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "get post draft", slog.Int64("post_id", postID), slog.String("error", err.Error()))
		return
	}
	post, err := st.GetPost(ctx, postID)
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "get post", slog.Int64("post_id", postID), slog.String("error", err.Error()))
		return
	}

	schedule, schedErr := st.GetPostScheduleByID(ctx, postID)
	hasSchedule := schedErr == nil
	if schedErr != nil && !errors.Is(schedErr, sql.ErrNoRows) {
		slog.LogAttrs(ctx, slog.LevelError, "get schedule", slog.Int64("post_id", postID), slog.String("error", schedErr.Error()))
		return
	}

	mediaCounts := draftMediaCounts(ctx, st, postID)

	var target db.TargetChannel
	targetStr := i18n.T(lang, i18n.TargetNotSelected)
	if draft.TargetChannelID.Valid {
		if t, err := st.GetTargetChannelByID(ctx, draft.TargetChannelID.Int64); err == nil {
			target = t
			targetStr = targetName(t)
		}
	}

	text, rows := buildDetail(lang, draft, post, schedule, hasSchedule, mediaCounts, target, targetStr, thisCallback, origin)

	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID: msgID,
		ChatID:    chatID,
		Text:      text,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: rows,
		},
		ParseMode: models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to draft detail failed",
			slog.Int64("post_id", postID),
			slog.String("error", err.Error()),
		)
	}
}

func buildDetail(
	lang i18n.Lang,
	draft db.PostDraft,
	post db.Post,
	schedule db.Schedule,
	hasSchedule bool,
	mediaCounts map[string]int,
	target db.TargetChannel,
	targetStr string,
	thisCallback string,
	origin string,
) (string, [][]models.InlineKeyboardButton) {
	backText := i18n.T(lang, i18n.BtnBack)
	attachments := render.AttachmentsSummary(mediaCounts)

	targetButton := []models.InlineKeyboardButton{{
		Text:         i18n.T(lang, i18n.BtnDraftTarget, targetStr),
		CallbackData: callback.Format(callback.DraftTargetList, draft.PostID, 0, thisCallback),
	}}

	editTextButton := []models.InlineKeyboardButton{{
		Text:         i18n.T(lang, i18n.BtnDraftEditText),
		CallbackData: callback.Format(callback.DraftPromptEditText, draft.PostID, thisCallback),
	}}
	editMediaButton := []models.InlineKeyboardButton{{
		Text:         i18n.T(lang, i18n.BtnDraftEditMedia),
		CallbackData: callback.Format(callback.DraftPromptEditMedia, draft.PostID, thisCallback),
	}}
	deleteButton := []models.InlineKeyboardButton{{
		Text:         i18n.T(lang, i18n.BtnDraftDelete),
		CallbackData: callback.Format(callback.DraftPromptDelete, draft.PostID, thisCallback),
	}}
	showButton := []models.InlineKeyboardButton{{
		Text:         i18n.T(lang, i18n.BtnDraftShow),
		CallbackData: callback.Format(callback.DraftShow, draft.PostID, thisCallback),
	}}
	backButton := []models.InlineKeyboardButton{{Text: backText, CallbackData: origin}}

	scheduleButtons := func() [][]models.InlineKeyboardButton {
		if hasSchedule {
			return [][]models.InlineKeyboardButton{
				{{
					Text:         i18n.T(lang, i18n.BtnDraftRequeue),
					CallbackData: callback.Format(callback.DraftPromptQueue, draft.PostID, thisCallback),
				}},
				{{
					Text:         i18n.T(lang, i18n.BtnDraftDequeue),
					CallbackData: callback.Format(callback.DraftDequeue, draft.PostID, thisCallback),
				}},
			}
		}
		return [][]models.InlineKeyboardButton{
			{{
				Text:         i18n.T(lang, i18n.BtnDraftQueue),
				CallbackData: callback.Format(callback.DraftPromptQueue, draft.PostID, thisCallback),
			}},
		}
	}

	switch post.Status {
	case "sent":
		sentTime := i18n.T(lang, i18n.Unscheduled)
		if schedule.SentAt.Valid {
			sentTime = moscow.Format(schedule.SentAt.Time)
		}
		link := render.MessageLink(target.Handle.String, schedule.SentMessageID.Int64)
		text := i18n.T(lang, i18n.DraftDetailPosted,
			draft.PostID, sentTime, render.PostedLinkHTML(link), draft.FinalText, attachments,
		)
		var rows [][]models.InlineKeyboardButton
		if link != "" {
			rows = append(rows, []models.InlineKeyboardButton{{
				Text: i18n.T(lang, i18n.BtnOpenPosted), URL: link,
			}})
		}
		rows = append(rows, showButton, deleteButton, backButton)
		return text, rows

	case "error_sending":
		errText := schedule.LastError.String
		if errText == "" {
			errText = i18n.T(lang, i18n.DraftErrorUnknown)
		}
		scheduleTime := i18n.T(lang, i18n.Unscheduled)
		if hasSchedule {
			scheduleTime = moscow.Format(schedule.ScheduledAt)
		}
		text := i18n.T(lang, i18n.DraftDetailFailed,
			draft.PostID, errText, targetStr, scheduleTime, draft.FinalText, attachments,
		)
		rows := [][]models.InlineKeyboardButton{
			targetButton, editTextButton, editMediaButton,
		}
		rows = append(rows, scheduleButtons()...)
		rows = append(rows, deleteButton, showButton, backButton)
		return text, rows

	default:
		scheduleTime := i18n.T(lang, i18n.Unscheduled)
		if hasSchedule {
			scheduleTime = moscow.Format(schedule.ScheduledAt)
		}
		text := i18n.T(lang, i18n.DraftDetailActive,
			draft.PostID, targetStr, scheduleTime,
			render.PostStatusLabel(lang, post.Status), draft.FinalText, attachments,
		)
		if hasSchedule && !draft.TargetChannelID.Valid {
			text += "\n\n" + i18n.T(lang, i18n.DraftTargetWarning)
		}
		rows := [][]models.InlineKeyboardButton{
			targetButton, editTextButton, editMediaButton,
		}
		rows = append(rows, scheduleButtons()...)
		rows = append(rows, deleteButton, showButton, backButton)
		return text, rows
	}
}

func targetName(t db.TargetChannel) string {
	if t.Handle.Valid && t.Handle.String != "" {
		return "@" + t.Handle.String
	}
	if t.Title != "" {
		return fmt.Sprintf("%s (ID %d)", t.Title, t.ChatID)
	}
	return fmt.Sprintf("ID %d", t.ChatID)
}
