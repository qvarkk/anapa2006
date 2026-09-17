-- name: AddPostMedia :one
INSERT INTO post_media (post_id, kind, url, position) VALUES (?, ?, ?, ?) RETURNING *;

-- name: ListPostMedia :many
SELECT * FROM post_media WHERE post_id = ? ORDER BY position ASC;

-- name: CountMediaKindsByPost :many
SELECT kind, COUNT(*) AS cnt FROM post_media WHERE post_id = ? GROUP BY kind;

-- name: SetPostMediaFileID :exec
UPDATE post_media SET file_id = ? WHERE id = ?;