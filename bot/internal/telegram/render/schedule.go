package render

import "qq/anapa2006/internal/i18n"

func ScheduleStatusLabel(lang i18n.Lang, status string) string {
	switch status {
	case "pending":
		return i18n.T(lang, i18n.ScheduledStatusPending)
	case "sending":
		return i18n.T(lang, i18n.ScheduledStatusSending)
	case "sent":
		return i18n.T(lang, i18n.ScheduledStatusSent)
	case "cancelled":
		return i18n.T(lang, i18n.ScheduledStatusCancelled)
	default:
		return status
	}
}
