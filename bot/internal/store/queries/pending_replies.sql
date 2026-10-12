-- name: CreatePendingReply :exec
INSERT INTO pending_replies (chat_id, prompt_message_id, action, post_id, origin)
VALUES (?, ?, ?, ?, ?);

-- name: GetPendingReply :one
SELECT * FROM pending_replies WHERE chat_id = ? AND prompt_message_id = ?;

-- name: GetLatestPendingReplyByAction :one
SELECT * FROM pending_replies
WHERE chat_id = ? AND action = ?
ORDER BY created_at DESC
LIMIT 1;

-- name: DeletePendingReply :exec
DELETE FROM pending_replies WHERE chat_id = ? AND prompt_message_id = ?;
