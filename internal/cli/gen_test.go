package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenEndToEnd renders a small CV through the real typst binary and
// asserts the PDF and the sibling schema YAML exist.
func TestGenEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst not installed (run: sudo pacman -S typst)")
	}

	dir := t.TempDir()
	in := filepath.Join(dir, "cv.md")
	content := "---\ntitle: Jane Doe\ntype: cv\nposition: Engineer\nlocation: Berlin, Germany\nemail: jane@example.com\n---\n" +
		"## Summary\nGo engineer.\n\n## Experience\n### Developer\n**Acme** — Berlin\n2020 – 2024\n- Built **services**.\n"
	if err := os.WriteFile(in, []byte(content), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	if err := runGen([]string{in}); err != nil {
		t.Fatalf("runGen() error = %v", err)
	}

	wantPDF := filepath.Join(dir, "cv.pdf")
	raw, err := os.ReadFile(wantPDF)
	if err != nil {
		t.Fatalf("read %s: %v", wantPDF, err)
	}
	if string(raw[:4]) != "%PDF" {
		t.Errorf("%s is not a PDF", wantPDF)
	}
	schema, err := os.ReadFile(filepath.Join(dir, "cv.yaml"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if !strings.Contains(string(schema), `name: "Jane Doe"`) {
		t.Errorf("schema = %q", schema)
	}
}

func TestGenUsage(t *testing.T) {
	t.Parallel()
	if err := runGen(nil); err == nil || !strings.Contains(err.Error(), "usage: stoa gen") {
		t.Errorf("runGen(nil) = %v, want usage error", err)
	}
	if err := runGen([]string{"a.md", "b.md"}); err == nil || !strings.Contains(err.Error(), "usage: stoa gen") {
		t.Errorf("runGen(two files) = %v, want usage error", err)
	}
}

// writeCV is the minimal valid CV input shared by the gen error tests.
func writeCV(t *testing.T, dir string) string {
	t.Helper()
	in := filepath.Join(dir, "cv.md")
	content := "---\ntitle: Jane Doe\ntype: cv\nposition: Engineer\n---\n## Summary\nGo engineer.\n"
	if err := os.WriteFile(in, []byte(content), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	return in
}

// TestGenThemeTemplateFile pins the --theme resolution through runGen:
// .typ paths select templates from disk (fail-fast before typst), and
// unknown non-.typ values stay theme errors.
func TestGenThemeTemplateFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	in := writeCV(t, dir)

	err := runGen([]string{"--theme", filepath.Join(dir, "missing.typ"), in})
	if err == nil || !strings.Contains(err.Error(), "missing.typ") {
		t.Errorf("runGen(--theme missing.typ) = %v, want template-not-found error", err)
	}

	err = runGen([]string{"--theme", "nope", in})
	if err == nil || !strings.Contains(err.Error(), "unknown theme") {
		t.Errorf("runGen(--theme nope) = %v, want unknown-theme error", err)
	}

	// contract failure surfaces at selection, never as an empty PDF
	bad := filepath.Join(dir, "nodata.typ")
	if err := os.WriteFile(bad, []byte("= Hello world"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = runGen([]string{"--theme", bad, in})
	if err == nil || !strings.Contains(err.Error(), "data.yaml") {
		t.Errorf("runGen(--theme nodata.typ) = %v, want contract error naming data.yaml", err)
	}
}

func TestGenUnknownTheme(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	in := filepath.Join(dir, "cv.md")
	content := "---\ntitle: Jane Doe\ntype: cv\nposition: Engineer\n---\n## Summary\nGo engineer.\n"
	if err := os.WriteFile(in, []byte(content), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	err := runGen([]string{"--theme", "nope", in})
	if err == nil || !strings.Contains(err.Error(), "unknown theme") {
		t.Errorf("runGen(--theme nope) = %v, want unknown-theme error", err)
	}
}

func TestGenMissingFile(t *testing.T) {
	t.Parallel()
	err := runGen([]string{filepath.Join(t.TempDir(), "nope.md")})
	if err == nil || !strings.Contains(err.Error(), "nope.md") {
		t.Errorf("runGen(missing) = %v, want read error", err)
	}
}

func TestGenBadCV(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	in := filepath.Join(dir, "cv.md")
	if err := os.WriteFile(in, []byte("no frontmatter\n"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	err := runGen([]string{in})
	if err == nil || !strings.Contains(err.Error(), "frontmatter") {
		t.Errorf("runGen(bad cv) = %v, want parse error", err)
	}
}
