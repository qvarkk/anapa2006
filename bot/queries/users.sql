-- name: IsAllowedUser :one
SELECT EXISTS(SELECT 1 FROM allowed_users WHERE user_id = ?) AS allowed;

-- name: AddAllowedUser :exec
INSERT INTO allowed_users (user_id, username) VALUES (?, ?)
ON CONFLICT(user_id) DO NOTHING;