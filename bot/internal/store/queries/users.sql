-- name: GetAllowedUser :one
SELECT user_id, username, lang FROM allowed_users WHERE user_id = ?;

-- name: AddAllowedUser :exec
INSERT INTO allowed_users (user_id, username) VALUES (?, ?)
ON CONFLICT(user_id) DO NOTHING;