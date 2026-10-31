package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stoicknight/stoa/internal/index"
)

// runRemove moves notes into vault/.trash (or deletes with --purge),
// guarded by a confirm prompt unless --yes is given.
func runRemove(args []string) error {
	fs := fsBase("rm")
	yes := fs.Bool("yes", false, "skip confirmation prompt")
	purge := fs.Bool("purge", false, "delete files instead of moving to .trash")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return failf("usage: stoa rm <ref>...")
	}
	cfg := loadConfig(fs.Lookup("config").Value.String())
	ctx := context.Background()
	st, vault, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	var targets []struct{ id, path string }
	for _, arg := range fs.Args() {
		ref, err := resolveQuery(ctx, st, arg)
		if err != nil {
			return err
		}
		note, err := st.Q.GetNoteByID(ctx, ref)
		if err != nil {
			return failf("note not found: %v", err)
		}
		targets = append(targets, struct{ id, path string }{note.ID, note.Path})
	}
	if !*yes && !promptContinue(fmt.Sprintf("remove %d note(s)%s?", len(targets), purgeSuffix(*purge))) {
		fmt.Println("aborted")
		return nil
	}
	for _, t := range targets {
		full := filepath.Join(vault, filepath.FromSlash(t.path))
		if *purge {
			if err := os.Remove(full); err != nil {
				return failf("delete %s: %v", t.path, err)
			}
		} else {
			trashDir := filepath.Join(vault, ".trash")
			if err := os.MkdirAll(trashDir, 0o700); err != nil {
				return failf("create trash: %v", err)
			}
			dst := filepath.Join(trashDir, filepath.Base(full))
			if err := os.Rename(full, dst); err != nil {
				return failf("move to trash %s: %v", t.path, err)
			}
		}
	}
	res, err := index.Reindex(ctx, st, vault)
	if err != nil {
		return err
	}
	fmt.Printf("removed %d note(s), pruned %d row(s)\n", len(targets), res.Pruned)
	return nil
}

func purgeSuffix(purge bool) string {
	if purge {
		return " permanently"
	}
	return ""
}
