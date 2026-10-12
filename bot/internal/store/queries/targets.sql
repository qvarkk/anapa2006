-- name: CreateTargetChannel :one
INSERT INTO target_channels (chat_id, handle, title) VALUES (?, ?, ?) RETURNING *;

-- name: GetTargetChannelByID :one
SELECT * FROM target_channels WHERE id = ?;

-- name: GetTargetChannelByChatID :one
SELECT * FROM target_channels WHERE chat_id = ?;

-- name: GetTargetChannelByHandle :one
SELECT * FROM target_channels WHERE handle = ?;

-- name: ListTargetChannels :many
SELECT * FROM target_channels ORDER BY title ASC, handle ASC LIMIT ? OFFSET ?;

-- name: CountTargetChannels :one
SELECT COUNT(*) FROM target_channels;

-- name: DeleteTargetChannel :exec
DELETE FROM target_channels WHERE id = ?;

-- name: ListScheduledDraftsByTarget :many
SELECT pd.post_id FROM post_drafts pd
JOIN posts p ON p.id = pd.post_id
WHERE pd.target_channel_id = ? AND p.status = 'scheduled';
