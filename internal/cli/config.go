package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config carries the flat settings 'stoa' needs. Kept deliberately small;
// new keys go in one place (config.Defaults) and flow everywhere.
type Config struct {
	Vault  string `koanf:"vault"`
	Editor string `koanf:"editor"`
	Debug  bool   `koanf:"debug"`
}

// expandHome resolves a leading "~" against $HOME.
func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

// Default returns the built-in baseline configuration.
func Default() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	cfg := Config{Vault: filepath.Join(home, ".stoa", "vault"), Editor: os.Getenv("EDITOR")}
	if cfg.Editor == "" {
		cfg.Editor = "vi"
	}
	return cfg
}

// DefaultConfigPath is where stoa looks for stoa.toml unless overridden.
func defaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "stoa", "stoa.toml")
}

// loadConfig merges defaults < config file < STOA_VAULT < CLI flag override.
func loadConfig(path string) Config {
	k := koanf.New(".")
	_ = k.Load(confmap.Provider(map[string]interface{}{
		"vault":  Default().Vault,
		"editor": Default().Editor,
		"debug":  false,
	}, "."), nil)
	target := path
	if target == "" {
		target = defaultConfigPath()
	}
	if target != "" {
		if _, err := os.Stat(target); err == nil {
			if err := k.Load(file.Provider(target), toml.Parser()); err != nil {
				fmt.Fprintf(os.Stderr, "warning: reading config %s: %v\n", target, err)
			}
		}
	}
	if v := os.Getenv("STOA_VAULT"); v != "" {
		_ = k.Set("vault", v)
	}
	var cfg Config
	_ = k.Unmarshal("", &cfg)
	cfg.Vault = expandHome(cfg.Vault)
	return cfg
}

// runConfig prints or updates individual config keys.
func runConfig(args []string) error {
	fs := fsBase("config")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	path := fs.Lookup("config").Value.String()
	if path == "" {
		path = defaultConfigPath()
	}
	switch len(rest) {
	case 0:
		cfg := loadConfig(path)
		fmt.Printf("vault  = %s\neditor = %s\ndebug  = %v\n", cfg.Vault, cfg.Editor, cfg.Debug)
		return nil
	case 1:
		cfg := loadConfig(path)
		switch rest[0] {
		case "vault":
			fmt.Println(cfg.Vault)
		case "editor":
			fmt.Println(cfg.Editor)
		case "debug":
			fmt.Println(cfg.Debug)
		default:
			return failf("unknown key %q (vault, editor, debug)", rest[0])
		}
		return nil
	case 2:
		key, value := strings.ToLower(rest[0]), rest[1]
		switch key {
		case "vault", "editor", "debug":
		default:
			return failf("unknown key %q (vault, editor, debug)", key)
		}
		return setConfigKey(path, key, value)
	default:
		return failf("usage: stoa config [key [value]]")
	}
}

// setConfigKey writes one key into the config file, preserving the rest.
func setConfigKey(path, key, value string) error {
	if key == "vault" {
		value = expandHome(value)
	}
	k := koanf.New(".")
	if _, err := os.Stat(path); err == nil {
		if err := k.Load(file.Provider(path), toml.Parser()); err != nil {
			return failf("read config %s: %v", path, err)
		}
	}
	if err := k.Set(key, value); err != nil {
		return failf("set %s: %v", key, err)
	}
	out, err := k.Marshal(toml.Parser())
	if err != nil {
		return failf("serialize config: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return failf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return failf("write config: %v", err)
	}
	fmt.Printf("%s = %s (written to %s)\n", key, value, path)
	return nil
}
