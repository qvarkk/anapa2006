-- name: GetPostDraftByID :one
SELECT * FROM post_drafts WHERE post_id = ?;

-- name: CheckDraftScheduled :one
SELECT EXISTS (
    SELECT 1
    FROM schedule
    WHERE post_id = ?
        AND scheduled_at IS NOT NULL
);

-- name: CheckPostDraftExists :one
SELECT EXISTS (
    SELECT 1 
    FROM post_drafts 
    WHERE post_id = ?
);

-- name: CreatePostDraft :one
INSERT INTO post_drafts (post_id, final_text) VALUES (?, ?) RETURNING *;

-- name: UpdatePostDraftText :exec
UPDATE post_drafts SET final_text = ? WHERE post_id = ?;

-- name: SetDraftTargetChannel :exec
UPDATE post_drafts SET target_channel_id = ? WHERE post_id = ?;

-- name: DeleteDraft :exec
DELETE FROM post_drafts WHERE post_id = ?;

-- name: CountActiveDrafts :one
SELECT COUNT(*) FROM post_drafts pd
JOIN posts p ON p.id = pd.post_id
WHERE p.status NOT IN ('sent', 'error_sending');

-- name: CountPostedDrafts :one
SELECT COUNT(*) FROM post_drafts pd
JOIN posts p ON p.id = pd.post_id
WHERE p.status = 'sent';

-- name: CountFailedDrafts :one
SELECT COUNT(*) FROM post_drafts pd
JOIN posts p ON p.id = pd.post_id
WHERE p.status = 'error_sending';

-- name: ListActiveDrafts :many
SELECT pd.post_id, pd.final_text, pd.created_at, pd.target_channel_id,
       p.status, s.scheduled_at, s.sent_at, s.sent_message_id, s.last_error,
       t.handle AS target_handle
FROM post_drafts pd
JOIN posts p ON p.id = pd.post_id
LEFT JOIN schedule s ON s.post_id = pd.post_id
LEFT JOIN target_channels t ON t.id = pd.target_channel_id
WHERE p.status NOT IN ('sent', 'error_sending')
ORDER BY COALESCE(s.scheduled_at, pd.created_at) DESC
LIMIT ? OFFSET ?;

-- name: ListPostedDrafts :many
SELECT pd.post_id, pd.final_text, pd.created_at, pd.target_channel_id,
       p.status, s.scheduled_at, s.sent_at, s.sent_message_id, s.last_error,
       t.handle AS target_handle
FROM post_drafts pd
JOIN posts p ON p.id = pd.post_id
LEFT JOIN schedule s ON s.post_id = pd.post_id
LEFT JOIN target_channels t ON t.id = pd.target_channel_id
WHERE p.status = 'sent'
ORDER BY s.sent_at DESC
LIMIT ? OFFSET ?;

-- name: ListFailedDrafts :many
SELECT pd.post_id, pd.final_text, pd.created_at, pd.target_channel_id,
       p.status, s.scheduled_at, s.sent_at, s.sent_message_id, s.last_error,
       t.handle AS target_handle
FROM post_drafts pd
JOIN posts p ON p.id = pd.post_id
LEFT JOIN schedule s ON s.post_id = pd.post_id
LEFT JOIN target_channels t ON t.id = pd.target_channel_id
WHERE p.status = 'error_sending'
ORDER BY COALESCE(s.scheduled_at, pd.created_at) DESC
LIMIT ? OFFSET ?;
