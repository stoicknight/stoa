// Package cv parses CV-type markdown notes into a fixed schema and renders
// them to PDF through the Typst CLI.
//
// The schema (CV) is the stability boundary between source and output:
// reformatting a CV means writing a new Typst template against the same
// schema — no Go changes. All text fields hold Typst markup produced by
// the inline converter; templates evaluate them with eval().
package cv

import (
	"fmt"
	"regexp"
	"strings"
)

// Contact is one frontmatter contact rendered in the header line.
type Contact struct {
	Label string // display text, e.g. "+1 555 555 5555"
	URI   string // link target; empty means plain text (location)
}

// Entry is one ### item inside an entry section (jobs, projects, degrees).
type Entry struct {
	Title        string
	Organization string   // the **bold** part of the org line
	Location     string   // text after the em dash on the org line
	DateRange    string   // bare or italic line containing a year
	Bullets      []string // "- " lines
	Details      []string // stray bare lines (e.g. study focus areas)
}

// Section is one ## block. Kind is "paragraph", "bullets" or "entries",
// inferred from the section's content shape.
type Section struct {
	Name    string
	Kind    string
	Text    string   // paragraph kind: all non-bullet lines joined with spaces
	Bullets []string // bullets kind, in file order
	Entries []Entry  // entries kind, in file order
}

// CV is the fixed schema handed to Typst templates.
type CV struct {
	Position string // frontmatter "position"
	Name     string // frontmatter "title"
	Contacts []Contact
	Sections []Section // in file order
}

const (
	kindParagraph = "paragraph"
	kindBullets   = "bullets"
	kindEntries   = "entries"
)

