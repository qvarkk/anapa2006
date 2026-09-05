-- name: UpsertPost :one
INSERT INTO posts (source_id, external_id, raw_text, published_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(source_id, external_id) DO NOTHING
RETURNING *;