package access

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/domain"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/render"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type userRefKind int

const (
	refInvalid userRefKind = iota
	refID
	refUsername
)

// ParseUserRef reads either a numeric Telegram ID or an @username (the @ is
// optional).
func ParseUserRef(input string) (kind userRefKind, id int64, username string) {
	s := strings.TrimSpace(input)
	if s == "" {
		return refInvalid, 0, ""
	}

	if strings.HasPrefix(s, "@") {
		name := domain.NormalizeUsername(s)
		if domain.ValidUsername(name) {
			return refUsername, 0, name
		}
		return refInvalid, 0, ""
	}

	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		if n <= 0 {
			return refInvalid, 0, ""
		}
		return refID, n, ""
	}

	name := domain.NormalizeUsername(s)
	if domain.ValidUsername(name) {
		return refUsername, 0, name
	}

	return refInvalid, 0, ""
}

// HandleInput turns the answer to the "add user" prompt into an allowed user.
func HandleInput(
	ctx context.Context,
	b *bot.Bot,
	st *store.Store,
	pending db.PendingReply,
	update *models.Update,
) {
	if !extract.IsOwner(ctx) {
		return
	}

	lang := extract.Lang(ctx)
	chatID := update.Message.Chat.ID
	promptID := pending.PromptMessageID

	kind, id, handle := ParseUserRef(update.Message.Text)
	switch kind {
	case refID:
		// getChat only knows users the bot has met; enrich when possible.
		username := ""
		if chat, err := b.GetChat(ctx, &bot.GetChatParams{ChatID: id}); err == nil {
			username = chat.Username
		}
		addUser(ctx, b, st, lang, chatID, promptID, pending.Origin, id, username)

	case refUsername:
		chat, err := b.GetChat(ctx, &bot.GetChatParams{ChatID: "@" + handle})
		if err != nil || chat.Type != models.ChatTypePrivate {
			// Telegram does not disclose user IDs by username, so ask for an
			// ID or a forwarded message instead.
			sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.UserAddNeedID)
			return
		}

		username := chat.Username
		if username == "" {
			username = handle
		}
		addUser(ctx, b, st, lang, chatID, promptID, pending.Origin, chat.ID, username)

	default:
		sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.UserAddInvalid)
	}
}

// HandleForward captures the sender of a forwarded message as the user to add,
// when an add-user prompt is pending in this chat. It reports whether the
// update was consumed.
func HandleForward(
	ctx context.Context,
	b *bot.Bot,
	st *store.Store,
	update *models.Update,
) bool {
	forward := update.Message.ForwardOrigin
	if forward == nil || forward.MessageOriginUser == nil {
		return false
	}

	sender := forward.MessageOriginUser.SenderUser
	if sender.ID == 0 {
		return false
	}

	chatID := update.Message.Chat.ID
	pending, err := st.GetLatestPendingReplyByAction(ctx, db.GetLatestPendingReplyByActionParams{
		ChatID: chatID, Action: domain.PendingActionAwaitingUserRef,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"get pending user prompt",
			slog.Int64("chat_id", chatID),
			slog.String("error", err.Error()),
		)
		return false
	}

	if !extract.IsOwner(ctx) {
		return true
	}

	lang := extract.Lang(ctx)
	slog.LogAttrs(
		ctx, slog.LevelInfo,
		"adding allowed user from a forwarded message",
		slog.Int64("user_id", sender.ID),
	)
	addUser(ctx, b, st, lang, chatID, pending.PromptMessageID, pending.Origin, sender.ID, sender.Username)

	return true
}

func addUser(
	ctx context.Context,
	b *bot.Bot,
	st *store.Store,
	lang i18n.Lang,
	chatID int64,
	promptID int64,
	origin string,
	userID int64,
	username string,
) {
	var name sql.NullString
	if username != "" {
		name = sql.NullString{String: username, Valid: true}
	}

	_, err := st.GetAllowedUser(ctx, userID)
	switch {
	case err == nil:
		forgetPrompt(ctx, st, chatID, promptID)
		sendNotice(ctx, b, lang, chatID, origin, i18n.UserAddDuplicate)
		return
	case !errors.Is(err, sql.ErrNoRows):
		slog.LogAttrs(
			ctx, slog.LevelError,
			"get allowed user",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		return
	}

	if err := st.AddAllowedUser(ctx, db.AddAllowedUserParams{
		UserID: userID, Username: name,
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"add allowed user",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		return
	}

	forgetPrompt(ctx, st, chatID, promptID)
	slog.LogAttrs(ctx, slog.LevelInfo, "allowed user added", slog.Int64("user_id", userID))
	sendNotice(ctx, b, lang, chatID, origin, i18n.UserAddSuccess, userLabel(lang, userID, name))
}

func sendNotice(
	ctx context.Context,
	b *bot.Bot,
	lang i18n.Lang,
	chatID int64,
	origin string,
	key i18n.Key,
	args ...any,
) {
	render.Notice(ctx, b, chatID, i18n.T(lang, key, args...), i18n.T(lang, i18n.BtnBack), origin)
}

func forgetPrompt(ctx context.Context, st *store.Store, chatID, promptID int64) {
	if err := st.DeletePendingReplyByPrompt(ctx, chatID, promptID); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"delete pending reply",
			slog.Int64("chat_id", chatID),
			slog.Int64("prompt_id", promptID),
			slog.String("error", err.Error()),
		)
	}
}
