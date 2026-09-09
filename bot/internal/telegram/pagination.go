package telegram

import (
	"fmt"

	"github.com/go-telegram/bot/models"
)

func totalPages(total, pageSize int) int {
	if total == 0 {
		return 1
	}
	return (total + pageSize - 1) / pageSize
}

func clampPage(page, totalPages int) int {
	if page < 0 {
		return 0
	}
	if page >= totalPages {
		return totalPages - 1
	}
	return page
}

func buildPaginationRow(page, totalPages int, prevCallback, nextCallback string) []models.InlineKeyboardButton {
	return []models.InlineKeyboardButton{
		{Text: "⏪", CallbackData: prevCallback},
		{Text: fmt.Sprintf("%d/%d", page+1, totalPages), CallbackData: "noop"},
		{Text: "⏩", CallbackData: nextCallback},
	}
}
