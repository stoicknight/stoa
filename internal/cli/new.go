package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/stoicknight/stoa/internal/domain"
	"github.com/stoicknight/stoa/internal/frontmatter"
	"github.com/stoicknight/stoa/internal/index"
	"github.com/stoicknight/stoa/internal/markdown"
)

// runNew creates a typed note file with frontmatter and opens $EDITOR.
func runNew(args []string) error {
	fs := fsBase("new")
	tags := fs.StringSlice("tags", nil, "comma-separated tags")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) < 2 {
		return failf("usage: stoa new <note|journal|doc> <title>")
	}
	typ, err := domain.ParseNoteType(rest[0])
	if err != nil {
		return err
	}
	title := strings.Join(rest[1:], " ")
	if strings.TrimSpace(title) == "" {
		return failf("title is required")
	}

	cfg := loadConfig(fs.Lookup("config").Value.String())
	vault, err := vaultOf(cfg)
	if err != nil {
		return err
	}
	folder, err := typ.Folder()
	if err != nil {
		return err
	}
	id := uuid.NewString()
	ctx := context.Background()
	body := "# " + title + "\n\n"
	content, err := frontmatter.Render(frontmatter.New(id, title, typ.String(), *tags), body)
	if err != nil {
		return err
	}
	target := filepath.Join(vault, folder, markdown.Slug(title)+"-"+id[:8]+".md")
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		return failf("create %s: %v", target, err)
	}
	if err := runEditor(cfg.Editor, target); err != nil {
		return err
	}
	if err := reindexQuiet(ctx, cfg); err != nil {
		return err
	}
	fmt.Printf("created %s\n", pathRelTo(vault, target))
	return nil
}

// reindexQuiet refreshes the index, surfacing errors cleanly. For vaults
// over ~1k files this stays cheap thanks to the hash-skip write path.
func reindexQuiet(ctx context.Context, cfg Config) error {
	st, vault, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	_, err = index.Reindex(ctx, st, vault)
	return err
}

// pathRelTo renders target relative to base when possible.
func pathRelTo(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}
