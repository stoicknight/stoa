package render

import (
	"strings"
	"testing"
)

// letterSample is the canonical cover letter: all frontmatter keys, a
// wrapped prose body with inline markup, salutation first, closing last.
const letterSample = `---
id: 01a1e50b-c929-7382-a493-2b19a78b0245
title: Alexis Skouris
type: letter
position: Senior CloudOps Engineer (Network Engineer)
location: Larnaca, Cyprus
tel: +357 95906760
email: skouris.alexis@gmail.com
company: Wrike
attn: Dmitrii Vlasov - SysOps Team Lead
address: Nicosia, Cyprus
subject: Re: Application for Senior CloudOps Engineer (Network Engineer) - Job Reference 4484921435
---

Dear Hiring Manager,

The Senior CloudOps Engineer (Network Engineer) role describes the job I
have already been doing for years: owning network infrastructure across
hybrid cloud and on-premise environments.

What that practice brings to a team: network and cloud infrastructure
designed as code, with **zero-drift compliance** — see
[my portfolio](https://example.com).

Yours sincerely,
`

// parseLetterTest runs content through the Parse dispatch and asserts
// the result is the Letter schema.
func parseLetterTest(t *testing.T, content string) *Letter {
	t.Helper()
	d, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	l, ok := d.(*Letter)
	if !ok {
		t.Fatalf("Parse() = %T, want *Letter", d)
	}
	return l
}

func TestParseLetterSample(t *testing.T) {
	t.Parallel()
	l := parseLetterTest(t, letterSample)

	if l.Kind() != "letter" {
		t.Errorf("Kind() = %q, want letter", l.Kind())
	}
	if l.Name != "Alexis Skouris" {
		t.Errorf("name = %q", l.Name)
	}
	if l.Position != "Senior CloudOps Engineer (Network Engineer)" {
		t.Errorf("position = %q", l.Position)
	}
	if l.Location != "Larnaca, Cyprus" || l.Tel != "+357 95906760" || l.Email != "skouris.alexis@gmail.com" {
		t.Errorf("sender = %q / %q / %q", l.Location, l.Tel, l.Email)
	}
	if l.Company != "Wrike" {
		t.Errorf("company = %q", l.Company)
	}
	if l.Attn != "Dmitrii Vlasov - SysOps Team Lead" {
		t.Errorf("attn = %q", l.Attn)
	}
	if l.Address != "Nicosia, Cyprus" {
		t.Errorf("address = %q", l.Address)
	}
	if !strings.HasPrefix(l.Subject, "Re: Application for Senior CloudOps Engineer") {
		t.Errorf("subject = %q", l.Subject)
	}

	// salutation first, closing last, wrapped lines joined, inline converted
	if len(l.Paragraphs) != 4 {
		t.Fatalf("paragraphs = %d, want 4: %q", len(l.Paragraphs), l.Paragraphs)
	}
	if l.Paragraphs[0] != `Dear Hiring Manager,` {
		t.Errorf("salutation = %q", l.Paragraphs[0])
	}
	if !strings.Contains(l.Paragraphs[1], "The Senior CloudOps Engineer (Network Engineer) role describes the job I have already been doing for years") {
		t.Errorf("paragraph 1 = %q", l.Paragraphs[1])
	}
	if !strings.Contains(l.Paragraphs[2], `#strong[zero-drift compliance]`) ||
		!strings.Contains(l.Paragraphs[2], `#link("https://example.com")[my portfolio]`) {
		t.Errorf("paragraph 2 = %q", l.Paragraphs[2])
	}
	if l.Paragraphs[3] != `Yours sincerely,` {
		t.Errorf("closing = %q", l.Paragraphs[3])
	}
}

