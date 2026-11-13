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

	r, err := NewTheme("classic") // the default theme
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
	for _, name := range []string{"modern", "minimalist"} { // classic is covered by TestRenderOneShot
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
	if !strings.Contains(err.Error(), ".typ") {
		t.Errorf("error %q does not mention the .typ escape hatch", err)
	}
}

// TestNewThemeContract pins selection and the template contract: exact
// built-in names win, file paths must end in .typ (so a typo'd theme
// name cannot load a same-named file), and templates that would render
// silently empty are rejected at selection time.
func TestNewThemeContract(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// a same-named file must not shadow a built-in theme
	shadow := filepath.Join(dir, "classic")
	if err := os.WriteFile(shadow, []byte("junk"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTheme("classic"); err != nil {
		t.Fatalf("built-in name shadowed by file: %v", err)
	}

	// .typ paths resolve from disk; non-.typ unknowns stay theme errors
	if _, err := NewTheme(filepath.Join(dir, "missing.typ")); err == nil || !strings.Contains(err.Error(), "missing.typ") {
		t.Fatalf("NewTheme(missing.typ) = %v, want read error naming the file", err)
	}
	if _, err := NewTheme("somefile"); err == nil || !strings.Contains(err.Error(), "unknown theme") {
		t.Fatalf("NewTheme(somefile) = %v, want unknown-theme error", err)
	}

	// contract: empty file
	empty := filepath.Join(dir, "empty.typ")
	if err := os.WriteFile(empty, []byte("   \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTheme(empty); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("NewTheme(empty) = %v, want empty-template error", err)
	}

	// contract: template that never reads data.yaml would render an
	// empty PDF — reject at selection, not at compile
	noData := filepath.Join(dir, "nodata.typ")
	if err := os.WriteFile(noData, []byte("= Hello world"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewTheme(noData)
	if err == nil || !strings.Contains(err.Error(), "data.yaml") {
		t.Fatalf("NewTheme(noData) = %v, want contract error naming data.yaml", err)
	}

	// valid external template passes validation and lands in the workspace
	requireTypst(t)
	good := filepath.Join(dir, "good.typ")
	if err := os.WriteFile(good, []byte("#let data = yaml(\"data.yaml\")\nHello #data.name"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := NewTheme(good)
	if err != nil {
		t.Fatalf("NewTheme(good) error = %v", err)
	}
	defer func() { _ = r.Close() }()
	written, err := os.ReadFile(r.tmplFile)
	if err != nil {
		t.Fatalf("read workspace template: %v", err)
	}
	if !strings.Contains(string(written), "data.yaml") {
		t.Errorf("workspace template = %q, want the user template", written)
	}
	if r.src != good {
		t.Errorf("template origin = %q, want the user path %q", r.src, good)
	}
}

// TestRelabel pins that typst diagnostics cite the template origin
// (user path or built-in label), never the temp workspace path.
func TestRelabel(t *testing.T) {
	t.Parallel()
	r := &Renderer{tmplFile: "/tmp/stoa-cv-1/cv.typ", src: `built-in theme "classic"`}
	got := relabel("error: unknown variable\n  ┌─ /tmp/stoa-cv-1/cv.typ:12:3", r)
	if strings.Contains(got, r.tmplFile) {
		t.Errorf("relabel() kept the temp path: %q", got)
	}
	if !strings.Contains(got, r.src) {
		t.Errorf("relabel() lost the origin label: %q", got)
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

	r, err := NewTheme(tmpl)
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
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
