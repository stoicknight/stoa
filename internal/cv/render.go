package cv

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed themes/minimalist.typ
var minimalistTemplate string

//go:embed themes/classic.typ
var classicTemplate string

//go:embed themes/modern.typ
var modernTemplate string

// themes maps theme names to embedded templates. The templates are
// self-contained (no external Typst packages): classic and modern
// reimplement rendercv 0.3.0's classic and engineeringresumes looks
// against stoa's data.yaml schema; minimalist is stoa's own design.
var themes = map[string]string{
	"classic":    classicTemplate, // default
	"modern":     modernTemplate,
	"minimalist": minimalistTemplate,
}

// themeNames returns the valid theme names, sorted for stable error
// messages and help output.
func themeNames() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Renderer owns the temporary Typst workspace: data.yaml plus the active
// template. It is reusable — watch mode rewrites data.yaml and lets
// `typst watch` recompile.
type Renderer struct {
	tmp      string
	typst    string
	tmplFile string
	dataFile string
	src      string // template origin: user path or built-in theme label
}

// NewTheme sets up the workspace with the given look. spec is either a
// built-in theme name (classic | modern | minimalist) or the path to a
// user-provided .typ file — the escape hatch for full control over the
// layout (see docs/reference/cv-template.md for the template contract).
func NewTheme(spec string) (*Renderer, error) {
	if tmpl, ok := themes[spec]; ok {
		return newWorkspace([]byte(tmpl), fmt.Sprintf("built-in theme %q", spec))
	}
	if !strings.HasSuffix(spec, ".typ") {
		return nil, fmt.Errorf("unknown theme %q (valid themes: %s — or pass a .typ file)", spec, strings.Join(themeNames(), ", "))
	}
	raw, err := os.ReadFile(spec)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", spec, err)
	}
	return newWorkspace(raw, spec)
}

// newWorkspace creates the temp directory, checks for a typst binary
// (with an install hint), validates the template against the stoa CV
// contract, and writes the template into it. src labels the template's
// origin (user path or built-in theme) in typst diagnostics.
func newWorkspace(tmpl []byte, src string) (*Renderer, error) {
	if err := validateTemplate(tmpl); err != nil {
		return nil, err
	}
	typst, err := exec.LookPath("typst")
	if err != nil {
		return nil, missingTypst()
	}
	tmp, err := os.MkdirTemp("", "stoa-cv-")
	if err != nil {
		return nil, fmt.Errorf("create temp workspace: %w", err)
	}
	r := &Renderer{
		tmp:      tmp,
		typst:    typst,
		tmplFile: filepath.Join(tmp, "cv.typ"),
		dataFile: filepath.Join(tmp, "data.yaml"),
		src:      src,
	}
	if err := os.WriteFile(r.tmplFile, tmpl, 0o600); err != nil {
		_ = os.RemoveAll(tmp) // best-effort cleanup
		return nil, fmt.Errorf("write template: %w", err)
	}
	return r, nil
}

// validateTemplate fails fast on templates that would compile clean but
// silently render an empty document: the stoa CV contract requires the
// template to load the schema itself via yaml("data.yaml").
func validateTemplate(tmpl []byte) error {
	if len(strings.TrimSpace(string(tmpl))) == 0 {
		return errors.New("template is empty")
	}
	if !strings.Contains(string(tmpl), "data.yaml") {
		return errors.New(`template never reads data.yaml — a stoa CV template must load the schema, e.g.: #let data = yaml("data.yaml") (see docs/reference/cv-template.md)`)
	}
	return nil
}

// Close removes the temporary workspace.
func (r *Renderer) Close() error {
	return os.RemoveAll(r.tmp)
}

// WriteData writes the schema into the workspace (and refreshes it in
// watch mode; typst watch recompiles when the file changes).
func (r *Renderer) WriteData(c *CV) error {
	if err := os.WriteFile(r.dataFile, []byte(c.YAML()), 0o600); err != nil {
		return fmt.Errorf("write data.yaml: %w", err)
	}
	return nil
}

// SchemaPath returns the schema YAML path paired with a PDF output path:
// same base name, .yaml extension.
func SchemaPath(out string) string {
	return strings.TrimSuffix(out, filepath.Ext(out)) + ".yaml"
}

