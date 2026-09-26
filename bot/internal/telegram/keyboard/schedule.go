package keyboard

import (
	"fmt"
	"qq/anapa2006/internal/i18n"
	"qq/anapa2006/internal/telegram/callback"

	"github.com/go-telegram/bot/models"
)

func ScheduleKeyboard(draftID int64, origin string, lang i18n.Lang) *models.InlineKeyboardMarkup {
	mk := func(labelKey i18n.Key, duration string) models.InlineKeyboardButton {
		return models.InlineKeyboardButton{
			Text:         i18n.T(lang, labelKey),
			CallbackData: fmt.Sprintf(callback.ScheduleCreate, draftID, duration, origin),
		}
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{mk(i18n.ScheduleBtn30m, callback.In30m)},
		{mk(i18n.ScheduleBtn1hr, callback.In1hr)},
		{mk(i18n.ScheduleBtn3hr, callback.In3hr)},
		{mk(i18n.ScheduleBtn6hr, callback.In6hr)},
		{mk(i18n.ScheduleBtn12hr, callback.In12hr)},
		{{Text: i18n.T(lang, i18n.ScheduleBtnCustom), CallbackData: callback.Noop}},
		{{Text: i18n.T(lang, i18n.BtnBack), CallbackData: origin}},
	}}
}
