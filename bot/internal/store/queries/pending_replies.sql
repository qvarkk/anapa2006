-- name: CreatePendingReply :exec
INSERT INTO pending_replies (chat_id, prompt_message_id, action, draft_id, origin)
VALUES (?, ?, ?, ?, ?);

-- name: GetPendingReply :one
SELECT * FROM pending_replies WHERE chat_id = ? AND prompt_message_id = ?;

-- name: DeletePendingReply :exec
DELETE FROM pending_replies WHERE chat_id = ? AND prompt_message_id = ?;
