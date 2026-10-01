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
INSERT INTO post_drafts (post_id, final_text, user_id) VALUES (?, ?, ?) RETURNING *;

-- name: UpdatePostDraftText :exec
UPDATE post_drafts SET final_text = ? WHERE post_id = ?;

-- name: CountDrafts :one
SELECT COUNT(*) FROM post_drafts;

-- name: ListDraftsLatest :many
SELECT pd.*, p.status, s.scheduled_at FROM post_drafts pd
JOIN posts p ON p.id = pd.post_id
LEFT JOIN schedule s ON s.post_id = pd.post_id
WHERE p.status <> 'sent'
ORDER BY s.scheduled_at DESC
LIMIT ? OFFSET ?;

-- name: DeleteDraft :exec
DELETE FROM post_drafts WHERE post_id = ?;
