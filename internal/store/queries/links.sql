-- name: GetLinksForNote :many
SELECT target_id, target, kind FROM links
WHERE source_id = ?
ORDER BY id;

-- name: GetBacklinks :many
SELECT n.* FROM notes n
JOIN links l ON l.source_id = n.id
WHERE l.target_id = ?
GROUP BY n.id
ORDER BY n.created DESC;

-- name: ClearAllLinks :exec
DELETE FROM links;

-- name: InsertLink :exec
INSERT INTO links (source_id, target_id, target, kind)
VALUES (?, ?, ?, ?);