// Parse converts a CV markdown note into the schema. The input must follow
// the restricted CV format (see README): flat frontmatter with type: cv,
// then "## Section" blocks whose entries are "### Heading" plus an
// organization line, an optional date line, and "- " bullets. Anything
// else is an error naming the line number.
func Parse(content string) (*CV, error) {
	keys, body, baseLine, err := parseFrontmatter(content)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(keys["type"], "cv") {
		return nil, fmt.Errorf("not a CV note: frontmatter must contain type: cv")
	}
	c := &CV{Position: strings.TrimSpace(keys["position"])}
	c.Name = strings.TrimSpace(keys["title"])
	if c.Name == "" {
		return nil, fmt.Errorf("CV note needs a title in frontmatter (the applicant name)")
	}
	c.Contacts = buildContacts(keys)
	if strings.TrimSpace(body) != "" {
		c.Sections, err = parseSections(body, baseLine)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

// parseFrontmatter splits the flat "key: value" block off the body. Only
// flat scalars are supported, mirroring internal/frontmatter's philosophy.
// It returns the 1-based source line of the first body line.
func parseFrontmatter(content string) (keys map[string]string, body string, baseLine int, err error) {
	trimmed := strings.TrimLeft(content, "\ufeff")
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, "", 0, fmt.Errorf("missing frontmatter block (--- ... ---)")
	}
	keys = map[string]string{}
	end := -1
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" {
			end = i
			break
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			return nil, "", 0, fmt.Errorf("frontmatter line %d: not a flat \"key: value\" scalar: %q", i+1, line)
		}
		keys[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if end < 0 {
		return nil, "", 0, fmt.Errorf("frontmatter block not closed (--- ... ---)")
	}
	return keys, strings.Join(lines[end+1:], "\n"), end + 2, nil
}

// contactOrder fixes the header contact order: location first, then
// phone, email, linkedin, github, website — mirroring the draft CV.
var contactOrder = []string{"location", "tel", "email", "linkedin", "github", "website"}

// buildContacts turns frontmatter values into ordered contact links.
// Everything except location becomes a link (tel:/mailto:/https:).
func buildContacts(keys map[string]string) []Contact {
	var out []Contact
	for _, key := range contactOrder {
		v := strings.TrimSpace(keys[key])
		if v == "" {
			continue
		}
		switch key {
		case "location":
			out = append(out, Contact{Label: v})
		case "tel":
			out = append(out, Contact{Label: v, URI: "tel:" + strings.ReplaceAll(v, " ", "")})
		case "email":
			out = append(out, Contact{Label: v, URI: "mailto:" + v})
		case "linkedin", "github":
			label, uri := socialLink(key, v)
			out = append(out, Contact{Label: label, URI: uri})
		case "website":
			uri := v
			if !strings.HasPrefix(uri, "http://") && !strings.HasPrefix(uri, "https://") {
				uri = "https://" + uri
			}
			label := strings.TrimPrefix(strings.TrimPrefix(uri, "https://"), "http://")
			out = append(out, Contact{Label: label, URI: uri})
		}
	}
	return out
}

// socialLink normalizes linkedin/github values (bare handle, "host/path"
// form, or full URL) into a short display label and a https URI.
func socialLink(key, v string) (label, uri string) {
	host := key + ".com"
	s := v
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	s = strings.TrimPrefix(s, "www.")
	s = strings.TrimPrefix(s, host+"/")
	s = strings.TrimSuffix(s, "/")
	if s == "" {
		return v, "https://" + host
	}
	if key == "linkedin" && !strings.HasPrefix(s, "in/") {
		s = "in/" + s
	}
	if key == "linkedin" {
		return host + "/in/" + strings.TrimPrefix(s, "in/"), "https://" + host + "/" + s
	}
	return host + "/" + s, "https://" + host + "/" + s
}

// yearRe matches a year token in a line; used to spot date ranges.
var yearRe = regexp.MustCompile(`(?:^|[^\w])(?:19|20)\d{2}(?:[^\w]|$)`)

// looksLikeDate reports whether a bare or italic entry line is a date
// range (it must contain a 4-digit year in 1900–2099).
func looksLikeDate(line string) bool {
	return yearRe.MatchString(line)
}

// rawLine is one line of a section body, classified.
type rawLine struct {
	kind   string // "bullet" or "text"
	text   string // content after the "- " marker for bullets
	lineNo int    // 1-based source line
}

// rawEntry is one "### Heading" block with its raw lines.
type rawEntry struct {
	title string
	lines []rawLine
}

// rawSection collects the body of one "## Section".
type rawSection struct {
	name      string
	lineNo    int
	entries   []rawEntry
	lines     []rawLine // lines outside entries
	hasBullet bool
	hasText   bool
}

// parseSections splits the body into ordered sections and infers each
// section kind. Strictness is deliberate: the CV format is fixed, and a
// clear error beats a silently mangled layout.
func parseSections(body string, baseLine int) ([]Section, error) {
	var (
		out    []Section
		cur    *rawSection
		entIdx = -1 // index of the current entry in cur.entries
	)
	finish := func() error {
		if cur == nil {
			return nil
		}
		s, err := cur.section()
		if err != nil {
			return err
		}
		out = append(out, s)
		return nil
	}
	for i, raw := range strings.Split(body, "\n") {
		lineNo := baseLine + i
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "####"):
			return nil, fmt.Errorf("line %d: only ## and ### headings are supported in a CV: %q", lineNo, trimmed)
		case strings.HasPrefix(trimmed, "###"):
			if cur == nil {
				return nil, fmt.Errorf("line %d: ### entry outside a section (add a ## heading first)", lineNo)
			}
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "###"))
			if title == "" {
				return nil, fmt.Errorf("line %d: entry heading needs a title", lineNo)
			}
			cur.entries = append(cur.entries, rawEntry{title: title})
			entIdx = len(cur.entries) - 1
		case strings.HasPrefix(trimmed, "##"):
			if err := finish(); err != nil {
				return nil, err
			}
			name := strings.TrimSpace(strings.TrimPrefix(trimmed, "##"))
			if name == "" {
				return nil, fmt.Errorf("line %d: section heading needs a name", lineNo)
			}
			cur, entIdx = &rawSection{name: name, lineNo: lineNo}, -1
		case strings.HasPrefix(trimmed, "#"):
			return nil, fmt.Errorf("line %d: top-level heading not allowed in a CV (put the position in frontmatter): %q", lineNo, trimmed)
		default:
			if cur == nil {
				return nil, fmt.Errorf("line %d: text outside a section (add a ## heading first): %q", lineNo, trimmed)
			}
			kind, text := "text", trimmed
			if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
				kind, text = "bullet", strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			}
			line := rawLine{kind: kind, text: text, lineNo: lineNo}
			if entIdx >= 0 {
				cur.entries[entIdx].lines = append(cur.entries[entIdx].lines, line)
			} else {
				cur.lines = append(cur.lines, line)
				if kind == "bullet" {
					cur.hasBullet = true
				} else {
					cur.hasText = true
				}
			}
		}
	}
	if err := finish(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no ## sections found in CV body")
	}
	return out, nil
}

