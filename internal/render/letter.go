package render

import (
	"fmt"
	"strings"

	"github.com/stoicknight/stoa/internal/markdown"
)

// Letter is the fixed schema handed to Typst templates for cover
// letters. The body is the letter prose: the salutation and the closing
// line are ordinary paragraphs written by the user — stoa adds only the
// sign-off gap and the name in the template layer. Attn and Address may
// be empty; templates must skip empty lines.
type Letter struct {
	Name       string   // frontmatter "title" (the sender name)
	Position   string   // may be ""
	Location   string   // may be ""
	Tel        string   // may be ""
	Email      string   // may be ""
	Company    string   // the recipient
	Attn       string   // may be ""
	Address    string   // may be ""
	Subject    string   // frontmatter "subject", else the single # heading
	Paragraphs []string // body paragraphs, in file order (Typst markup)
}

// Kind implements Doc.
func (l *Letter) Kind() string { return "letter" }

// parseLetter builds the Letter schema from pre-split frontmatter and
// body. Required: title (the sender name), company, a subject, and at
// least one body paragraph. The subject comes from the frontmatter
// subject: key, falling back to a single # heading in the body; when
// both exist the frontmatter wins and the heading is dropped.
func parseLetter(keys map[string]string, body string, baseLine int) (*Letter, error) {
	l := &Letter{
		Name:     strings.TrimSpace(keys["title"]),
		Position: strings.TrimSpace(keys["position"]),
		Location: strings.TrimSpace(keys["location"]),
		Tel:      strings.TrimSpace(keys["tel"]),
		Email:    strings.TrimSpace(keys["email"]),
		Company:  strings.TrimSpace(keys["company"]),
		Attn:     strings.TrimSpace(keys["attn"]),
		Address:  strings.TrimSpace(keys["address"]),
		Subject:  strings.TrimSpace(keys["subject"]),
	}
	if l.Name == "" {
		return nil, fmt.Errorf("letter note needs a title in frontmatter (the sender name)")
	}
	if l.Company == "" {
		return nil, fmt.Errorf("letter note needs a company in frontmatter (the recipient)")
	}
	if err := l.parseParagraphs(body, baseLine); err != nil {
		return nil, err
	}
	if l.Subject == "" {
		return nil, fmt.Errorf("letter needs a subject: add subject: to the frontmatter or a single # heading in the body")
	}
	if len(l.Paragraphs) == 0 {
		return nil, fmt.Errorf("letter needs at least one paragraph of body text (the prose is the letter)")
	}
	return l, nil
}

// parseParagraphs splits the body into blank-line-separated paragraphs
// (wrapped lines join with spaces) and converts each through the inline
// converter. A single # heading supplies the subject when the
// frontmatter has none; any other heading is an error — the letter body
// is prose only.
func (l *Letter) parseParagraphs(body string, baseLine int) error {
	var (
		lines  []string // the paragraph being collected
		h1Seen bool
	)
	flush := func() {
		if len(lines) > 0 {
			l.Paragraphs = append(l.Paragraphs, markdown.Inline(strings.Join(lines, " ")))
			lines = nil
		}
	}
	for i, raw := range strings.Split(body, "\n") {
		lineNo := baseLine + i
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			flush()
			continue
		}
		switch n := headingLevel(trimmed); {
		case n > 1:
			return fmt.Errorf("line %d: headings are not allowed in a letter (the body is prose only): %q", lineNo, trimmed)
		case n == 1:
			flush() // a heading breaks the paragraph it interrupts
			if h1Seen {
				return fmt.Errorf("line %d: more than one # heading in the letter body (the first one is the subject)", lineNo)
			}
			h1Seen = true
			if l.Subject == "" { // frontmatter subject wins; a stray H1 is dropped
				l.Subject = strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			}
		default:
			lines = append(lines, trimmed)
		}
	}
	flush()
	return nil
}

// headingLevel reports the ATX heading level of a trimmed line
// (CommonMark: the #s must be followed by a space or end the line), or
// 0 when the line is not a heading.
func headingLevel(line string) int {
	n := 0
	for n < len(line) && line[n] == '#' {
		n++
	}
	if n == 0 || n > 6 {
		return 0
	}
	if n == len(line) || line[n] == ' ' {
		return n
	}
	return 0
}

// YAML renders the Letter schema as YAML — same rules as the CV schema:
// all scalars double-quoted, lists as [] when empty, so templates never
// see a missing key.
func (l *Letter) YAML() string {
	var b strings.Builder
	for _, kv := range []struct{ key, val string }{
		{"name", l.Name},
		{"position", l.Position},
		{"location", l.Location},
		{"tel", l.Tel},
		{"email", l.Email},
		{"company", l.Company},
		{"attn", l.Attn},
		{"address", l.Address},
		{"subject", l.Subject},
	} {
		b.WriteString(kv.key + ": " + yamlString(kv.val) + "\n")
	}
	b.WriteString(yamlList("", l.Paragraphs, "paragraphs"))
	return b.String()
}
