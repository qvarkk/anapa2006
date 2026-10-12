package domain

// Actions of a pending reply, matching the CHECK constraint on pending_replies.action.
const (
	PendingActionAwaitingText             = "awaiting_text"
	PendingActionAwaitingSchedule         = "awaiting_schedule"
	PendingActionAwaitingChannelHandle    = "awaiting_channel_handle"
	PendingActionAwaitingTargetHandle     = "awaiting_target_handle"
	PendingActionAwaitingUserRef          = "awaiting_user_ref"
	PendingActionAwaitingMediaAddPosition = "awaiting_media_add_position"
	PendingActionAwaitingMediaAdd         = "awaiting_media_add"
	PendingActionAwaitingMediaSubstitute  = "awaiting_media_substitute"
)
