package access

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

func HandleDeletePrompt(st *store.Store, ownerID int64) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if !requireOwner(ctx, b, update) {
			return
		}
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
		userID := data[0]

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)
		backText := i18n.T(lang, i18n.BtnBack)

		if userID == ownerID {
			render.EditNotice(
				ctx, b, chatID, msgID,
				i18n.T(lang, i18n.UserDeleteOwnerForbidden), backText, origin,
			)
			return
		}

		user, err := st.GetAllowedUser(ctx, userID)
		if errors.Is(err, sql.ErrNoRows) {
			render.EditNotice(
				ctx, b, chatID, msgID,
				i18n.T(lang, i18n.OperationSuccess), backText, origin,
			)
			return
		}
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"get allowed user",
				slog.Int64("user_id", userID),
				slog.String("error", err.Error()),
			)
			return
		}

		// origin is the list callback, so its second component is the page.
		page := callback.ParseIntPart(origin, 2)

		if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID,
			Text: i18n.T(lang, i18n.UserDeletePrompt, userLabel(lang, user.UserID, user.Username)),
			ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
				{{
					Text:         i18n.T(lang, i18n.BtnDelete),
					CallbackData: callback.Format(callback.AccessDelete, userID, page),
				}},
				{{Text: backText, CallbackData: origin}},
			}},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"edit message to user delete prompt failed",
				slog.Int64("user_id", userID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func HandleDelete(st *store.Store, ownerID int64) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if !requireOwner(ctx, b, update) {
			return
		}
		callback.Ack(ctx, b, update)

		userID := int64(callback.ParseIntPart(update.CallbackQuery.Data, 2))
		page := callback.ParseIntPart(update.CallbackQuery.Data, 3)

		lang := extract.Lang(ctx)
		chatID, msgID := extract.CallbackTarget(update)
		backText := i18n.T(lang, i18n.BtnBack)

		if userID == ownerID {
			render.EditNotice(
				ctx, b, chatID, msgID,
				i18n.T(lang, i18n.UserDeleteOwnerForbidden),
				backText, callback.Format(callback.AccessList, page),
			)
			return
		}

		if err := st.DeleteAllowedUser(ctx, userID); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"delete allowed user",
				slog.Int64("user_id", userID),
				slog.String("error", err.Error()),
			)
			return
		}

		slog.LogAttrs(ctx, slog.LevelInfo, "allowed user removed", slog.Int64("user_id", userID))
		renderList(ctx, b, st, update, page, ownerID)
	}
}
