package channels

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
	errChannelNotFound = errors.New("channel not found")
	errNotAChannel     = errors.New("not a public channel")
)

// HandleInput turns the answer to the "add channel" prompt into a source.
// Invalid or unverifiable input keeps the prompt pending, so the answer can be
// repeated or the user can go back.
func HandleInput(ctx context.Context, b *bot.Bot, st *store.Store, pending db.PendingReply, update *models.Update) {
	lang := extract.Lang(ctx)
	chatID := update.Message.Chat.ID
	promptID := pending.PromptMessageID
	handle := domain.NormalizeUsername(update.Message.Text)

	if !domain.ValidUsername(handle) {
		sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.ChannelAddInvalid)
		return
	}

	_, err := st.GetSourceByHandle(ctx, handle)
	switch {
	case err == nil:
		forgetPrompt(ctx, st, chatID, promptID)
		sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.ChannelAddDuplicate, handle)
		return
	case !errors.Is(err, sql.ErrNoRows):
		slog.LogAttrs(
			ctx, slog.LevelError,
			"get source by handle",
			slog.String("handle", handle),
			slog.String("error", err.Error()),
		)
		return
	}

	switch err := verifyChannel(ctx, b, handle); {
	case errors.Is(err, errNotAChannel):
		sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.ChannelAddNotChannel, handle)
		return
	case errors.Is(err, errChannelNotFound):
		sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.ChannelAddNotFound, handle)
		return
	case err != nil:
		// Inconclusive (network, rate limit, ...): accept the handle anyway.
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"channel verification inconclusive, adding anyway",
			slog.String("handle", handle),
			slog.String("error", err.Error()),
		)
	}

	if _, err := st.CreateSource(ctx, db.CreateSourceParams{
		ChannelHandle: handle, Active: 1,
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"create source",
			slog.String("handle", handle),
			slog.String("error", err.Error()),
		)
		return
	}

	forgetPrompt(ctx, st, chatID, promptID)
	sendNotice(ctx, b, lang, chatID, pending.Origin, i18n.ChannelAddSuccess, handle)
}

// verifyChannel asks Telegram about the handle. A definitive "not found" or a
// non-channel result rejects it; anything else is inconclusive.
func verifyChannel(ctx context.Context, b *bot.Bot, handle string) error {
	chat, err := b.GetChat(ctx, &bot.GetChatParams{ChatID: "@" + handle})
	if err != nil {
		if errors.Is(err, bot.ErrorNotFound) || errors.Is(err, bot.ErrorBadRequest) {
			return errChannelNotFound
		}
		return fmt.Errorf("get chat: %w", err)
	}

	if chat.Type != models.ChatTypeChannel && chat.Type != models.ChatTypeSupergroup {
		return errNotAChannel
	}

	return nil
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
