package drafts

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
	"qq/anapa2006/internal/telegram/sender"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type actionResultMessagePayload struct {
	MessageID int
	ChatID    int64

	MessageText string

	ReturnText     string
	ReturnCallback string
}

func HandleCreate(st *store.Store) bot.HandlerFunc {
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

		post, media, err := postWithMediaByID(ctx, st, postID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get post with media by id",
				slog.Int64("post_id", postID),
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

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)

		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID: msgID,
			ChatID:    chatID,
			Text:      i18n.T(lang, i18n.DraftCreated, draft.PostID),
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						{
							Text:         i18n.T(lang, i18n.BtnDraftView),
							CallbackData: callback.Format(callback.DraftDetail, draft.PostID, origin),
						},
					},
					{
						{
							Text:         i18n.T(lang, i18n.BtnBack),
							CallbackData: origin,
						},
					},
				},
			},
			ParseMode: models.ParseModeHTML,
			LinkPreviewOptions: &models.LinkPreviewOptions{
				IsDisabled: bot.True(),
			},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to draft create failed",
				slog.Int64("post_id", draft.PostID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func HandleQueue(st *store.Store, channelID int64) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		parts := strings.SplitN(update.CallbackQuery.Data, ":", 5)
		if len(parts) != 5 {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"malformed schedule callback",
				slog.String("data", update.CallbackQuery.Data),
			)
			return
		}
		postID, err := strconv.ParseInt(parts[2], 10, 64)
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

		_ = st.Deschedule(ctx, postID)
		if err := st.CreateSchedule(ctx, db.CreateScheduleParams{
			PostID: postID, TargetChatID: channelID, ScheduledAt: scheduledAt,
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"create schedule",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
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
			slog.Int64("post_id", postID),
			slog.Int64("channel_id", channelID),
			slog.String("scheduled_at", scheduledAt.Format("02.01.2006 15:04")),
		)

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: msgID,
			Text:      i18n.T(lang, i18n.ScheduledAt, scheduledAt.Format("02.01.2006 15:04")),
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						{
							Text:         i18n.T(lang, i18n.BtnBack),
							CallbackData: origin,
						},
					},
				}},
		})
	}
}

func HandleDequeue(st *store.Store) bot.HandlerFunc {
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

		if err := st.Deschedule(ctx, postID); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"dequeue draft",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)

		payload := actionResultMessagePayload{
			MessageID:      msgID,
			ChatID:         chatID,
			MessageText:    i18n.T(lang, i18n.OperationSuccess),
			ReturnText:     i18n.T(lang, i18n.BtnBack),
			ReturnCallback: origin,
		}

		editActionResultMessage(ctx, b, payload)
	}
}

func HandleDelete(st *store.Store) bot.HandlerFunc {
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

		if err := st.DeleteDraft(ctx, postID); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"delete draft",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)

		payload := actionResultMessagePayload{
			MessageID:      msgID,
			ChatID:         chatID,
			MessageText:    i18n.T(lang, i18n.OperationSuccess),
			ReturnText:     i18n.T(lang, i18n.BtnBack),
			ReturnCallback: origin,
		}

		editActionResultMessage(ctx, b, payload)
	}
}

func HandleShow(st *store.Store) bot.HandlerFunc {
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
				"get post draft by id",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}
		media, err := st.ListDraftMedia(ctx, postID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get draft media",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}

		lang := extract.Lang(ctx)
		chatID, _ := extract.CallbackTarget(update)

		sender := sender.New(b)

		if len(media) > 0 {
			err = sender.SendMediaWithCaption(ctx, chatID, media, draft.FinalText)
		} else {
			err = sender.SendMessage(ctx, chatID, draft.FinalText)
		}
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"send draft preview",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			return
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   i18n.T(lang, i18n.OperationSuccess),
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
		})
	}
}

func editActionResultMessage(ctx context.Context, b *bot.Bot, payload actionResultMessagePayload) {
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID: payload.MessageID,
		ChatID:    payload.ChatID,
		Text:      payload.MessageText,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{
						Text:         payload.ReturnText,
						CallbackData: payload.ReturnCallback,
					},
				},
			},
		},
	})
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
) (*db.PostDraft, error) {
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

	draft, err := qtx.CreatePostDraft(ctx, db.CreatePostDraftParams{
		PostID: post.ID, FinalText: draftText, UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	draftMedia := make([]db.CreateDraftMediaParams, len(media))
	for i, medium := range media {
		draftMedia[i] = db.CreateDraftMediaParams{
			DraftID:       draft.PostID,
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
