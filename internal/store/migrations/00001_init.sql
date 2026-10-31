-- +goose Up
CREATE TABLE notes (
    id      TEXT PRIMARY KEY,
    path    TEXT NOT NULL UNIQUE,
    title   TEXT NOT NULL,
    type    TEXT NOT NULL CHECK (type IN ('note', 'journal', 'doc')),
    created TEXT NOT NULL,
    updated TEXT NOT NULL,
    hash    TEXT NOT NULL
);

CREATE TABLE tags (
    note_id TEXT NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    tag     TEXT NOT NULL,
    PRIMARY KEY (note_id, tag)
);

CREATE TABLE links (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id TEXT NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    target_id TEXT REFERENCES notes (id) ON DELETE CASCADE,
    target    TEXT NOT NULL,
    kind      TEXT NOT NULL DEFAULT 'ref'
);

CREATE INDEX idx_links_source ON links (source_id);
CREATE INDEX idx_links_target ON links (target_id);
CREATE INDEX idx_notes_type ON notes (type);
CREATE INDEX idx_tags_tag ON tags (tag);

CREATE VIRTUAL TABLE notes_fts USING fts5 (note_id UNINDEXED, title, body);

-- +goose Down
DROP TABLE notes_fts;
DROP TABLE links;
DROP TABLE tags;
DROP TABLE notes;
