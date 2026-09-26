package render

import "qq/anapa2006/internal/i18n"

func PostStatusLabel(lang i18n.Lang, status string) string {
	switch status {
	case "new":
		return i18n.T(lang, i18n.PostStatusNew)
	case "skipped":
		return i18n.T(lang, i18n.PostStatusSkipped)
	case "scheduled":
		return i18n.T(lang, i18n.PostStatusScheduled)
	case "sent":
		return i18n.T(lang, i18n.PostStatusSent)
	default:
		return status
	}
}
