-- name: GetAllowedUser :one
SELECT user_id, username, lang FROM allowed_users WHERE user_id = ?;

-- name: AddAllowedUser :exec
INSERT INTO allowed_users (user_id, username) VALUES (?, ?)
ON CONFLICT(user_id) DO NOTHING;

-- name: ListAllowedUsers :many
SELECT user_id, username, lang, added_at FROM allowed_users
ORDER BY added_at ASC, user_id ASC
LIMIT ? OFFSET ?;

-- name: CountAllowedUsers :one
SELECT COUNT(*) FROM allowed_users;

-- name: DeleteAllowedUser :exec
DELETE FROM allowed_users WHERE user_id = ?;