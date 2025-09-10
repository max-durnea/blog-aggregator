-- name: CreateFeedFollow :one
WITH inserted AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING *
)
SELECT 
    inserted.id,
    inserted.created_at,
    inserted.updated_at,
    inserted.user_id,
    u.name AS user_name,
    inserted.feed_id,
    f.name AS feed_name
FROM inserted
JOIN users u ON inserted.user_id = u.id
JOIN feeds f ON inserted.feed_id = f.id;

-- name: GetFeedFollowsForUser :many
SELECT 
    ff.id,
    ff.created_at,
    ff.updated_at,
    ff.user_id,
    u.name AS user_name,
    ff.feed_id,
    f.name AS feed_name
FROM feed_follows ff
JOIN users u ON ff.user_id = u.id
JOIN feeds f ON ff.feed_id = f.id
WHERE ff.user_id = $1;

-- name: DeleteFeedFollow

DELETE FROM feed_follows ff
USING users u, feeds f
WHERE ff.user_id = u.id
  AND ff.feed_id = f.id
  AND u.username = $1
  AND f.url = $2;

