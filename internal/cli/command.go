package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/stoicknight/stoa/internal/store"
)

// command describes one stoa subcommand. Adding a command means adding
// one struct literal to the registry in cli.go.
type command struct {
	name    string
	args    string // argument spec shown in usage, empty if none
	summary string // one-line description shown in usage
	run     func(args []string) error
}

// fsBase returns flag set without shared globals (per-command isolation).
func fsBase(name string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(name, pflag.ExitOnError)
	fs.String("config", "", "path to stoa.toml (default ~/.config/stoa/stoa.toml)")
	return fs
}

// vaultOf pulls vault path from config, ensuring the directory exists.
func vaultOf(cfg Config) (string, error) {
	vault, err := filepath.Abs(expandHome(cfg.Vault))
	if err != nil {
		return "", fmt.Errorf("resolve vault: %w", err)
	}
	fi, err := os.Stat(vault)
	if err != nil || !fi.IsDir() {
		return "", fmt.Errorf("vault not initialized: %s (run 'stoa init')", vault)
	}
	return vault, nil
}

// openStore initializes DB (creating if needed) under vault dir and returns it plus vault path.
func openStore(ctx context.Context, cfg Config) (*store.Store, string, error) {
	vault, err := vaultOf(cfg)
	if err != nil {
		return nil, "", err
	}
	dbPath := filepath.Join(vault, ".stoa", "index.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, "", fmt.Errorf("create db dir: %w", err)
	}
	st, err := store.Open(ctx, dbPath)
	if err != nil {
		return nil, "", err
	}
	return st, vault, nil
}

func failf(format string, a ...any) error {
	return fmt.Errorf(format, a...)
}
