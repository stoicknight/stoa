package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/stoicknight/stoa/internal/index"
)

// runEdit opens the resolved note's file in $EDITOR, then reindexes.
func runEdit(args []string) error {
	fs := fsBase("edit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return failf("usage: stoa edit <id|title|filename>")
	}
	cfg := loadConfig(fs.Lookup("config").Value.String())
	ctx := context.Background()
	st, vault, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	ref, err := resolveQuery(ctx, st, fs.Arg(0))
	if err != nil {
		return err
	}
	note, err := st.Q.GetNoteByID(ctx, ref)
	if err != nil {
		return failf("note not found: %v", err)
	}
	target := filepath.Join(vault, filepath.FromSlash(note.Path))
	if err := runEditor(cfg.Editor, target); err != nil {
		return err
	}
	res, err := index.Reindex(ctx, st, vault)
	if err != nil {
		return err
	}
	fmt.Printf("updated %s (reindexed %d files)\n", note.Title, res.Files)
	return nil
}
