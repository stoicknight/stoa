package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/stoicknight/stoa/internal/cv"
)

// schemaFor writes the parsed schema YAML next to the PDF output so the
// data boundary stays inspectable (MVP testing/preview artifact).
func schemaFor(out string, c *cv.CV) error {
	if err := os.WriteFile(cv.SchemaPath(out), []byte(c.YAML()), 0o600); err != nil {
		return fmt.Errorf("write schema: %w", err)
	}
	return nil
}

// runGen renders a CV markdown file to PDF (Typst). It works on any
// markdown file — no vault needed — and keeps the parsed schema YAML
// next to the PDF as a preview/debug artifact.
func runGen(args []string) error {
	fs := fsBase("gen")
	outPath := fs.StringP("out", "o", "", "output PDF path (default: next to the input file)")
	pdf := fs.Bool("pdf", false, "output format; PDF is currently the only one (and the default)")
	theme := fs.String("theme", "classic", "built-in theme (classic | modern | minimalist) or a custom .typ template")
	watch := fs.Bool("watch", false, "re-render when the input file changes")
	_ = pdf // future formats branch on this; pdf is the only one today
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return failf("usage: stoa gen <file.md> [--pdf] [-o out] [--theme name|.typ] [--watch]")
	}
	input := rest[0]

	raw, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("read %s: %w", input, err)
	}
	c, err := cv.Parse(string(raw))
	if err != nil {
		return failf("gen %s: %v", input, err)
	}
	out := *outPath
	if out == "" {
		out = strings.TrimSuffix(input, filepath.Ext(input)) + ".pdf"
	}

	ctx := context.Background()
	if *watch {
		ctx, _ = signal.NotifyContext(ctx, os.Interrupt)
	}

	r, err := cv.NewTheme(*theme)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	if *watch {
		return watchGen(ctx, r, c, input, out)
	}
	if err := r.Compile(ctx, c, out); err != nil {
		return failf("gen %s: %v", input, err)
	}
	fmt.Printf("generated %s\n", out)
	return nil
}

// watchGen runs typst watch on the workspace and re-parses the input
// markdown whenever it changes, rewriting the schema file so Typst
// recompiles automatically. Polling (mtime, 1s) keeps it stdlib-only.
func watchGen(ctx context.Context, r *cv.Renderer, c *cv.CV, input, out string) error {
	if err := r.WriteData(c); err != nil {
		return failf("gen %s: %v", input, err)
	}
	if err := schemaFor(out, c); err != nil {
		return failf("gen %s: %v", input, err)
	}
	watchErr := make(chan error, 1)
	go func() { watchErr <- r.Watch(ctx, out) }()
	fmt.Printf("watching %s — Ctrl-C to stop (writes %s)\n", input, out)

	last := modTime(input)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case err := <-watchErr:
			if err != nil && ctx.Err() == nil {
				return failf("gen %s: %v", input, err)
			}
			return nil
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			fi, err := os.Stat(input)
			if err != nil {
				fmt.Fprintln(os.Stderr, "stoa:", err)
				continue
			}
			if fi.ModTime().Equal(last) {
				continue
			}
			last = fi.ModTime()
			raw, err := os.ReadFile(input)
			if err != nil {
				fmt.Fprintln(os.Stderr, "stoa:", err)
				continue
			}
			updated, err := cv.Parse(string(raw))
			if err != nil {
				fmt.Fprintf(os.Stderr, "stoa: gen: %v\n", err)
				continue
			}
			if err := r.WriteData(updated); err != nil {
				fmt.Fprintln(os.Stderr, "stoa:", err)
				continue
			}
			if err := schemaFor(out, updated); err != nil {
				fmt.Fprintln(os.Stderr, "stoa:", err)
				continue
			}
			fmt.Printf("reloaded %s\n", input)
		}
	}
}

// modTime returns the file's mtime, or zero time if it cannot be read.
func modTime(path string) time.Time {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}
