-- name: GetDraftByID :one
SELECT * FROM drafts WHERE id = ?;

-- name: CreateDraft :one
INSERT INTO drafts (post_id, final_text, user_id) VALUES (?, ?, ?) RETURNING *;

-- name: UpdateDraftText :exec
UPDATE drafts SET final_text = ? WHERE id = ?;

-- name: GetDraftsPostID :one
SELECT post_id FROM drafts WHERE id = ?;
