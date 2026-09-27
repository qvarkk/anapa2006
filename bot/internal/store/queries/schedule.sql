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

-- name: CountScheduled :one
SELECT COUNT(*) FROM schedule;

-- name: ListScheduledLatest :many
SELECT s.*, d.final_text FROM schedule s
JOIN drafts d ON d.id = s.draft_id
WHERE s.status <> 'sent'
ORDER BY s.scheduled_at DESC
LIMIT ? OFFSET ?;

-- name: GetScheduleWithDraftData :one
SELECT s.*, d.final_text, p.external_id FROM schedule s
JOIN drafts d ON d.id = s.draft_id
JOIN posts p on p.id = d.post_id
WHERE s.id = ?;
