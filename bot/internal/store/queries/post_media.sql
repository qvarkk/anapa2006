-- name: AddPostMedia :one
INSERT INTO post_media (post_id, kind, url, position) VALUES (?, ?, ?, ?) RETURNING *;

-- name: ListPostMedia :many
SELECT * FROM post_media WHERE post_id = ? ORDER BY position ASC;