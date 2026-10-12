package drafts

import (
	"context"
	"database/sql"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/domain"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/callback"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/render"
	"qq/anapa2006/internal/telegram/sender"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const maxDraftMedia = 10

// HandleEditMediaPrompt is the "edit attachments" entry point from the draft
// detail screen (draft:p:e:m:<post_id>:<origin>).
func HandleEditMediaPrompt(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 4, 1)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "parse back navagation", slog.String("error", err.Error()))
			return
		}
		postID := data[0]
		renderMediaList(ctx, b, st, update, postID, origin)
	}
}

// HandleMediaList re-renders the media list after an in-place edit.
func HandleMediaList(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 2, 1)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "parse back navagation", slog.String("error", err.Error()))
			return
		}
		postID := data[0]
		renderMediaList(ctx, b, st, update, postID, origin)
	}
}

func renderMediaList(ctx context.Context, b *bot.Bot, st *store.Store, update *models.Update, postID int64, detailCB string) {
	lang := extract.Lang(ctx)

	media, err := st.ListDraftMedia(ctx, postID)
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "list draft media", slog.Int64("post_id", postID), slog.String("error", err.Error()))
		return
	}

	listCB := callback.Format(callback.DraftMediaList, postID, detailCB)

	var sb strings.Builder
	sb.WriteString(i18n.T(lang, i18n.DraftMediaMenu, len(media)))
	sb.WriteString("\n\n")

	var rows [][]models.InlineKeyboardButton
	for i, m := range media {
		sb.WriteString(i18n.T(lang, i18n.DraftMediaEntry, i+1, mediaKindLabel(m.Kind)))
		sb.WriteString("\n")
		rows = append(rows, []models.InlineKeyboardButton{{
			Text:         i18n.T(lang, i18n.BtnDraftMediaItem, i+1, mediaKindLabel(m.Kind)),
			CallbackData: callback.Format(callback.DraftMediaDetail, postID, m.ID, listCB),
		}})
	}

	rows = append(rows, []models.InlineKeyboardButton{{
		Text:         i18n.T(lang, i18n.BtnMediaAdd),
		CallbackData: callback.Format(callback.DraftMediaAddPrompt, postID, listCB),
	}})
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: i18n.T(lang, i18n.BtnBack), CallbackData: detailCB},
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
		slog.LogAttrs(ctx, slog.LevelError, "edit message to media list failed", slog.String("error", err.Error()))
	}
}

// HandleMediaDetail previews a single attachment and offers substitute/delete.
func HandleMediaDetail(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 3, 2)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "parse back navagation", slog.String("error", err.Error()))
			return
		}
		postID := data[0]
		mediaID := data[1]

		medium, err := st.GetDraftMediaByID(ctx, mediaID)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "get draft media by id", slog.Int64("media_id", mediaID), slog.String("error", err.Error()))
			return
		}

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)

		// Preview the attachment the user is about to change.
		s := sender.New(b)
		if _, err := s.SendMediaWithCaption(ctx, chatID, []db.DraftMedium{medium}, ""); err != nil {
			slog.LogAttrs(ctx, slog.LevelWarn, "send media preview", slog.Int64("media_id", mediaID), slog.String("error", err.Error()))
		}

		text := i18n.T(lang, i18n.DraftMediaDetail, mediaKindLabel(medium.Kind))
		kb := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{
				Text:         i18n.T(lang, i18n.BtnMediaSubstitute),
				CallbackData: callback.Format(callback.DraftMediaSubstitutePrompt, postID, mediaID, origin),
			}},
			{{
				Text:         i18n.T(lang, i18n.BtnMediaDelete),
				CallbackData: callback.Format(callback.DraftMediaDelete, postID, mediaID, origin),
			}},
			{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: origin}},
		}}

		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID, Text: text, ReplyMarkup: kb,
		}); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "edit message to media detail failed", slog.String("error", err.Error()))
		}
	}
}

// HandleMediaDelete removes one attachment and shifts subsequent positions.
func HandleMediaDelete(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 3, 2)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "parse back navagation", slog.String("error", err.Error()))
			return
		}
		postID := data[0]
		mediaID := data[1]

		if err := removeDraftMedia(ctx, st, postID, mediaID); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "remove draft media", slog.Int64("media_id", mediaID), slog.String("error", err.Error()))
			return
		}

		renderMediaList(ctx, b, st, update, postID, detailCallbackFromMediaList(origin))
	}
}