// section classifies a raw section into the schema. Mixing shapes inside
// one section is an error with the offending line number.
func (r *rawSection) section() (Section, error) {
	s := Section{Name: r.name}
	if len(r.entries) > 0 {
		if r.hasBullet || r.hasText {
			return s, fmt.Errorf("line %d: section %q mixes ### entries and plain lines (entry sections take ### blocks only)", r.firstStrayLine(), r.name)
		}
		s.Kind = kindEntries
		for _, e := range r.entries {
			se, err := e.entry()
			if err != nil {
				return s, fmt.Errorf("section %q: %w", r.name, err)
			}
			s.Entries = append(s.Entries, se)
		}
		return s, nil
	}
	switch {
	case r.hasBullet && r.hasText:
		return s, fmt.Errorf("line %d: section %q mixes bullets and prose (use one shape per section)", r.firstStrayLine(), r.name)
	case r.hasBullet:
		s.Kind = kindBullets
		for _, l := range r.lines {
			s.Bullets = append(s.Bullets, inline(l.text))
		}
		return s, nil
	default:
		s.Kind = kindParagraph
		for _, l := range r.lines {
			if s.Text != "" {
				s.Text += " "
			}
			s.Text += inline(l.text)
		}
		return s, nil
	}
}

// firstStrayLine returns the source line of the first non-entry line in
// the section (for error messages), or the section heading itself.
func (r *rawSection) firstStrayLine() int {
	for _, l := range r.lines {
		return l.lineNo
	}
	return r.lineNo
}

// entry classifies the lines of one raw entry into org/location, date
// range, bullets and details.
func (e *rawEntry) entry() (Entry, error) {
	out := Entry{Title: inline(e.title)}
	for _, l := range e.lines {
		switch {
		case l.kind == "bullet":
			out.Bullets = append(out.Bullets, inline(l.text))
		case strings.HasPrefix(l.text, "**") && strings.Contains(l.text, "**"):
			if out.Organization != "" {
				return out, fmt.Errorf("line %d: duplicate organization line", l.lineNo)
			}
			org, loc, date, err := splitOrgLine(l.text)
			if err != nil {
				return out, fmt.Errorf("line %d: %w", l.lineNo, err)
			}
			out.Organization, out.Location, out.DateRange = org, loc, date
		case l.kind == "text" && looksLikeDate(l.text):
			if out.DateRange != "" {
				return out, fmt.Errorf("line %d: duplicate date range", l.lineNo)
			}
			out.DateRange = inline(stripMarkup(l.text))
		default:
			out.Details = append(out.Details, inline(l.text))
		}
	}
	return out, nil
}

// splitOrgLine parses "**Organization** — Location" (or a date in place of
// the location, education-style). The em/en dash separates the parts.
func splitOrgLine(line string) (org, loc, date string, err error) {
	rest, ok := strings.CutPrefix(line, "**")
	if !ok {
		return "", "", "", fmt.Errorf("organization line must start with **bold**")
	}
	var orgPart string
	orgPart, rest, ok = strings.Cut(rest, "**")
	if !ok || strings.TrimSpace(orgPart) == "" {
		return "", "", "", fmt.Errorf("organization line must start with **bold**")
	}
	rest = strings.TrimSpace(rest)
	for _, dash := range []string{"—", "–"} {
		if tail, ok := strings.CutPrefix(rest, dash); ok {
			rest = strings.TrimSpace(tail)
			if looksLikeDate(stripMarkup(rest)) {
				date = stripMarkup(rest)
			} else {
				loc = stripMarkup(rest)
			}
			break
		}
	}
	return strings.TrimSpace(orgPart), loc, date, nil
}

// stripMarkup removes wrapping **bold** and *italic* markers from a line
// used as a plain value (dates, locations).
func stripMarkup(s string) string {
	s = strings.TrimSpace(s)
	for _, marker := range []string{"**", "*"} {
		if !strings.HasPrefix(s, marker) {
			continue
		}
		if inner, _, ok := strings.Cut(strings.TrimPrefix(s, marker), marker); ok {
			s = inner
		}
	}
	return strings.TrimSpace(s)
}
