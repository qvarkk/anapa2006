package planner

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/keyboard"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func HandleScheduleUse(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)
		lang := extract.Lang(ctx)

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

		post, media, err := postWithMediaByID(ctx, st, postID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get post with media by id",
				slog.Int64("post_id", post.ID),
				slog.String("error", err.Error()),
			)
			return
		}

		draft, err := createDraftWithMediaTx(ctx, update, st, post, media, post.RawText)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"create draft with media",
				slog.Int64("post_id", post.ID),
				slog.String("error", err.Error()),
			)
			return
		}

		kb := keyboard.ScheduleKeyboard(draft.ID, origin, lang)
		chatID, msgID := extract.CallbackTarget(update)

		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID: msgID, ChatID: chatID, Text: i18n.T(lang, i18n.SchedulePrompt, draft.FinalText),
			ReplyMarkup: kb, ParseMode: models.ParseModeHTML,
			LinkPreviewOptions: &models.LinkPreviewOptions{
				IsDisabled: bot.True(),
			},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to schedule use failed",
				slog.Int("message_id", msgID),
				slog.Int64("chat_id", chatID),
				slog.Int64("draft_id", draft.ID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func HandleScheduleEdit(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)
		lang := extract.Lang(ctx)

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

		post, media, err := postWithMediaByID(ctx, st, postID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get post with media by id",
				slog.Int64("post_id", post.ID),
				slog.String("error", err.Error()),
			)
			return
		}

		draft, err := createDraftWithMediaTx(ctx, update, st, post, media, post.RawText)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"create draft with media",
				slog.Int64("post_id", post.ID),
				slog.String("error", err.Error()),
			)
			return
		}

		chatID, _ := extract.CallbackTarget(update)

		sent, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      i18n.T(lang, i18n.EditPrompt, draft.FinalText),
			ParseMode: models.ParseModeHTML,
			ReplyMarkup: &models.ForceReply{
				ForceReply:            true,
				Selective:             true,
				InputFieldPlaceholder: i18n.T(lang, i18n.EditPlaceholder),
			},
		})
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"send edit prompt",
				slog.Int64("draft_id", draft.ID),
				slog.String("error", err.Error()),
			)
			return
		}

		if err := st.CreatePendingReply(ctx, db.CreatePendingReplyParams{
			ChatID:          chatID,
			PromptMessageID: int64(sent.ID),
			Action:          "awaiting_text",
			DraftID:         draft.ID,
			Origin:          origin,
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"create pending reply",
				slog.Int64("draft_id", draft.ID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func HandleScheduleSkip(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)
		lang := extract.Lang(ctx)

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

		if err := st.SkipPost(ctx, postID); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"skip post",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}

		chatID, msgID := extract.CallbackTarget(update)

		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID,
			Text: i18n.T(lang, i18n.ScheduleSkipped),
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: origin}},
				}},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to schedule skip failed",
				slog.Int("message_id", msgID),
				slog.Int64("chat_id", chatID),
				slog.String("error", err.Error()),
			)
			return
		}
	}
}

func HandleScheduleCreate(st *store.Store, channelID int64) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		lang := extract.Lang(ctx)

		parts := strings.SplitN(update.CallbackQuery.Data, ":", 5)
		if len(parts) != 5 {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"malformed schedule callback",
				slog.String("data", update.CallbackQuery.Data),
			)
			return
		}
		draftID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"bad draft id in schedule callback",
				slog.String("data", update.CallbackQuery.Data),
			)
			return
		}
		dur, err := time.ParseDuration(parts[3])
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"bad duration in schedule callback",
				slog.String("raw", parts[3]),
				slog.String("error", err.Error()),
			)
			return
		}
		origin := parts[4]

		scheduledAt := time.Now().Add(dur)

		if err := st.CreateSchedule(ctx, db.CreateScheduleParams{
			DraftID: draftID, TargetChatID: channelID, ScheduledAt: scheduledAt,
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"create schedule",
				slog.Int64("draft_id", draftID),
				slog.String("error", err.Error()),
			)
			return
		}

		postID, err := st.GetDraftsPostID(ctx, draftID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"get drafts post id",
				slog.Int64("draft_id", draftID),
				slog.String("error", err.Error()),
			)
		}

		if err := st.MarkPostScheduled(ctx, postID); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"mark post scheduled",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
		}

		slog.LogAttrs(
			ctx, slog.LevelInfo,
			"post was scheduled",
			slog.Int64("draft_id", draftID),
			slog.Int64("channel_id", channelID),
			slog.String("scheduled_at", scheduledAt.Format("02.01.2006 15:04")),
		)

		chatID, msgID := extract.CallbackTarget(update)

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID,
			Text: i18n.T(lang, i18n.ScheduledAt, scheduledAt.Format("02.01.2006 15:04")),
			ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: origin}},
			}},
		})
	}
}

func postWithMediaByID(
	ctx context.Context,
	st *store.Store,
	postID int64,
) (*db.Post, []db.PostMedium, error) {
	post, err := st.GetPost(ctx, postID)
	if err != nil {
		return nil, nil, err
	}

	media, err := st.ListPostMedia(ctx, post.ID)
	if err != nil {
		return nil, nil, err
	}

	return &post, media, nil
}

func createDraftWithMediaTx(
	ctx context.Context,
	update *models.Update,
	st *store.Store,
	post *db.Post,
	media []db.PostMedium,
	draftText string,
) (*db.Draft, error) {
	userID, _, ok := extract.Identity(update)
	if !ok {
		return nil, fmt.Errorf("extract identity")
	}

	tx, err := st.Conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := st.Queries.WithTx(tx)

	draft, err := qtx.CreateDraft(ctx, db.CreateDraftParams{
		PostID: post.ID, FinalText: draftText, UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	draftMedia := make([]db.CreateDraftMediaParams, len(media))
	for i, medium := range media {
		draftMedia[i] = db.CreateDraftMediaParams{
			DraftID:       draft.ID,
			Kind:          medium.Kind,
			OriginMediaID: medium.ID,
			FileID:        medium.FileID,
			Url:           sql.NullString{String: medium.Url, Valid: true},
			Position:      medium.Position,
		}
	}

	for _, medium := range draftMedia {
		if err := qtx.CreateDraftMedia(ctx, medium); err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &draft, nil
}