func TestParseLetterMinimal(t *testing.T) {
	t.Parallel()
	// only the required keys: optional sender fields stay empty, subject
	// falls back to the # heading, optional recipient fields empty
	l := parseLetterTest(t, "---\ntitle: Jane Doe\ntype: letter\ncompany: Acme\n---\n\n# Application for the Role\n\nDear Hiring Manager,\n\nI am writing.\n\nYours sincerely,\n")
	if l.Position != "" || l.Location != "" || l.Tel != "" || l.Email != "" || l.Attn != "" || l.Address != "" {
		t.Errorf("optional fields = %+v, want all empty", l)
	}
	if l.Subject != "Application for the Role" {
		t.Errorf("subject = %q, want the # heading text", l.Subject)
	}
	if len(l.Paragraphs) != 3 {
		t.Fatalf("paragraphs = %q", l.Paragraphs)
	}
	for _, p := range l.Paragraphs {
		if strings.Contains(p, "Application for the Role") {
			t.Errorf("heading leaked into the prose: %q", p)
		}
	}
}

func TestParseLetterSubjectResolution(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		front     string // frontmatter lines after type: letter
		body      string
		wantSub   string
		wantParas int
		wantErr   string
	}{
		{
			name:      "frontmatter subject wins, H1 dropped",
			front:     "subject: From frontmatter\n",
			body:      "# From heading\n\nDear X,\n\nText.\n",
			wantSub:   "From frontmatter",
			wantParas: 2,
		},
		{
			name:      "H1 is the subject when frontmatter has none",
			front:     "",
			body:      "# Application\n\nDear X,\n\nText.\n",
			wantSub:   "Application",
			wantParas: 2,
		},
		{
			name:    "no subject anywhere",
			front:   "",
			body:    "Dear X,\n\nText.\n",
			wantErr: "letter needs a subject",
		},
		{
			name:    "two H1s without frontmatter subject",
			front:   "",
			body:    "# One\n\nDear X,\n\nText.\n\n# Two\n",
			wantErr: "more than one # heading",
		},
		{
			name:    "two H1s even with frontmatter subject",
			front:   "subject: From frontmatter\n",
			body:    "# One\n\nDear X,\n\nText.\n\n# Two\n",
			wantErr: "more than one # heading",
		},
		{
			name:      "H1 mid-body still resolves and splits paragraphs",
			front:     "",
			body:      "Dear X,\n\n# Application\n\nText.\n",
			wantSub:   "Application",
			wantParas: 2,
		},
		{
			name:    "empty H1 heading",
			front:   "",
			body:    "#\n\nText.\n",
			wantErr: "letter needs a subject",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			content := "---\ntitle: Jane Doe\ntype: letter\ncompany: Acme\n" + tt.front + "---\n" + tt.body
			l, err := Parse(content)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Parse() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			lt := l.(*Letter)
			if lt.Subject != tt.wantSub {
				t.Errorf("subject = %q, want %q", lt.Subject, tt.wantSub)
			}
			if len(lt.Paragraphs) != tt.wantParas {
				t.Errorf("paragraphs = %d, want %d: %q", len(lt.Paragraphs), tt.wantParas, lt.Paragraphs)
			}
		})
	}
}

func TestParseLetterHeadingsForbidden(t *testing.T) {
	t.Parallel()
	_, err := Parse("---\ntitle: x\ntype: letter\ncompany: Acme\nsubject: S\n---\nDear X,\n\n## Section\n")
	if err == nil || !strings.Contains(err.Error(), "headings are not allowed in a letter") {
		t.Fatalf("Parse() error = %v, want headings-forbidden error", err)
	}
	// the same for deeper headings, and it cites the line number
	_, err = Parse("---\ntitle: x\ntype: letter\ncompany: Acme\nsubject: S\n---\nDear X,\n\nsome text\n### Deep\n")
	if err == nil || !strings.Contains(err.Error(), "### Deep") || !strings.Contains(err.Error(), "line 10") {
		t.Fatalf("Parse() error = %v, want line-numbered headings error", err)
	}
}

