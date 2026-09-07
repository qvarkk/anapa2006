-- name: GetActiveSources :many
SELECT * FROM sources WHERE active = 1;

-- name: CreateSource :one
INSERT INTO sources (channel_handle, active) VALUES (?, ?) RETURNING *;