package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stoicknight/stoa/internal/domain"
	"github.com/stoicknight/stoa/internal/store"
)

// runInit creates the vault layout, default config and empty index.
func runInit(args []string) error {
	fs := fsBase("init")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg := loadConfig(fs.Lookup("config").Value.String())
	if fs.Lookup("config").Value.String() == "" {
		writeConfigIfMissing()
	}
	vault, err := filepath.Abs(expandHome(cfg.Vault))
	if err != nil {
		return failf("resolve vault: %v", err)
	}
	for _, dir := range []string{
		domain.FolderNote,
		domain.FolderJournal,
		domain.FolderDoc,
		".stoa",
		".trash",
	} {
		if err := os.MkdirAll(filepath.Join(vault, dir), 0o700); err != nil {
			return failf("create %s: %v", dir, err)
		}
	}
	st, err := store.Open(context.Background(), filepath.Join(vault, ".stoa", "index.db"))
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	fmt.Printf("vault ready: %s (config: %s)\n", vault, defaultConfigPath())
	return nil
}

// writeConfigIfMissing drops a commented template at the default config path.
func writeConfigIfMissing() {
	path := defaultConfigPath()
	if path == "" {
		return
	}
	if _, err := os.Stat(path); err == nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	_ = os.WriteFile(path, []byte(defaultConfigTOML), 0o600)
}

const defaultConfigTOML = `# stoa configuration
vault = "~/.stoa/vault"   # where .md files live
editor = "nvim"           # falls back to $EDITOR, then vi
debug = false
`
