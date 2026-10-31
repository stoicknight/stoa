package cv

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// requireTypst skips render tests when the typst CLI is not installed.
func requireTypst(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst not installed (run: sudo pacman -S typst)")
	}
}

func TestRenderOneShot(t *testing.T) {
	t.Parallel()
	requireTypst(t)

	c, err := Parse(sample)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "cv.pdf")

	r, err := NewTheme("modern")
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	defer func() { _ = r.Close() }()

	if err := r.Compile(context.Background(), c, out); err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(raw) < 4 || string(raw[:4]) != "%PDF" {
		t.Errorf("output is not a PDF (%d bytes, starts %q)", len(raw), raw[:min(8, len(raw))])
	}
	schema, err := os.ReadFile(SchemaPath(out))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if !strings.Contains(string(schema), `name: "Jane Doe"`) {
		t.Errorf("schema artifact missing name field")
	}
}

func TestRenderThemesCompile(t *testing.T) {
	t.Parallel()
	requireTypst(t)

	c, err := Parse(sample)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	for _, name := range []string{"classic", "minimalist"} { // modern is covered by TestRenderOneShot
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.Join(dir, "cv.pdf")

			r, err := NewTheme(name)
			if err != nil {
				t.Fatalf("NewTheme(%q) error = %v", name, err)
			}
			defer func() { _ = r.Close() }()

			if err := r.Compile(context.Background(), c, out); err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			raw, err := os.ReadFile(out)
			if err != nil {
				t.Fatalf("read output: %v", err)
			}
			if string(raw[:4]) != "%PDF" {
				t.Errorf("%s theme output is not a PDF", name)
			}
		})
	}
}

func TestNewThemeUnknown(t *testing.T) {
	t.Parallel()
	_, err := NewTheme("nope")
	if err == nil {
		t.Fatal("NewTheme(nope) = nil error, want unknown-theme error")
	}
	for _, name := range themeNames() {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not list valid theme %q", err, name)
		}
	}
}

func TestRenderCustomTemplate(t *testing.T) {
	t.Parallel()
	requireTypst(t)

	c, err := Parse("---\ntitle: T\ntype: cv\n---\n## A\n- b\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "custom.typ")
	if err := os.WriteFile(tmpl, []byte("#let data = yaml(\"data.yaml\")\nHello #data.name"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	out := filepath.Join(dir, "out.pdf")

	r, err := NewRenderer(tmpl)
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	defer func() { _ = r.Close() }()

	if err := r.Compile(context.Background(), c, out); err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	outfile, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(outfile[:4]) != "%PDF" {
		t.Errorf("custom template output is not a PDF")
	}
}

func TestNewRendererMissingTypst(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("typst"); err == nil {
		t.Skip("typst installed; missing-binary path not testable here")
	}
	_, err := NewTheme("modern")
	if err == nil {
		t.Fatal("NewRenderer() = nil error, want install hint")
	}
	if !strings.Contains(err.Error(), "typst CLI not found") {
		t.Errorf("error = %q, want install hint", err)
	}
}
