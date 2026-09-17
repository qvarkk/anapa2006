-- name: GetDraftByID :one
SELECT * FROM drafts WHERE id = ?;