// Compile renders the CV to out in one shot. It also writes the schema
// YAML next to the PDF as a preview/debug artifact.
func (r *Renderer) Compile(ctx context.Context, c *CV, out string) error {
	if err := r.WriteData(c); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, r.typst, "compile", "--root", r.tmp, r.tmplFile, out)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// typst diagnostics cite the workspace copy; relabel so the user
		// sees their own file (or the theme name) instead of a temp path
		return fmt.Errorf("typst compile: %w\n%s", err, relabel(trimFontWarnings(stderr.String()), r))
	}
	schema := SchemaPath(out)
	if err := os.WriteFile(schema, []byte(c.YAML()), 0o600); err != nil {
		return fmt.Errorf("write schema %s: %w", schema, err)
	}
	return nil
}

// Watch runs `typst watch` (which compiles once immediately) until ctx is
// done. Callers refresh the data with WriteData to trigger recompiles.
func (r *Renderer) Watch(ctx context.Context, out string) error {
	cmd := exec.CommandContext(ctx, r.typst, "watch", "--root", r.tmp, r.tmplFile, out)
	pr, pw := io.Pipe()
	cmd.Stderr = pw
	go func() {
		defer func() { _ = pr.Close() }()
		sc := bufio.NewScanner(dropFontWarnings(pr))
		for sc.Scan() {
			fmt.Fprintln(os.Stderr, relabel(sc.Text(), r))
		}
		if err := sc.Err(); err != nil && ctx.Err() == nil {
			fmt.Fprintln(os.Stderr, "stoa:", err)
		}
		_ = pw.Close()
	}()
	if err := cmd.Run(); err != nil && ctx.Err() == nil {
		return fmt.Errorf("typst watch: %w", err)
	}
	return nil
}

// trimFontWarnings removes "unknown font family" warnings (and their
// attached snippets) from typst output: the template's font list is a
// deliberate fallback chain, so those warnings are pure noise.
func trimFontWarnings(out string) string {
	var kept []string
	for _, block := range strings.Split(strings.TrimRight(out, "\n"), "\n\n") {
		if !strings.HasPrefix(block, "warning: unknown font family") {
			kept = append(kept, block)
		}
	}
	return strings.Join(kept, "\n\n")
}

// dropFontWarnings streams filtered typst output, dropping "unknown font
// family" warning blocks (a warning block ends at the first line that is
// not indented or empty).
func dropFontWarnings(r io.Reader) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		defer func() { _ = pw.Close() }()
		sc := bufio.NewScanner(r)
		skipping := false
		for sc.Scan() {
			line := sc.Text()
			if skipping {
				if line == "" || strings.HasPrefix(line, " ") {
					continue // still inside the warning snippet
				}
				skipping = false
			}
			if strings.HasPrefix(line, "warning: unknown font family") {
				skipping = true
				continue
			}
			_, _ = fmt.Fprintln(pw, line)
		}
	}()
	return pr
}

// relabel replaces the temp workspace path in typst output with the
// template's origin (user path or built-in theme label), so diagnostics
// point at a file the user can actually find.
func relabel(out string, r *Renderer) string {
	return strings.ReplaceAll(out, r.tmplFile, r.src)
}

// missingTypst returns an actionable install hint: a pacman command on
// Arch-based systems, a generic pointer elsewhere.
func missingTypst() error {
	if archLike(osRelease()) {
		return errors.New("typst CLI not found; on Arch-based systems install it with: sudo pacman -S typst")
	}
	return errors.New("typst CLI not found; install it (https://typst.app/docs/install/) and make sure it is on PATH")
}

// osRelease returns the contents of /etc/os-release, or "" if unreadable.
func osRelease() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	var b strings.Builder
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		b.WriteString(sc.Text())
		b.WriteByte('\n')
	}
	return b.String()
}

// archLike reports whether an /etc/os-release payload describes an
// Arch-based system (ID or ID_LIKE contains an Arch family identifier).
func archLike(release string) bool {
	for _, line := range strings.Split(release, "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok || (k != "ID" && k != "ID_LIKE") {
			continue
		}
		v = strings.Trim(v, `"'`)
		for _, family := range strings.Fields(v) {
			switch strings.ToLower(family) {
			case "arch", "archlinux", "manjaro", "endeavouros", "cachyos", "omarchy":
				return true
			}
		}
	}
	return false
}