func TestParseLetterProseEdges(t *testing.T) {
	t.Parallel()
	content := "---\ntitle: x\ntype: letter\ncompany: Acme\nsubject: S\n---\n" +
		"\n\n   Padded first line  \n\tindented continuation\n\n\n\n" +
		"- starts with a dash\n\n" +
		"C# devs and 5*3 math\n"
	l := parseLetterTest(t, content)
	if len(l.Paragraphs) != 3 {
		t.Fatalf("paragraphs = %q", l.Paragraphs)
	}
	if l.Paragraphs[0] != "Padded first line indented continuation" {
		t.Errorf("wrapped/joined paragraph = %q", l.Paragraphs[0])
	}
	if l.Paragraphs[1] != `\- starts with a dash` {
		t.Errorf("dash-leading line = %q, want escaped text (no bullet parsing in letters)", l.Paragraphs[1])
	}
	if l.Paragraphs[2] != `C\# devs and 5\*3 math` {
		t.Errorf("specials = %q", l.Paragraphs[2])
	}
}

func TestParseLetterErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "missing type",
			content: "---\ntitle: x\ncompany: Acme\n---\nDear X,\n",
			want:    "must contain type",
		},
		{
			name:    "wrong type",
			content: "---\ntitle: x\ntype: memo\ncompany: Acme\n---\nDear X,\n",
			want:    "type must be cv or letter",
		},
		{
			name:    "missing title",
			content: "---\ntype: letter\ncompany: Acme\nsubject: S\n---\nDear X,\n",
			want:    "title in frontmatter",
		},
		{
			name:    "missing company",
			content: "---\ntitle: x\ntype: letter\nsubject: S\n---\nDear X,\n",
			want:    "company in frontmatter",
		},
		{
			name:    "empty body",
			content: "---\ntitle: x\ntype: letter\ncompany: Acme\nsubject: S\n---\n",
			want:    "at least one paragraph",
		},
		{
			name:    "body is only the subject heading",
			content: "---\ntitle: x\ntype: letter\ncompany: Acme\n---\n# Only a heading\n",
			want:    "at least one paragraph",
		},
		{
			name:    "no frontmatter",
			content: "Dear X,\n",
			want:    "frontmatter",
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

func TestLetterYAMLEmission(t *testing.T) {
	t.Parallel()
	y := parseLetterTest(t, letterSample).YAML()
	for _, want := range []string{
		`name: "Alexis Skouris"`,
		`position: "Senior CloudOps Engineer (Network Engineer)"`,
		`location: "Larnaca, Cyprus"`,
		`tel: "+357 95906760"`,
		`email: "skouris.alexis@gmail.com"`,
		`company: "Wrike"`,
		`attn: "Dmitrii Vlasov - SysOps Team Lead"`,
		`address: "Nicosia, Cyprus"`,
		`subject: "Re: Application for Senior CloudOps Engineer (Network Engineer) - Job Reference 4484921435"`,
		`paragraphs:` + "\n  - \"Dear Hiring Manager,\"",
		`  - "Yours sincerely,"`,
	} {
		if !strings.Contains(y, want) {
			t.Errorf("YAML() missing %q in:\n%s", want, y)
		}
	}
}

func TestLetterYAMLEmptyOptionals(t *testing.T) {
	t.Parallel()
	// empty optional keys are still emitted (as ""), so templates never
	// see a missing key; paragraphs list is a real list
	l := parseLetterTest(t, "---\ntitle: x\ntype: letter\ncompany: Acme\nsubject: S\n---\nDear X,\n")
	want := `name: "x"` + "\n" +
		`position: ""` + "\n" +
		`location: ""` + "\n" +
		`tel: ""` + "\n" +
		`email: ""` + "\n" +
		`company: "Acme"` + "\n" +
		`attn: ""` + "\n" +
		`address: ""` + "\n" +
		`subject: "S"` + "\n" +
		"paragraphs:\n  - \"Dear X,\"\n"
	if got := l.YAML(); got != want {
		t.Errorf("YAML() =\n%q\nwant\n%q", got, want)
	}
}

func TestHeadingLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want int
	}{
		{"# h", 1},
		{"## h", 2},
		{"###### h", 6},
		{"####### h", 0}, // seven hashes are not a heading
		{"#", 1},
		{"##", 2},
		{"#tag", 0}, // no space: not a heading (CommonMark)
		{"##foo", 0},
		{"plain", 0},
		{"", 0},
	}
	for _, tt := range tests {
		if got := headingLevel(tt.in); got != tt.want {
			t.Errorf("headingLevel(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
