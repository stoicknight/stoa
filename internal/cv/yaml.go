package cv

import (
	"fmt"
	"strings"
)

// YAML renders the CV schema as YAML — the stable data boundary the Typst
// templates consume via yaml("data.yaml"), and the sibling artifact stoa
// writes next to every PDF for preview/debugging. All scalars are emitted
// as double-quoted YAML strings, so no value can break the structure.
func (c *CV) YAML() string {
	var b strings.Builder
	b.WriteString("position: " + yamlString(c.Position) + "\n")
	b.WriteString("name: " + yamlString(c.Name) + "\n")
	b.WriteString("contacts:\n")
	if len(c.Contacts) == 0 {
		b.WriteString("  []\n")
	}
	for _, ct := range c.Contacts {
		b.WriteString("  - label: " + yamlString(ct.Label) + "\n")
		b.WriteString("    uri: " + yamlString(ct.URI) + "\n")
	}
	b.WriteString("sections:\n")
	if len(c.Sections) == 0 {
		b.WriteString("  []\n")
	}
	for _, s := range c.Sections {
		b.WriteString("  - name: " + yamlString(s.Name) + "\n")
		b.WriteString("    kind: " + yamlString(s.Kind) + "\n")
		b.WriteString("    text: " + yamlString(s.Text) + "\n")
		b.WriteString(yamlList("    ", s.Bullets))
		b.WriteString("    entries:\n")
		if len(s.Entries) == 0 {
			b.WriteString("      []\n")
		}
		for _, e := range s.Entries {
			b.WriteString("      - title: " + yamlString(e.Title) + "\n")
			b.WriteString("        organization: " + yamlString(e.Organization) + "\n")
			b.WriteString("        location: " + yamlString(e.Location) + "\n")
			b.WriteString("        dateRange: " + yamlString(e.DateRange) + "\n")
			b.WriteString(yamlList("        ", e.Bullets))
			b.WriteString(yamlList("        ", e.Details, "details"))
		}
	}
	return b.String()
}

// yamlList renders a named list at the given indent; empty lists become
// "[]" so templates never see a missing key. Without a name (bullets in
// the schema layout) the key is "bullets".
func yamlList(indent string, items []string, names ...string) string {
	name := "bullets"
	if len(names) > 0 {
		name = names[0]
	}
	if len(items) == 0 {
		return indent + name + ": []\n"
	}
	var b strings.Builder
	b.WriteString(indent + name + ":\n")
	for _, item := range items {
		b.WriteString(indent + "  - " + yamlString(item) + "\n")
	}
	return b.String()
}

// yamlString quotes s as a YAML double-quoted scalar, escaping quotes,
// backslashes and control characters.
func yamlString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04X`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
