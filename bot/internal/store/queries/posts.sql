-- name: UpsertPost :one
INSERT INTO posts (source_id, external_id, raw_text, published_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(source_id, external_id) DO NOTHING
RETURNING *;

-- name: GetPost :one
SELECT * FROM posts WHERE id = ?;

-- name: ListPostsBySource :many
SELECT p.*, s.channel_handle FROM posts p
JOIN sources s ON s.id = p.source_id
WHERE p.source_id = ?
  AND p.status <> 'fetch_error'
ORDER BY p.published_at DESC
LIMIT ? OFFSET ?;

-- name: CountPostsBySource :one
SELECT COUNT(*) FROM posts WHERE source_id = ? AND status <> 'fetch_error';

-- name: ListPostsLatest :many
SELECT p.*, s.channel_handle FROM posts p
JOIN sources s ON s.id = p.source_id
WHERE p.status <> 'fetch_error'
ORDER BY p.published_at DESC
LIMIT ? OFFSET ?;

-- name: CountPosts :one
SELECT COUNT(*) FROM posts WHERE status <> 'fetch_error';

-- name: GetPostWithSource :one
SELECT p.*, s.channel_handle FROM posts p
JOIN sources s ON s.id = p.source_id
WHERE p.id = ?;

-- name: ClaimScheduledPost :one
UPDATE posts SET status = 'sending'
WHERE id = ? AND status = 'scheduled'
RETURNING *;

-- name: GetRetrieablePosts :many
SELECT * FROM posts WHERE status = 'fetch_error' AND retry_count < ?;

-- name: IncrementRetryCount :exec
UPDATE posts SET retry_count = retry_count + 1 WHERE id = ?;

-- name: HidePost :exec
UPDATE posts SET status = 'skipped' WHERE id = ?;

-- name: UnhidePost :exec
UPDATE posts SET status = 'new' WHERE id = ?;

-- name: MarkPostScheduled :exec
UPDATE posts SET status = 'scheduled' WHERE id = ?;

-- name: MarkPostSending :exec
UPDATE posts SET status = 'sending' WHERE id = ?;

-- name: MarkPostSent :exec
UPDATE posts SET status = 'sent' WHERE id = ?;

-- name: MarkPostFailed :exec
UPDATE posts SET status = 'fetch_error' WHERE id = ?;

-- name: MarkPostArchived :exec
UPDATE posts SET status = 'archived' WHERE id = ?;
