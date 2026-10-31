package cli

import (
	"context"
	"fmt"
)

// runBacklinks shows notes linking to the given one.
func runBacklinks(args []string) error {
	fs := fsBase("backlinks")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return failf("usage: stoa backlinks <ref>")
	}
	cfg := loadConfig(fs.Lookup("config").Value.String())
	ctx := context.Background()
	st, _, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	ref, err := resolveQuery(ctx, st, fs.Arg(0))
	if err != nil {
		return err
	}
	rows, err := st.Q.GetBacklinks(ctx, &ref)
	if err != nil {
		return failf("backlinks: %v", err)
	}
	if len(rows) == 0 {
		fmt.Println("no backlinks")
		return nil
	}
	for _, n := range rows {
		fmt.Printf("%s  %s\n", fuzzyID(n.ID, 8), n.Title)
	}
	return nil
}
