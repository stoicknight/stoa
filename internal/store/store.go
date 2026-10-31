// Package store opens the SQLite index and exposes generated queries.
// The database is a derived, disposable index over the vault files.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/pressly/goose/v3"

	"github.com/stoicknight/stoa/internal/store/dbx"
	"github.com/stoicknight/stoa/internal/store/migrations"
)

// Store wraps the sql handle and generated queries.
type Store struct {
	DB *sql.DB
	Q  *dbx.Queries
}

// Open opens (creating if needed) the SQLite file and applies migrations.
func Open(ctx context.Context, path string) (*Store, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve db path: %w", err)
	}
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", abs)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{DB: db, Q: dbx.New(db)}, nil
}

// Close releases the sqlite handle.
func (s *Store) Close() error { return s.DB.Close() }
