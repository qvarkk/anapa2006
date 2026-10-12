-- name: ListDraftMedia :many
SELECT * FROM draft_media WHERE draft_id = ? ORDER BY position ASC;

-- name: CreateDraftMedia :exec
INSERT INTO draft_media (
  draft_id, kind, origin_media_id,
  file_id, url, position
) VALUES (
  ?, ?, ?, ?, ?, ?
);

-- name: CountMediaKindsByDraft :many
SELECT kind, COUNT(*) AS cnt FROM draft_media WHERE draft_id = ? GROUP BY kind;

-- name: CountDraftMedia :one
SELECT COUNT(*) FROM draft_media WHERE draft_id = ?;

-- name: GetDraftMediaByID :one
SELECT * FROM draft_media WHERE id = ?;

-- name: DeleteDraftMediaByDraft :exec
DELETE FROM draft_media WHERE draft_id = ?;

-- name: DeleteDraftMediaByID :exec
DELETE FROM draft_media WHERE id = ?;

-- name: UpdateDraftMedia :exec
UPDATE draft_media SET kind = ?, file_id = ?, url = NULL, origin_media_id = NULL WHERE id = ?;
