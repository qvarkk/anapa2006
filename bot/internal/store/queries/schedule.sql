-- name: ListDuePending :many
SELECT s.post_id, s.scheduled_at, s.sent_at, s.sent_message_id, s.last_error,
       t.id AS target_channel_id, t.chat_id, t.handle
FROM schedule s
JOIN posts p ON p.id = s.post_id
JOIN post_drafts pd ON pd.post_id = s.post_id
JOIN target_channels t ON t.id = pd.target_channel_id
WHERE s.scheduled_at <= ?
  AND s.sent_at IS NULL
  AND p.status = 'scheduled'
ORDER BY s.scheduled_at DESC;

-- name: MarkScheduleSent :exec
UPDATE schedule SET sent_at = CURRENT_TIMESTAMP, sent_message_id = ? WHERE post_id = ?;

-- name: MarkScheduleSendError :exec
UPDATE schedule SET last_error = ? WHERE post_id = ?;

-- name: GetPostScheduleByID :one
SELECT * FROM schedule WHERE post_id = ?;

-- name: CreateSchedule :exec
INSERT INTO schedule (
  post_id, scheduled_at
) VALUES (
  ?, ?
);

-- name: Deschedule :exec
DELETE FROM schedule WHERE post_id = ?;
