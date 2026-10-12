package access

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
	"qq/anapa2006/internal/telegram/pagination"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const RecordsPerPage = 5

func HandleList(st *store.Store, ownerID int64) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if !requireOwner(ctx, b, update) {
			return
		}
		callback.Ack(ctx, b, update)

		page := callback.ParseIntPart(update.CallbackQuery.Data, 2)
		renderList(ctx, b, st, update, page, ownerID)
	}
}

func renderList(
	ctx context.Context,
	b *bot.Bot,
	st *store.Store,
	update *models.Update,
	page int,
	ownerID int64,
) {
	lang := extract.Lang(ctx)

	total, err := st.CountAllowedUsers(ctx)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"count allowed users",
			slog.String("error", err.Error()),
		)
		return
	}

	totalPages := pagination.TotalPages(int(total), RecordsPerPage)
	page = pagination.ClampPage(page, totalPages)

	users, err := st.ListAllowedUsers(ctx, db.ListAllowedUsersParams{
		Limit: RecordsPerPage, Offset: int64(page * RecordsPerPage),
	})
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"list allowed users",
			slog.String("error", err.Error()),
		)
		return
	}

	listCallback := callback.Format(callback.AccessList, page)

	var sb strings.Builder
	sb.WriteString(i18n.T(lang, i18n.AccessListHeader, total))
	sb.WriteString("\n\n")

	rows := make([][]models.InlineKeyboardButton, 0, len(users)+3)
	for _, user := range users {
		role := i18n.T(lang, i18n.UserRoleAdmin)
		if user.UserID == ownerID {
			role = i18n.T(lang, i18n.UserRoleOwner)
		}

		sb.WriteString(i18n.T(lang, i18n.AccessListEntry, user.UserID, usernameLabel(lang, user.Username), role))
		sb.WriteString("\n\n")

		rows = append(rows, []models.InlineKeyboardButton{{
			Text:         fmt.Sprintf("👤 %s", userLabel(lang, user.UserID, user.Username)),
			CallbackData: callback.Format(callback.AccessUser, user.UserID, listCallback),
		}})
	}

	if totalPages > 1 {
		prev, next := callback.FirstPage, callback.LastPage
		if page > 0 {
			prev = callback.Format(callback.AccessList, page-1)
		}
		if page < totalPages-1 {
			next = callback.Format(callback.AccessList, page+1)
		}
		rows = append(rows, pagination.BuildPaginationRow(page, totalPages, prev, next))
	}

	rows = append(rows, []models.InlineKeyboardButton{
		{Text: i18n.T(lang, i18n.BtnUserAdd), CallbackData: callback.Format(callback.AccessAdd, listCallback)},
	})
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: i18n.T(lang, i18n.BtnBack), CallbackData: callback.AccessMenu},
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
		slog.LogAttrs(
			ctx, slog.LevelError,
			"edit message to access list failed",
			slog.String("error", err.Error()),
		)
	}
}

func usernameLabel(lang i18n.Lang, username sql.NullString) string {
	if username.Valid && username.String != "" {
		return "@" + username.String
	}
	return i18n.T(lang, i18n.NoUsername)
}

func userLabel(lang i18n.Lang, userID int64, username sql.NullString) string {
	if username.Valid && username.String != "" {
		return fmt.Sprintf("@%s (ID %d)", username.String, userID)
	}
	return fmt.Sprintf("ID %d", userID)
}
