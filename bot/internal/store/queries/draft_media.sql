-- name: ListDraftMedia :many
SELECT * FROM draft_media WHERE draft_id = ?;

-- name: CreateDraftMedia :exec
INSERT INTO draft_media (
  draft_id, kind, origin_media_id,
  file_id, url, position
) VALUES (
  ?, ?, ?, ?, ?, ?
);
