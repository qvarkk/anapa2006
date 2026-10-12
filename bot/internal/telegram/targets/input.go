package targets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/domain"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/store"
	"qq/anapa2006/internal/telegram/extract"
	"qq/anapa2006/internal/telegram/render"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var (
	errTargetNotFound   = errors.New("target channel not found")
	errTargetNotChannel = errors.New("target is not a public channel")
)

// HandleInput turns the answer to the "add target channel" prompt into a target.
func HandleInput(ctx context.Context, b *bot.Bot, st *store.Store, pending db.PendingReply, update *models.Update) {
	lang := extract.Lang(ctx)
	chatID := update.Message.Chat.ID
	promptID := pending.PromptMessageID
	handle := domain.NormalizeUsername(update.Message.Text)

	if !domain.ValidUsername(handle) {
		sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.TargetAddInvalid)
		return
	}

	_, err := st.GetTargetChannelByHandle(ctx, nullHandle(handle))
	switch {
	case err == nil:
		forgetPrompt(ctx, st, chatID, promptID)
		sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.TargetAddDuplicate, handle)
		return
	case !errors.Is(err, sql.ErrNoRows):
		slog.LogAttrs(
			ctx, slog.LevelError,
			"get target channel by handle",
			slog.String("handle", handle),
			slog.String("error", err.Error()),
		)
		return
	}

	chat, isAdmin, err := verifyTarget(ctx, b, handle)
	switch {
	case errors.Is(err, errTargetNotChannel):
		sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.TargetAddNotChannel, handle)
		return
	case errors.Is(err, errTargetNotFound):
		sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.TargetAddNotFound, handle)
		return
	case err != nil:
		// Inconclusive (network, rate limit, ...): keep the prompt pending.
		sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.TargetAddInconclusive, handle)
		return
	}

	username := ""
	if chat.Username != "" {
		username = domain.NormalizeUsername(chat.Username)
	}
	title := handle
	if chat.Title != "" {
		title = chat.Title
	}

	target, err := st.CreateTargetChannel(ctx, db.CreateTargetChannelParams{
		ChatID: chat.ID, Handle: nullHandle(username), Title: title,
	})
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"create target channel",
			slog.String("handle", handle),
			slog.String("error", err.Error()),
		)
		return
	}

	forgetPrompt(ctx, st, chatID, promptID)

	slog.LogAttrs(ctx, slog.LevelInfo, "target channel added", slog.Int64("target_id", target.ID), slog.String("handle", handle))

	if !isAdmin {
		sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.TargetAddNoAdmin, handle)
		return
	}
	sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.TargetAddSuccess, handle)
}

// verifyTarget asks Telegram about the handle and whether the bot can post to
// it. A definitive "not found" or non-channel result rejects it; a missing
// admin right is reported back but the channel is still accepted.
func verifyTarget(ctx context.Context, b *bot.Bot, handle string) (*models.ChatFullInfo, bool, error) {
	chat, err := b.GetChat(ctx, &bot.GetChatParams{ChatID: "@" + handle})
	if err != nil {
		if errors.Is(err, bot.ErrorNotFound) || errors.Is(err, bot.ErrorBadRequest) {
			return nil, false, errTargetNotFound
		}
		return nil, false, fmt.Errorf("get chat: %w", err)
	}

	if chat.Type != models.ChatTypeChannel && chat.Type != models.ChatTypeSupergroup {
		return nil, false, errTargetNotChannel
	}

	// Best-effort: verify the bot is an admin so posts can actually be sent.
	me, err := b.GetMe(ctx)
	if err != nil {
		return chat, false, nil
	}
	member, err := b.GetChatMember(ctx, &bot.GetChatMemberParams{ChatID: chat.ID, UserID: me.ID})
	if err != nil {
		return chat, false, nil
	}
	isAdmin := member.Type == models.ChatMemberTypeOwner || member.Type == models.ChatMemberTypeAdministrator
	return chat, isAdmin, nil
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
