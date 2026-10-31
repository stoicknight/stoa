// Package index rebuilds the SQLite index from vault files. The vault stays
// authoritative: every row can be regenerated with stoa reindex. Files whose
// content hash is unchanged are skipped for the tag/FTS write path.
package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/stoicknight/stoa/internal/domain"
	"github.com/stoicknight/stoa/internal/frontmatter"
	"github.com/stoicknight/stoa/internal/markdown"
	"github.com/stoicknight/stoa/internal/store"
	"github.com/stoicknight/stoa/internal/store/dbx"
)

// Result reports what the last reindex did.
type Result struct {
	Files   int // .md files found under the vault
	Updated int // DB rows written (new or changed content)
	Pruned  int // rows removed for deleted/moved files
	Links   int // link rows rebuilt
}

// doc is one parsed vault file kept for the duration of a reindex run.
type doc struct {
	rel   string // vault-relative path
	meta  frontmatter.Meta
	body  string
	hash  string
	links []markdown.Link
}

// Reindex walks the vault, upserts notes/tags/FTS, prunes orphans and
// rebuilds the links table from [[wikilinks]] found in file bodies.
func Reindex(ctx context.Context, st *store.Store, vault string) (Result, error) {
	vaultRoot, err := filepath.Abs(vault)
	if err != nil {
		return Result{}, fmt.Errorf("resolve vault: %w", err)
	}
	fi, err := os.Stat(vaultRoot)
	if err != nil {
		return Result{}, fmt.Errorf("vault not found: %s: %w", vaultRoot, err)
	}
	if !fi.IsDir() {
		return Result{}, fmt.Errorf("vault is not a directory: %s", vaultRoot)
	}

	docs, err := collect(vaultRoot)
	if err != nil {
		return Result{}, err
	}
	res := Result{Files: len(docs)}
	res.Updated, err = upsertDocs(ctx, st, docs)
	if err != nil {
		return res, err
	}
	res.Pruned, err = pruneOrphans(ctx, st, docs)
	if err != nil {
		return res, err
	}
	res.Links, err = rebuildLinks(ctx, st, docs)
	if err != nil {
		return res, err
	}
	return res, nil
}

