package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// runShow prints a note's file content; --path prints the file path instead.
func runShow(args []string) error {
	fs := fsBase("show")
	pathOnly := fs.Bool("path", false, "print the file path instead of content")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return failf("usage: stoa show <id|title|filename> [--path]")
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
	full := filepath.Join(vault, filepath.FromSlash(note.Path))
	if *pathOnly {
		fmt.Println(full)
		return nil
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return failf("read %s: %v", full, err)
	}
	_, _ = os.Stdout.Write(raw)
	return nil
}
