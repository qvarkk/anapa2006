package domain

// Actions of a pending reply, matching the CHECK constraint on pending_replies.action.
const (
	PendingActionAwaitingText     = "awaiting_text"
	PendingActionAwaitingSchedule = "awaiting_schedule"
)
