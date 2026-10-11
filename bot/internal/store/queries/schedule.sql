-- name: ListDuePending :many
SELECT s.* FROM schedule s
JOIN posts p ON p.id = s.post_id
WHERE s.scheduled_at <= ?
  AND s.sent_at IS NULL
  AND p.status = 'scheduled'
ORDER BY s.scheduled_at DESC;

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

-- name: Deschedule :exec
DELETE FROM schedule WHERE post_id = ?;