// HandleMediaSubstitutePrompt asks the user to send the replacement attachment.
func HandleMediaSubstitutePrompt(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 3, 2)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "parse back navagation", slog.String("error", err.Error()))
			return
		}
		postID := data[0]
		mediaID := data[1]

		lang := extract.Lang(ctx)
		chatID, _ := extract.CallbackTarget(update)

		sent, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   i18n.T(lang, i18n.MediaSubstitutePrompt),
			ReplyMarkup: &models.ForceReply{
				ForceReply: true, Selective: true,
				InputFieldPlaceholder: i18n.T(lang, i18n.MediaPlaceholder),
			},
		})
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "send media substitute prompt", slog.String("error", err.Error()))
			return
		}

		if err := st.CreatePendingReply(ctx, db.CreatePendingReplyParams{
			ChatID:          chatID,
			PromptMessageID: int64(sent.ID),
			Action:          domain.PendingActionAwaitingMediaSubstitute,
			PostID:          sql.NullInt64{Int64: postID, Valid: true},
			MediaID:         sql.NullInt64{Int64: mediaID, Valid: true},
			Origin:          origin,
		}); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "create pending reply", slog.String("error", err.Error()))
		}
	}
}

// HandleMediaAddPrompt asks for the insertion position of the new attachment.
func HandleMediaAddPrompt(st *store.Store) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback.Ack(ctx, b, update)

		data, origin, err := extract.BackNavigation(update, 2, 1)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "parse back navagation", slog.String("error", err.Error()))
			return
		}
		postID := data[0]

		lang := extract.Lang(ctx)
		chatID, _ := extract.CallbackTarget(update)

		sent, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   i18n.T(lang, i18n.MediaAddPositionPrompt),
			ReplyMarkup: &models.ForceReply{
				ForceReply: true, Selective: true,
				InputFieldPlaceholder: i18n.T(lang, i18n.MediaAddPositionHolder),
			},
		})
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "send media add position prompt", slog.String("error", err.Error()))
			return
		}

		if err := st.CreatePendingReply(ctx, db.CreatePendingReplyParams{
			ChatID:          chatID,
			PromptMessageID: int64(sent.ID),
			Action:          domain.PendingActionAwaitingMediaAddPosition,
			PostID:          sql.NullInt64{Int64: postID, Valid: true},
			Origin:          origin,
		}); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "create pending reply", slog.String("error", err.Error()))
		}
	}
}

// HandleMediaAddPosition parses the insertion position and then asks for the
// attachment itself.
func HandleMediaAddPosition(ctx context.Context, b *bot.Bot, st *store.Store, pending db.PendingReply, update *models.Update) {
	lang := extract.Lang(ctx)
	chatID := update.Message.Chat.ID
	promptID := pending.PromptMessageID
	postID := pending.PostID.Int64

	pos, err := strconv.Atoi(strings.TrimSpace(update.Message.Text))
	if err != nil || pos < 0 {
		render.Notice(ctx, b, chatID, i18n.T(lang, i18n.MediaAddInvalidPosition), i18n.T(lang, i18n.BtnBack), pending.Origin)
		return
	}

	count, err := st.CountDraftMedia(ctx, postID)
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "count draft media", slog.Int64("post_id", postID), slog.String("error", err.Error()))
		return
	}
	if count >= maxDraftMedia {
		forgetPending(ctx, st, chatID, promptID)
		render.Notice(ctx, b, chatID, i18n.T(lang, i18n.MediaLimitReached), i18n.T(lang, i18n.BtnBack), pending.Origin)
		return
	}
	if int64(pos) > count {
		pos = int(count)
	}

	// Replace the position prompt with the attachment prompt.
	forgetPending(ctx, st, chatID, promptID)

	sent, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   i18n.T(lang, i18n.MediaAddPrompt),
		ReplyMarkup: &models.ForceReply{
			ForceReply: true, Selective: true,
			InputFieldPlaceholder: i18n.T(lang, i18n.MediaPlaceholder),
		},
	})
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "send media add prompt", slog.String("error", err.Error()))
		return
	}

	if err := st.CreatePendingReply(ctx, db.CreatePendingReplyParams{
		ChatID:          chatID,
		PromptMessageID: int64(sent.ID),
		Action:          domain.PendingActionAwaitingMediaAdd,
		PostID:          sql.NullInt64{Int64: postID, Valid: true},
		Position:        sql.NullInt64{Int64: int64(pos), Valid: true},
		Origin:          pending.Origin,
	}); err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "create pending reply", slog.String("error", err.Error()))
	}
}

// HandleMediaAdd inserts the attachment at the recorded position.
func HandleMediaAdd(ctx context.Context, b *bot.Bot, st *store.Store, pending db.PendingReply, update *models.Update) {
	lang := extract.Lang(ctx)
	chatID := update.Message.Chat.ID
	promptID := pending.PromptMessageID
	postID := pending.PostID.Int64

	kind, fileID, ok := mediaFromMessage(update.Message)
	if !ok {
		render.Notice(ctx, b, chatID, i18n.T(lang, i18n.MediaUnsupported), i18n.T(lang, i18n.BtnBack), pending.Origin)
		return
	}

	pos := int(pending.Position.Int64)
	if err := insertDraftMedia(ctx, st, postID, pos, kind, fileID); err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "insert draft media", slog.Int64("post_id", postID), slog.String("error", err.Error()))
		return
	}

	forgetPending(ctx, st, chatID, promptID)
	render.Notice(ctx, b, chatID, i18n.T(lang, i18n.OperationSuccess), i18n.T(lang, i18n.BtnMediaBack), pending.Origin)
}

