package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stoicknight/stoa/internal/domain"
	"github.com/stoicknight/stoa/internal/frontmatter"
	"github.com/stoicknight/stoa/internal/store"
	"github.com/stoicknight/stoa/internal/store/dbx"
)

const (
	idOne = "00000000-0000-7777-8000-000000000001"
	idTwo = "00000000-0000-7777-8000-000000000002"
)

// newVault makes a temp vault with a migrated DB plus the given files.
func newVault(t *testing.T, files map[string]string) (*store.Store, string) {
	t.Helper()
	vault := t.TempDir()
	for _, d := range []string{domain.FolderNote, domain.FolderJournal, domain.FolderDoc, ".stoa"} {
		if err := os.MkdirAll(filepath.Join(vault, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for rel, content := range files {
		abs := filepath.Join(vault, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	st, err := store.Open(context.Background(), filepath.Join(vault, ".stoa", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st, vault
}

// noteContent renders frontmatter + body with a fixed id for determinism.
func noteContent(id, title, typ string, tags []string, body string) string {
	m := frontmatter.Meta{ID: id, Title: title, Type: typ, Tags: tags, Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z"}
	out, err := frontmatter.Render(m, body)
	if err != nil {
		panic(err)
	}
	return out
}

func idByTitle(t *testing.T, st *store.Store, title string) string {
	t.Helper()
	rows, err := st.Q.ListAllNotes(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if strings.EqualFold(r.Title, title) {
			return r.ID
		}
	}
	t.Fatalf("no note titled %s", title)
	return ""
}

var tagParams = dbx.ListNotesByTagParams{Tag: "pk", Limit: 100}

func TestReindexRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st, vault := newVault(t, map[string]string{
		"notes/zettel.md": noteContent(idOne, "Zettel", "note", []string{"pk"}, "links to [[Doc One]] and [[Missing]]\n"),
		"docs/one.md":     noteContent(idTwo, "Doc One", "doc", nil, "back to [[Zettel]]\n"),
	})

	res, err := Reindex(ctx, st, vault)
	if err != nil {
		t.Fatalf("first reindex: %v", err)
	}
	if res.Files != 2 || res.Updated != 2 || res.Links != 3 {
		t.Fatalf("first reindex result = %+v, want {2 2 0 3}", res)
	}
	oneID := idByTitle(t, st, "Doc One")
	back, err := st.Q.GetBacklinks(ctx, &oneID)
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != 1 || back[0].Title != "Zettel" {
		t.Fatalf("backlinks = %+v, want Zettel", back)
	}
	// tags were indexed
	rows, err := st.Q.ListNotesByTag(ctx, tagParams)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Title != "Zettel" {
		t.Fatalf("list by tag = %+v", rows)
	}

	// second reindex: unchanged hashes must skip writes.
	res2, err := Reindex(ctx, st, vault)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Updated != 0 || res2.Pruned != 0 || res2.Links != 3 {
		t.Fatalf("second reindex result = %+v", res2)
	}

	// third reindex after removing a file: orphan pruning.
	if err := os.Remove(filepath.Join(vault, "notes", "zettel.md")); err != nil {
		t.Fatal(err)
	}
	res3, err := Reindex(ctx, st, vault)
	if err != nil {
		t.Fatal(err)
	}
	if res3.Pruned != 1 {
		t.Fatalf("pruned = %d, want 1", res3.Pruned)
	}
	linksLeft, err := Reindex(ctx, st, vault)
	if err != nil {
		t.Fatal(err)
	}
	if linksLeft.Links != 1 {
		t.Fatalf("links after prune = %d, want 1 (unresolved kept)", linksLeft.Links)
	}
	if _, err := st.Q.GetNoteByID(ctx, idOne); err == nil {
		t.Fatal("pruned note still present")
	}
}

func TestReindexAutoID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := "notes/raw.md"
	st, vault := newVault(t, map[string]string{
		path: "---\ntitle: Raw\ntype: note\ntags: \ncreated: c\nupdated: u\n---\nbody\n",
	})
	res, err := Reindex(ctx, st, vault)
	if err != nil {
		t.Fatalf("reindex: %v", err)
	}
	if res.Files != 1 || res.Updated != 1 {
		t.Fatalf("result = %+v", res)
	}
	raw, err := os.ReadFile(filepath.Join(vault, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	meta, _, ok := frontmatter.Parse(string(raw))
	if !ok || len(meta.ID) != 36 {
		t.Fatalf("auto id not injected: %q", meta.ID)
	}
}

func TestReindexRejectsBadType(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st, vault := newVault(t, map[string]string{
		"notes/bad.md": "---\nid: x1\ntitle: Bad\ntype: tweet\n---\n",
	})
	if _, err := Reindex(ctx, st, vault); err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestReindexHiddenDirsSkipped(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st, vault := newVault(t, map[string]string{
		"notes/keep.md":  noteContent(idOne, "Keep", "note", nil, "x"),
		".trash/gone.md": noteContent(idTwo, "Gone", "note", nil, "x"),
		".stoa/stuff.md": noteContent("00000000-0000-7777-8000-000000000009", "Stuff", "note", nil, "x"),
	})
	res, err := Reindex(ctx, st, vault)
	if err != nil {
		t.Fatal(err)
	}
	if res.Files != 1 {
		t.Fatalf("hidden dirs not skipped: %+v", res)
	}
}
