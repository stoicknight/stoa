package cli

import (
	"context"
	"fmt"

	"github.com/stoicknight/stoa/internal/index"
)

// runReindex rescans the vault and refreshes the disposable index.
func runReindex(args []string) error {
	fs := fsBase("reindex")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg := loadConfig(fs.Lookup("config").Value.String())
	st, vault, err := openStore(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	res, err := index.Reindex(context.Background(), st, vault)
	if err != nil {
		return err
	}
	fmt.Printf("indexed %d files (%d updated, %d pruned, %d links)\n",
		res.Files, res.Updated, res.Pruned, res.Links)
	return nil
}