// collect walks the vault (skipping hidden dirs such as .trash and .stoa),
// parses every .md file and assigns an id when one is missing.
func collect(vaultRoot string) ([]doc, error) {
	var docs []doc
	err := filepath.WalkDir(vaultRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(vaultRoot, path)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		meta, body, hasFM := frontmatter.Parse(string(raw))
		if !hasFM {
			return fmt.Errorf("%s: missing frontmatter; create files with 'stoa new' or add an id block", rel)
		}
		if meta.ID == "" {
			meta.ID = uuid.NewString()
			rendered, rerr := frontmatter.Render(meta, body)
			if rerr != nil {
				return fmt.Errorf("%s: render frontmatter: %w", rel, rerr)
			}
			if werr := os.WriteFile(path, []byte(rendered), 0o600); werr != nil {
				return fmt.Errorf("write %s: %w", rel, werr)
			}
			raw = []byte(rendered)
		}
		if strings.TrimSpace(meta.Title) == "" {
			return fmt.Errorf("%s: empty title in frontmatter", rel)
		}
		typ := meta.Type
		if typ == "" {
			typ = string(domain.Note)
		}
		if _, terr := domain.ParseNoteType(typ); terr != nil {
			return fmt.Errorf("%s: %w", rel, terr)
		}
		sum := sha256.Sum256(raw)
		docs = append(docs, doc{
			rel:   filepath.ToSlash(rel),
			meta:  meta,
			body:  body,
			hash:  hex.EncodeToString(sum[:]),
			links: markdown.ExtractLinks(body),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk vault: %w", err)
	}
	return docs, nil
}

func upsertDocs(ctx context.Context, st *store.Store, docs []doc) (int, error) {
	existing, err := st.Q.ListNoteHashes(ctx)
	if err != nil {
		return 0, fmt.Errorf("load note hashes: %w", err)
	}
	prev := make(map[string]string, len(existing)) // path -> hash
	for _, n := range existing {
		prev[n.Path] = n.Hash
	}
	written := 0
	for i := range docs {
		d := &docs[i]
		if prev[d.rel] == d.hash {
			continue
		}
		written++
		if err := st.Q.UpsertNote(ctx, dbx.UpsertNoteParams{
			ID:      d.meta.ID,
			Path:    d.rel,
			Title:   d.meta.Title,
			Type:    d.meta.Type,
			Created: d.meta.Created,
			Updated: d.meta.Updated,
			Hash:    d.hash,
		}); err != nil {
			return written, fmt.Errorf("upsert %s: %w", d.rel, err)
		}
		if err := storeTags(ctx, st, d); err != nil {
			return written, err
		}
	}
	return written, nil
}

func storeTags(ctx context.Context, st *store.Store, d *doc) error {
	if err := st.Q.ReplaceTags(ctx, d.meta.ID); err != nil {
		return fmt.Errorf("clear tags %s: %w", d.rel, err)
	}
	for _, tag := range d.meta.Tags {
		if err := st.Q.InsertTag(ctx, dbx.InsertTagParams{NoteID: d.meta.ID, Tag: tag}); err != nil {
			return fmt.Errorf("tag %q on %s: %w", tag, d.rel, err)
		}
	}
	if err := st.Q.ReplaceFTS(ctx, d.meta.ID); err != nil {
		return fmt.Errorf("clear fts %s: %w", d.rel, err)
	}
	body := strings.TrimSpace(strings.Join(d.meta.Tags, " ")) + "\n" + d.body
	if err := st.Q.InsertFTSRow(ctx, dbx.InsertFTSRowParams{
		NoteID: d.meta.ID,
		Title:  d.meta.Title,
		Body:   body,
	}); err != nil {
		return fmt.Errorf("index %s: %w", d.rel, err)
	}
	return nil
}

func pruneOrphans(ctx context.Context, st *store.Store, docs []doc) (int, error) {
	rows, err := st.Q.ListNoteIdentifiers(ctx)
	if err != nil {
		return 0, fmt.Errorf("load identifiers: %w", err)
	}
	seen := make(map[string]bool, len(docs))
	for _, d := range docs {
		seen[d.rel] = true
	}
	pruned := 0
	for _, r := range rows {
		if seen[r.Path] {
			continue
		}
		if err := st.Q.ReplaceFTS(ctx, r.ID); err != nil {
			return pruned, fmt.Errorf("fts cleanup: %w", err)
		}
		if err := st.Q.DeleteNoteByID(ctx, r.ID); err != nil {
			return pruned, fmt.Errorf("prune %s: %w", r.Path, err)
		}
		pruned++
	}
	return pruned, nil
}

// rebuildLinks wipes and rebuilds the links table from parsed bodies,
// resolving targets by exact id or case-insensitive title match.
func rebuildLinks(ctx context.Context, st *store.Store, docs []doc) (int, error) {
	all, err := st.Q.ListAllNotes(ctx, maxList)
	if err != nil {
		return 0, fmt.Errorf("load notes for link resolution: %w", err)
	}
	byID := map[string]bool{}
	byTitle := map[string]string{}
	for _, n := range all {
		byID[n.ID] = true
		byTitle[strings.ToLower(n.Title)] = n.ID
	}
	if err := st.Q.ClearAllLinks(ctx); err != nil {
		return 0, fmt.Errorf("clear links: %w", err)
	}
	count := 0
	for _, d := range docs {
		for _, l := range d.links {
			kind := l.Kind
			if kind == "" {
				kind = "ref"
			}
			args := dbx.InsertLinkParams{SourceID: d.meta.ID, Target: l.Target, Kind: kind}
			if id := byID[l.Target]; id {
				target := l.Target
				args.TargetID = &target
			} else if id, ok := byTitle[strings.ToLower(l.Target)]; ok {
				args.TargetID = &id
			}
			if err := st.Q.InsertLink(ctx, args); err != nil {
				return count, fmt.Errorf("insert link %s -> %s: %w", d.rel, l.Target, err)
			}
			count++
		}
	}
	return count, nil
}

// maxList bounds the rows loaded for in-memory resolution.
const maxList = 10_000
