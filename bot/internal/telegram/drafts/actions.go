package drafts

import (
	"context"
	"database/sql"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/moscow"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/pagination"
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
			slog.LogAttrs(ctx, slog.LevelError, "parse back navagation", slog.String("error", err.Error()))
			return
		}
		postID := data[0]

		post, media, err := postWithMediaByID(ctx, st, postID)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "get post with media by id", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return
		}

		draft, err := createDraftWithMediaTx(ctx, st, post, media, post.RawText)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "create draft with media", slog.Int64("post_id", post.ID), slog.String("error", err.Error()))
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
					{{Text: i18n.T(lang, i18n.BtnDraftView), CallbackData: callback.Format(callback.DraftDetail, draft.PostID, origin)}},
					{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: origin}},
				},
			},
			ParseMode: models.ParseModeHTML,
			LinkPreviewOptions: &models.LinkPreviewOptions{
				IsDisabled: bot.True(),
			},
		}); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "edit message to draft create failed", slog.Int64("post_id", draft.PostID), slog.String("error", err.Error()))
		}
	}
}

func HandleQueue(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		parts := strings.SplitN(update.CallbackQuery.Data, ":", 5)
		if len(parts) != 5 {
			slog.LogAttrs(ctx, slog.LevelError, "malformed schedule callback", slog.String("data", update.CallbackQuery.Data))
			return
		}
		postID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "bad draft id in schedule callback", slog.String("data", update.CallbackQuery.Data))
			return
		}
		dur, err := time.ParseDuration(parts[3])
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "bad duration in schedule callback", slog.String("raw", parts[3]), slog.String("error", err.Error()))
			return
		}
		origin := parts[4]

		draft, err := st.GetPostDraftByID(ctx, postID)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "get post draft", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return
		}
		if !draft.TargetChannelID.Valid {
			showSelectTargetNotice(ctx, b, update, postID, origin)
			return
		}

		scheduledAt := time.Now().Add(dur)

		_ = st.Deschedule(ctx, postID)
		if err := st.CreateSchedule(ctx, db.CreateScheduleParams{
			PostID: postID, ScheduledAt: scheduledAt,
		}); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "create schedule", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return
		}

		if err := st.MarkPostScheduled(ctx, postID); err != nil {
			slog.LogAttrs(ctx, slog.LevelWarn, "mark post scheduled", slog.Int64("post_id", postID), slog.String("error", err.Error()))
		}

		slog.LogAttrs(ctx, slog.LevelInfo, "post was scheduled", slog.Int64("post_id", postID), slog.String("scheduled_at", scheduledAt.Format("02.01.2006 15:04")))

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: msgID,
			Text:      i18n.T(lang, i18n.ScheduledAt, moscow.Format(scheduledAt)),
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: origin}},
				}},
		})
	}
}

func HandleDequeue(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 2, 1)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "parse back navagation", slog.String("error", err.Error()))
			return
		}
		postID := data[0]

		if err := st.Deschedule(ctx, postID); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "dequeue draft", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return
		}
		// Unscheduling returns the post to a plain "new" state.
		if err := st.UnhidePost(ctx, postID); err != nil {
			slog.LogAttrs(ctx, slog.LevelWarn, "reset post status on dequeue", slog.Int64("post_id", postID), slog.String("error", err.Error()))
		}

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)

		editActionResultMessage(ctx, b, actionResultMessagePayload{
			MessageID:      msgID,
			ChatID:         chatID,
			MessageText:    i18n.T(lang, i18n.OperationSuccess),
			ReturnText:     i18n.T(lang, i18n.BtnBack),
			ReturnCallback: origin,
		})
	}
}

func HandleDelete(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 2, 1)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "parse back navagation", slog.String("error", err.Error()))
			return
		}
		postID := data[0]

		if err := st.DeleteDraft(ctx, postID); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "delete draft", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return
		}

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)

		editActionResultMessage(ctx, b, actionResultMessagePayload{
			MessageID:      msgID,
			ChatID:         chatID,
			MessageText:    i18n.T(lang, i18n.OperationSuccess),
			ReturnText:     i18n.T(lang, i18n.BtnBack),
			ReturnCallback: origin,
		})
	}
}

func HandleShow(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 2, 1)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "parse back navagation", slog.String("error", err.Error()))
			return
		}
		postID := data[0]

		draft, err := st.GetPostDraftByID(ctx, postID)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "get post draft by id", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return
		}
		media, err := st.ListDraftMedia(ctx, postID)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "get draft media", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return
		}

		lang := extract.Lang(ctx)
		chatID, _ := extract.CallbackTarget(update)

		s := sender.New(b)
		if len(media) > 0 {
			_, err = s.SendMediaWithCaption(ctx, chatID, media, draft.FinalText)
		} else {
			_, err = s.SendMessage(ctx, chatID, draft.FinalText)
		}
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "send draft preview", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   i18n.T(lang, i18n.OperationSuccess),
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: origin}},
				},
			},
		})
	}
}