// HandleMediaSubstitute replaces the file of a single attachment.
func HandleMediaSubstitute(ctx context.Context, b *bot.Bot, st *store.Store, pending db.PendingReply, update *models.Update) {
	lang := extract.Lang(ctx)
	chatID := update.Message.Chat.ID
	promptID := pending.PromptMessageID
	mediaID := pending.MediaID.Int64

	kind, fileID, ok := mediaFromMessage(update.Message)
	if !ok {
		render.Notice(ctx, b, chatID, i18n.T(lang, i18n.MediaUnsupported), i18n.T(lang, i18n.BtnBack), pending.Origin)
		return
	}

	if err := st.UpdateDraftMedia(ctx, db.UpdateDraftMediaParams{
		ID: mediaID, Kind: kind, FileID: sql.NullString{String: fileID, Valid: true},
	}); err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "update draft media", slog.Int64("media_id", mediaID), slog.String("error", err.Error()))
		return
	}

	forgetPending(ctx, st, chatID, promptID)
	render.Notice(ctx, b, chatID, i18n.T(lang, i18n.OperationSuccess), i18n.T(lang, i18n.BtnMediaBack), pending.Origin)
}

func mediaFromMessage(msg *models.Message) (kind string, fileID string, ok bool) {
	switch {
	case len(msg.Photo) > 0:
		p := msg.Photo[len(msg.Photo)-1]
		return string(domain.MediaKindPhoto), p.FileID, true
	case msg.Video != nil:
		return string(domain.MediaKindVideo), msg.Video.FileID, true
	case msg.Document != nil:
		return string(domain.MediaKindDocument), msg.Document.FileID, true
	default:
		return "", "", false
	}
}

func mediaKindLabel(kind string) string {
	switch kind {
	case "photo":
		return "📷 Фото"
	case "video":
		return "🎥 Видео"
	case "document":
		return "📄 Документ"
	default:
		return kind
	}
}

func removeDraftMedia(ctx context.Context, st *store.Store, postID, mediaID int64) error {
	media, err := st.ListDraftMedia(ctx, postID)
	if err != nil {
		return err
	}

	keep := make([]db.CreateDraftMediaParams, 0, len(media))
	for _, m := range media {
		if m.ID == mediaID {
			continue
		}
		keep = append(keep, db.CreateDraftMediaParams{
			DraftID:       postID,
			Kind:          m.Kind,
			OriginMediaID: m.OriginMediaID,
			FileID:        m.FileID,
			Url:           m.Url,
		})
	}
	return rewriteDraftMedia(ctx, st, postID, keep)
}

func insertDraftMedia(ctx context.Context, st *store.Store, postID int64, pos int, kind, fileID string) error {
	media, err := st.ListDraftMedia(ctx, postID)
	if err != nil {
		return err
	}

	if pos > len(media) {
		pos = len(media)
	}

	items := make([]db.CreateDraftMediaParams, 0, len(media)+1)
	for i, m := range media {
		if i == pos {
			items = append(items, db.CreateDraftMediaParams{
				DraftID: postID, Kind: kind,
				FileID: sql.NullString{String: fileID, Valid: true},
			})
		}
		items = append(items, db.CreateDraftMediaParams{
			DraftID:       postID,
			Kind:          m.Kind,
			OriginMediaID: m.OriginMediaID,
			FileID:        m.FileID,
			Url:           m.Url,
		})
	}
	if pos >= len(media) {
		items = append(items, db.CreateDraftMediaParams{
			DraftID: postID, Kind: kind,
			FileID: sql.NullString{String: fileID, Valid: true},
		})
	}
	return rewriteDraftMedia(ctx, st, postID, items)
}

// rewriteDraftMedia replaces the whole attachment list in one transaction and
// reindexes positions, avoiding UNIQUE(draft_id, position) conflicts.
func rewriteDraftMedia(ctx context.Context, st *store.Store, postID int64, items []db.CreateDraftMediaParams) error {
	return st.WithTx(ctx, func(q *db.Queries) error {
		if err := q.DeleteDraftMediaByDraft(ctx, postID); err != nil {
			return err
		}
		for i := range items {
			items[i].DraftID = postID
			items[i].Position = int64(i)
			if err := q.CreateDraftMedia(ctx, items[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func detailCallbackFromMediaList(listCallback string) string {
	parts := strings.SplitN(listCallback, ":", 4)
	if len(parts) < 4 {
		return callback.Start
	}
	return parts[3]
}

func forgetPending(ctx context.Context, st *store.Store, chatID, promptID int64) {
	if err := st.DeletePendingReplyByPrompt(ctx, chatID, promptID); err != nil {
		slog.LogAttrs(ctx, slog.LevelWarn, "delete pending reply", slog.Int64("chat_id", chatID), slog.Int64("prompt_id", promptID), slog.String("error", err.Error()))
	}
}
