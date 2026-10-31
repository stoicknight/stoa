package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stoicknight/stoa/internal/frontmatter"
	"github.com/stoicknight/stoa/internal/index"
	"github.com/stoicknight/stoa/internal/markdown"
)

// runLink appends a [[target]] wikilink to the source note and reindexes.
func runLink(args []string) error {
	fs := fsBase("link")
	kind := fs.String("kind", "ref", "link kind (ref)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return failf("usage: stoa link <source-ref> <target-ref> [--kind=ref]")
	}
	cfg := loadConfig(fs.Lookup("config").Value.String())
	ctx := context.Background()
	st, vault, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	src, err := resolveQuery(ctx, st, fs.Arg(0))
	if err != nil {
		return err
	}
	dst, err := resolveQuery(ctx, st, fs.Arg(1))
	if err != nil {
		return err
	}
	if src == dst {
		return failf("cannot link a note to itself")
	}
	target, err := st.Q.GetNoteByID(ctx, dst)
	if err != nil {
		return failf("target note not found: %v", err)
	}
	source, err := st.Q.GetNoteByID(ctx, src)
	if err != nil {
		return failf("source note not found: %v", err)
	}
	full := filepath.Join(vault, filepath.FromSlash(source.Path))
	raw, err := os.ReadFile(full)
	if err != nil {
		return failf("read %s: %v", source.Path, err)
	}
	meta, body, hasFM := frontmatter.Parse(string(raw))
	if !hasFM {
		return failf("%s: missing frontmatter", source.Path)
	}
	updatedBody, changed := markdown.AppendLinkSection(body, target.Title, *kind)
	if !changed {
		fmt.Printf("link already present in %s\n", source.Title)
		return nil
	}
	content, err := frontmatter.Render(meta, updatedBody)
	if err != nil {
		return err
	}
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		return failf("write %s: %v", source.Path, err)
	}
	if _, err := index.Reindex(ctx, st, vault); err != nil {
		return err
	}
	fmt.Printf("linked %s -> %s\n", source.Title, target.Title)
	return nil
}
