-- name: UpsertPost :one
INSERT INTO posts (source_id, external_id, raw_text, published_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(source_id, external_id) DO NOTHING
RETURNING *;

-- name: ListPostsBySource :many
SELECT p.*, s.channel_handle FROM posts p
JOIN sources s ON s.id = p.source_id
WHERE p.source_id = ?
ORDER BY p.published_at DESC
LIMIT ? OFFSET ?;

-- name: CountPostsBySource :one
SELECT COUNT(*) FROM posts WHERE source_id = ?;

-- name: ListPostsLatest :many
SELECT p.*, s.channel_handle FROM posts p
JOIN sources s ON s.id = p.source_id
ORDER BY p.published_at DESC
LIMIT ? OFFSET ?;

-- name: CountPosts :one
SELECT COUNT(*) FROM posts;

-- name: GetPostWithSource :one
SELECT p.*, s.channel_handle FROM posts p
JOIN sources s ON s.id = p.source_id
WHERE p.id = ?;