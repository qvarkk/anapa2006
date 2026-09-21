-- name: ListDuePending :many
SELECT * FROM schedule WHERE scheduled_at <= ? ORDER BY scheduled_at DESC;

-- name: ClaimDueSchedule :one
UPDATE schedule SET status = 'sending'
WHERE id = ? AND status = 'pending'
RETURNING *;

-- name: ResetScheduleStatus :exec
UPDATE schedule SET status = 'pending' WHERE id = ?;

-- name: MarkScheduleSent :exec
UPDATE schedule SET status = 'sent', sent_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = 'sending';

-- name: CreateSchedule :exec
INSERT INTO schedule (
  draft_id, target_chat_id, scheduled_at
) VALUES (
  ?, ?, ?
);