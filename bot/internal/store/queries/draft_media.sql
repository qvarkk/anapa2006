-- name: ListDraftMedia :many
SELECT * FROM draft_media WHERE draft_id = ?;