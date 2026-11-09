package cv

import (
	"strings"
	"testing"
)

// sample is a cv-example-style note covering every section kind.
const sample = `---
id: 019518f5-9a34-7abc-8f17-6b3f2a5f0c1d
title: Jane Doe
type: cv
position: Cloud & Automation Engineer
location: London, UK
tel: +1 555 555 5555
email: jane.doe@example.com
linkedin: jane-doe
github: janedoe
website: example.com
---

## Summary

Cloud and automation engineer with 4+ years designing, automating, and
operating secure infrastructure at scale.

## Technical Skills

- **Cloud & IaC:** AWS, Google Cloud, Terraform/OpenTofu, Ansible
- **Containers & Orchestration:** Docker, Kubernetes, OpenShift
- **Languages:** English (native), German (fluent)

## Professional Experience

### Freelance IT Consultant & Software Engineer

**Self-employed** — Remote
*Aug 2025 – Present*

- Delivered and handed over 2 completed client engagements with **GitOps** CI/CD.
- Developing two products of my own (see [my portfolio](https://example.com)).

### Head of Operations & IT

**Acme Logistics Services Ltd.** — London, UK
*Feb 2024 – Sep 2025*

- Ran daily operations and IT for a logistics/services company.

## Projects

### Stoa — CLI Notes & Documentation Tool *(MVP complete)*

- Single integrated binary for notes, incident reports and documentation.

## Certifications

- Red Hat Certified System Administrator (RHCSA)
`

func TestParseSample(t *testing.T) {
	t.Parallel()
	c, err := Parse(sample)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if c.Position != "Cloud & Automation Engineer" || c.Name != "Jane Doe" {
		t.Errorf("header = %q / %q", c.Position, c.Name)
	}

	// contacts: fixed order, all links except location
	wantContacts := []Contact{
		{Label: "London, UK"},
		{Label: "+1 555 555 5555", URI: "tel:+15555555555"},
		{Label: "jane.doe@example.com", URI: "mailto:jane.doe@example.com"},
		{Label: "linkedin.com/in/jane-doe", URI: "https://linkedin.com/in/jane-doe"},
		{Label: "github.com/janedoe", URI: "https://github.com/janedoe"},
		{Label: "example.com", URI: "https://example.com"},
	}
	if len(c.Contacts) != len(wantContacts) {
		t.Fatalf("contacts = %d entries, want %d", len(c.Contacts), len(wantContacts))
	}
	for i, w := range wantContacts {
		if c.Contacts[i] != w {
			t.Errorf("contacts[%d] = %+v, want %+v", i, c.Contacts[i], w)
		}
	}

	// section order preserved
	wantOrder := []string{"Summary", "Technical Skills", "Professional Experience", "Projects", "Certifications"}
	if len(c.Sections) != len(wantOrder) {
		t.Fatalf("got %d sections (%v), want %v", len(c.Sections), sectionNames(c), wantOrder)
	}
	for i, name := range wantOrder {
		if c.Sections[i].Name != name {
			t.Errorf("sections[%d].Name = %q, want %q", i, c.Sections[i].Name, name)
		}
	}
	wantKinds := []string{kindParagraph, kindBullets, kindEntries, kindEntries, kindBullets}
	for i, k := range wantKinds {
		if c.Sections[i].Kind != k {
			t.Errorf("sections[%d].Kind = %q, want %q", i, c.Sections[i].Kind, k)
		}
	}

	// paragraph text joined, inline converted
	if got := c.Sections[0].Text; !strings.Contains(got, "operating secure infrastructure at scale.") {
		t.Errorf("summary text = %q", got)
	}

	// bullets keep order and convert bold
	if got := c.Sections[1].Bullets[0]; !strings.Contains(got, `#strong[Cloud & IaC:]`) {
		t.Errorf("skills bullet = %q", got)
	}

	// first experience entry
	e := c.Sections[2].Entries[0]
	if e.Title == "" || e.Organization != "Self-employed" {
		t.Errorf("entry title/org = %q / %q", e.Title, e.Organization)
	}
	if e.Location != "Remote" {
		t.Errorf("entry location = %q", e.Location)
	}
	if e.DateRange != "Aug 2025 – Present" {
		t.Errorf("entry dateRange = %q", e.DateRange)
	}
	if len(e.Bullets) != 2 {
		t.Fatalf("entry bullets = %d, want 2", len(e.Bullets))
	}
	if !strings.Contains(e.Bullets[0], `#strong[GitOps]`) {
		t.Errorf("bullet inline = %q", e.Bullets[0])
	}
	if !strings.Contains(e.Bullets[1], `#link("https://example.com")`) {
		t.Errorf("bullet link = %q", e.Bullets[1])
	}

	// project entry: italic badge in title survives conversion
	p := c.Sections[3].Entries[0]
	if !strings.Contains(p.Title, "_*(MVP complete)*_") && !strings.Contains(p.Title, "(MVP complete)") {
		t.Errorf("project title = %q", p.Title)
	}

	// certifications order
	if c.Sections[4].Bullets[0] != "Red Hat Certified System Administrator (RHCSA)" {
		t.Errorf("cert[0] = %q", c.Sections[4].Bullets[0])
	}
}

func TestParseErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "missing type",
			content: "---\ntitle: x\n---\n## A\n- b\n",
			want:    "type: cv",
		},
		{
			name:    "wrong type",
			content: "---\ntitle: x\ntype: note\n---\n## A\n- b\n",
			want:    "type: cv",
		},
		{
			name:    "missing title",
			content: "---\ntype: cv\n---\n## A\n- b\n",
			want:    "title",
		},
		{
			name:    "no frontmatter",
			content: "## A\n- b\n",
			want:    "frontmatter",
		},
		{
			name:    "h1 in body",
			content: "---\ntitle: x\ntype: cv\n---\n# Title\n## A\n- b\n",
			want:    "top-level heading",
		},
		{
			name:    "text before section",
			content: "---\ntitle: x\ntype: cv\n---\nfree text\n## A\n- b\n",
			want:    "outside a section",
		},
		{
			name:    "mixed bullets and prose",
			content: "---\ntitle: x\ntype: cv\n---\n## A\n- b\nprose\n",
			want:    "mixes bullets",
		},
		{
			name:    "plain lines in entry section",
			content: "---\ntitle: x\ntype: cv\n---\n## A\nstray prose\n### E\n**Org** — loc\n- b\n",
			want:    "mixes ### entries",
		},
		{
			name:    "entry outside section",
			content: "---\ntitle: x\ntype: cv\n---\n### E\n- b\n",
			want:    "outside a section",
		},
		{
			name:    "duplicate org line",
			content: "---\ntitle: x\ntype: cv\n---\n## A\n### E\n**O1**\n**O2**\n",
			want:    "duplicate organization",
		},
		{
			name:    "duplicate date",
			content: "---\ntitle: x\ntype: cv\n---\n## A\n### E\n2020 – 2021\n*2021 – 2022*\n",
			want:    "duplicate date range",
		},
		{
			name:    "empty section heading",
			content: "---\ntitle: x\ntype: cv\n---\n## \n",
			want:    "needs a name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tt.content)
			if err == nil {
				t.Fatalf("Parse() = nil error, want %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestParseLineNumbers(t *testing.T) {
	t.Parallel()
	// "stray" is body line 2 of section A (source line 6)
	_, err := Parse("---\ntitle: x\ntype: cv\n---\n## A\nstray\n### E\n**Org**\n- b\n")
	if err == nil || !strings.Contains(err.Error(), "mixes ### entries") || !strings.Contains(err.Error(), "line 6") {
		t.Errorf("error = %v, want line-numbered section mix error", err)
	}
}

func TestParseEntryDetailsAndDateOnOrgLine(t *testing.T) {
	t.Parallel()
	content := "---\ntitle: x\ntype: cv\n---\n" +
		"## Education\n" +
		"### Diplom in Electrical Engineering (equivalent to Master's)\n" +
		"**TU Dresden** — *Oct 2010 – Apr 2016*\n" +
		"Networking & Communications · Signal Processing\n"
	c, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	e := c.Sections[0].Entries[0]
	if e.Organization != "TU Dresden" {
		t.Errorf("org = %q", e.Organization)
	}
	if e.Location != "" {
		t.Errorf("location = %q, want empty", e.Location)
	}
	if e.DateRange != "Oct 2010 – Apr 2016" {
		t.Errorf("dateRange = %q", e.DateRange)
	}
	if len(e.Details) != 1 || !strings.Contains(e.Details[0], "Networking") {
		t.Errorf("details = %v", e.Details)
	}
}

func TestInline(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello world", "hello world"},
		{"bold", "**Go** and SQL", `#strong[Go] and SQL`},
		{"italic", "*in progress*", "_in progress_"},
		{"link", "[Projects](https://example.com)", `#link("https://example.com")[Projects]`},
		{"prefix before link", "see [my site](https://example.com) now", `see #link("https://example.com")[my site] now`},
		{"bold before link", "**Led** [team](https://x.io)", `#strong[Led] #link("https://x.io")[team]`},
		{"escape hash", "C# and #tag", `C\# and \#tag`},
		{"escape lone star", "5*3", `5\*3`},
		{"escape underscore", "snake_case", `snake\_case`},
		{"escape quotes", "Master's \"tool\"", `Master\'s \"tool\"`},
		{"escape brackets", "array[0]", `array\[0\]`},
		{"leading dash", "- item", `\- item`},
		{"leading equals", "= note", `\= note`},
		{"ampersand literal", "R&D", "R&D"},
		{"mailto url escaping", `[mail](mailto:a@b.c)`, `#link("mailto:a@b.c")[mail]`},
		{"code", "built with `Go 1.23`", `built with #raw("Go 1.23")`},
		{"code markup-safe inside", "see `slice[0]` here", `see #raw("slice[0]") here`},
		{"code keeps stars literal", "`*not emph*`", `#raw("*not emph*")`},
		{"bold with code inside", "**Uses `Go` daily**", `#strong[Uses #raw("Go") daily]`},
		{"code after link", "[Site](https://x.io) with `code`", `#link("https://x.io")[Site] with #raw("code")`},
		{"unmatched backtick", "cost ~100` each", "cost \\~100\\` each"},
		{"non-greedy pair", "a `b` c `d` e", `a #raw("b") c #raw("d") e`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := inline(tt.in); got != tt.want {
				t.Errorf("inline(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestYAMLEmission(t *testing.T) {
	t.Parallel()
	c, err := Parse(sample)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	y := c.YAML()
	for _, want := range []string{
		`position: "Cloud & Automation Engineer"`,
		`name: "Jane Doe"`,
		`uri: "tel:+15555555555"`,
		`kind: "paragraph"`,
		`kind: "entries"`,
		`details: []`,       // details always present
		`- name: "Summary"`, // sections keep file order
		`dateRange: "Aug 2025 – Present"`,
	} {
		if !strings.Contains(y, want) {
			t.Errorf("YAML() missing %q in:\n%s", want, y)
		}
	}
}

func TestYAMLStringEscaping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{`plain`, `"plain"`},
		{`say "hi"`, `"say \"hi\""`},
		{`back\slash`, `"back\\slash"`},
		{"two\nlines", `"two\nlines"`},
		{"tab\there", `"tab\there"`},
	}
	for _, tt := range tests {
		if got := yamlString(tt.in); got != tt.want {
			t.Errorf("yamlString(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSchemaPath(t *testing.T) {
	t.Parallel()
	if got := SchemaPath("/vault/cv.md.pdf"); got != "/vault/cv.md.yaml" {
		t.Errorf("SchemaPath = %q", got)
	}
	if got := SchemaPath("cv.pdf"); got != "cv.yaml" {
		t.Errorf("SchemaPath = %q", got)
	}
}

func TestArchLike(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		release string
		want    bool
	}{
		{"omarchy", "NAME=\"Omarchy\"\nID=omarchy\nID_LIKE=arch\n", true},
		{"arch quoted", "NAME=\"Arch Linux\"\nID=\"arch\"\n", true},
		{"manjaro", "ID=manjaro\nID_LIKE=\"arch\"\n", true},
		{"debian", "NAME=\"Debian GNU/Linux\"\nID=debian\n", false},
		{"empty", "", false},
		{"garbage", "not=release\ndata===\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := archLike(tt.release); got != tt.want {
				t.Errorf("archLike(%q) = %v, want %v", tt.release, got, tt.want)
			}
		})
	}
}

func sectionNames(c *CV) []string {
	names := make([]string, len(c.Sections))
	for i, s := range c.Sections {
		names[i] = s.Name
	}
	return names
}
