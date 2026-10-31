-- name: UpsertNote :exec
INSERT INTO notes (id, path, title, type, created, updated, hash)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    path    = excluded.path,
    title   = excluded.title,
    type    = excluded.type,
    created = excluded.created,
    updated = excluded.updated,
    hash    = excluded.hash;

-- name: GetNoteByID :one
SELECT * FROM notes WHERE id = ?;

-- name: ListNoteIdentifiers :many
SELECT id, path FROM notes;

-- name: ListNoteHashes :many
SELECT id, path, hash FROM notes;

-- name: ListAllNotes :many
SELECT * FROM notes ORDER BY created DESC LIMIT ?;

-- name: ListNotesByType :many
SELECT * FROM notes WHERE type = ? ORDER BY created DESC LIMIT ?;

-- name: ListNotesByTag :many
SELECT n.* FROM notes n
JOIN tags t ON t.note_id = n.id
WHERE t.tag = ?
ORDER BY n.created DESC LIMIT ?;

-- name: DeleteNoteByID :exec
DELETE FROM notes WHERE id = ?;

-- name: ReplaceTags :exec
DELETE FROM tags WHERE note_id = ?;

-- name: InsertTag :exec
INSERT INTO tags (note_id, tag) VALUES (?, ?);

-- name: ReplaceFTS :exec
DELETE FROM notes_fts WHERE note_id = ?;

-- name: InsertFTSRow :exec
INSERT INTO notes_fts (note_id, title, body) VALUES (?, ?, ?);

-- name: SearchNoteIDs :many
-- sqlc cannot parse fts5's hidden MATCH column yet, so MVP search is a
-- case-insensitive substring scan over the indexed columns. The fts table
-- stays populated so a raw prepared MATCH query can slot in later.
SELECT note_id FROM notes_fts
WHERE title LIKE @pattern OR body LIKE @pattern
ORDER BY note_id
LIMIT @max;

-- name: GetNotesByIDs :many
SELECT * FROM notes
WHERE id IN (sqlc.slice('ids'))
ORDER BY created DESC;
