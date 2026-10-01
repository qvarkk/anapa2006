-- name: ListDuePending :many
SELECT * FROM schedule WHERE scheduled_at <= ? ORDER BY scheduled_at DESC;

-- name: MarkScheduleSent :exec
UPDATE schedule SET sent_at = CURRENT_TIMESTAMP WHERE post_id = ?;

-- name: GetPostScheduleByID :one
SELECT * FROM schedule WHERE post_id = ?;

-- name: CreateSchedule :exec
INSERT INTO schedule (
  post_id, target_chat_id, scheduled_at
) VALUES (
  ?, ?, ?
);

-- name: CountScheduled :one
SELECT COUNT(*) FROM schedule;

-- name: ListScheduledLatest :many
SELECT s.*, pd.final_text, p.status FROM schedule s
JOIN post_drafts pd ON pd.post_id = s.draft_id
JOIN posts p on p.id = s.post_id
WHERE s.status <> 'sent'
ORDER BY s.scheduled_at DESC
LIMIT ? OFFSET ?;

-- name: GetScheduleWithDraftData :one
SELECT s.*, pd.final_text, p.external_id FROM schedule s
JOIN post_drafts pd ON pd.post_id = s.post_id
JOIN posts p on p.id = pd.post_id
WHERE s.post_id = ?;

-- name: Deschedule :exec
DELETE FROM schedule WHERE post_id = ?;
