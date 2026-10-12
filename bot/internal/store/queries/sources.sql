-- name: GetActiveSources :many
SELECT * FROM sources WHERE active = 1;

-- name: CreateSource :one
INSERT INTO sources (channel_handle, active) VALUES (?, ?) RETURNING *;

-- name: ListSourcesWithNewCount :many
SELECT s.id, s.channel_handle,
  CAST(COALESCE(SUM(CASE WHEN p.status = 'new' THEN 1 ELSE 0 END), 0) AS INTEGER) AS new_count
FROM sources s
LEFT JOIN posts p ON p.source_id = s.id
GROUP BY s.id, s.channel_handle
ORDER BY s.channel_handle
LIMIT ? OFFSET ?;

-- name: CountSources :one
SELECT COUNT(*) FROM sources;

-- name: ListSources :many
SELECT * FROM sources ORDER BY channel_handle ASC LIMIT ? OFFSET ?;

-- name: ListSourcesWithStats :many
SELECT s.id, s.channel_handle, s.active,
  CAST(COUNT(p.id) AS INTEGER) AS fetched,
  CAST(COALESCE(SUM(CASE WHEN p.status = 'fetch_error' THEN 1 ELSE 0 END), 0) AS INTEGER) AS failed,
  CAST(COALESCE(SUM(CASE WHEN p.status = 'sent' THEN 1 ELSE 0 END), 0) AS INTEGER) AS sent
FROM sources s
LEFT JOIN posts p ON p.source_id = s.id
GROUP BY s.id, s.channel_handle, s.active
ORDER BY s.channel_handle
LIMIT ? OFFSET ?;

-- name: GetSourceStats :one
SELECT
  CAST(COUNT(*) AS INTEGER) AS fetched,
  CAST(COALESCE(SUM(CASE WHEN status = 'fetch_error' THEN 1 ELSE 0 END), 0) AS INTEGER) AS failed,
  CAST(COALESCE(SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END), 0) AS INTEGER) AS sent
FROM posts WHERE source_id = ?;

-- name: GetSourceByID :one
SELECT * FROM sources WHERE id = ?;

-- name: GetSourceByHandle :one
SELECT * FROM sources WHERE channel_handle = ?;

-- name: ToggleSourceActive :exec
UPDATE sources SET active = 1 - active WHERE id = ?;

-- name: DeleteSource :exec
DELETE FROM sources WHERE id = ?;