// HandleTargetList renders the target channel picker for a draft.
func HandleTargetList(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 2, 2)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "parse back navagation", slog.String("error", err.Error()))
			return
		}
		postID := data[0]
		page := int(data[1])
		renderTargetPicker(ctx, b, st, update, postID, page, origin)
	}
}

func renderTargetPicker(ctx context.Context, b *bot.Bot, st *store.Store, update *models.Update, postID int64, page int, origin string) {
	lang := extract.Lang(ctx)

	total, err := st.CountTargetChannels(ctx)
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "count target channels", slog.String("error", err.Error()))
		return
	}
	tp := pagination.TotalPages(int(total), RecordsPerPage)
	page = pagination.ClampPage(page, tp)

	targets, err := st.ListTargetChannels(ctx, db.ListTargetChannelsParams{
		Limit: RecordsPerPage, Offset: int64(page * RecordsPerPage),
	})
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "list target channels", slog.String("error", err.Error()))
		return
	}

	var rows [][]models.InlineKeyboardButton
	for _, t := range targets {
		rows = append(rows, []models.InlineKeyboardButton{{
			Text:         targetName(t),
			CallbackData: callback.Format(callback.DraftTargetPick, postID, t.ID, origin),
		}})
	}
	rows = append(rows, []models.InlineKeyboardButton{{
		Text:         i18n.T(lang, i18n.BtnDraftTargetClear),
		CallbackData: callback.Format(callback.DraftTargetClear, postID, origin),
	}})

	if tp > 1 {
		prev, next := callback.FirstPage, callback.LastPage
		if page > 0 {
			prev = callback.Format(callback.DraftTargetList, postID, page-1, origin)
		}
		if page < tp-1 {
			next = callback.Format(callback.DraftTargetList, postID, page+1, origin)
		}
		rows = append(rows, pagination.BuildPaginationRow(page, tp, prev, next))
	}

	rows = append(rows, []models.InlineKeyboardButton{
		{Text: i18n.T(lang, i18n.BtnBack), CallbackData: origin},
	})

	chatID, msgID := extract.CallbackTarget(update)
	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID,
		Text:        i18n.T(lang, i18n.DraftTargetPrompt),
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
	}); err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "edit message to target picker failed", slog.String("error", err.Error()))
	}
}

func HandleTargetPick(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 2, 2)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "parse back navagation", slog.String("error", err.Error()))
			return
		}
		postID := data[0]
		targetID := data[1]

		if err := st.SetDraftTargetChannel(ctx, db.SetDraftTargetChannelParams{
			PostID: postID, TargetChannelID: sql.NullInt64{Int64: targetID, Valid: true},
		}); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "set draft target channel", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return
		}

		renderDetail(ctx, b, st, update, postID, listCallbackFromDetail(origin))
	}
}

func HandleTargetClear(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 2, 1)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "parse back navagation", slog.String("error", err.Error()))
			return
		}
		postID := data[0]

		if err := st.SetDraftTargetChannel(ctx, db.SetDraftTargetChannelParams{
			PostID: postID, TargetChannelID: sql.NullInt64{},
		}); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "clear draft target channel", slog.Int64("post_id", postID), slog.String("error", err.Error()))
			return
		}

		renderDetail(ctx, b, st, update, postID, listCallbackFromDetail(origin))
	}
}

func listCallbackFromDetail(detailCallback string) string {
	parts := strings.SplitN(detailCallback, ":", 4)
	if len(parts) < 4 {
		return callback.Format(callback.DraftsListActive, 0)
	}
	return parts[3]
}

func showSelectTargetNotice(ctx context.Context, b *bot.Bot, update *models.Update, postID int64, origin string) {
	lang := extract.Lang(ctx)
	chatID, msgID := extract.CallbackTarget(update)

	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: chatID, MessageID: msgID,
		Text: i18n.T(lang, i18n.DraftNeedTarget),
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: i18n.T(lang, i18n.BtnDraftSelectTarget), CallbackData: callback.Format(callback.DraftTargetList, postID, 0, origin)}},
			{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: origin}},
		}},
	}); err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "edit message to select target notice failed", slog.String("error", err.Error()))
	}
}

func editActionResultMessage(ctx context.Context, b *bot.Bot, payload actionResultMessagePayload) {
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID: payload.MessageID,
		ChatID:    payload.ChatID,
		Text:      payload.MessageText,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: payload.ReturnText, CallbackData: payload.ReturnCallback}},
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
	st *store.Store,
	post *db.Post,
	media []db.PostMedium,
	draftText string,
) (*db.PostDraft, error) {
	tx, err := st.Conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := st.Queries.WithTx(tx)

	draft, err := qtx.CreatePostDraft(ctx, db.CreatePostDraftParams{
		PostID: post.ID, FinalText: draftText,
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
