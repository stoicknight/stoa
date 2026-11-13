package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stoicknight/stoa/internal/store/dbx"
)

// TestOpenMigrations applies migrations through store.Open and asserts
// the fresh schema is usable with foreign_keys enforcement on.
func TestOpenMigrations(t *testing.T) {
	t.Parallel()
	st, err := Open(t.Context(), filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = st.Close() }()
	var on int
	if err := st.DB.QueryRow("PRAGMA foreign_keys").Scan(&on); err != nil {
		t.Fatal(err)
	}
	if on != 1 {
		t.Fatalf("PRAGMA foreign_keys = %d, want 1", on)
	}
	if err := st.Q.InsertTag(t.Context(), dbx.InsertTagParams{NoteID: "ghost", Tag: "pk"}); err == nil {
		t.Fatal("tag with unknown note_id: expected foreign-key violation")
	}
}

// TestOpenRejectsBadPath pins the error path for an unopenable DSN.
func TestOpenRejectsBadPath(t *testing.T) {
	t.Parallel()
	bad := filepath.Join(t.TempDir(), "nodir", "sub", "index.db")
	if err := os.MkdirAll(filepath.Dir(bad), 0o700); err != nil {
		t.Fatal(err)
	}
	// read-only leaf: sqlite cannot create the file on open
	if err := os.Chmod(filepath.Dir(bad), 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(bad), 0o700) })
	if _, err := Open(t.Context(), bad); err == nil {
		t.Fatal("expected error opening db under read-only directory")
	}
}
