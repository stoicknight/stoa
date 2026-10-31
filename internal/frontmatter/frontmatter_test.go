package frontmatter

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		meta    Meta
		body    string
		hasFM   bool
	}{
		{
			name:    "plain md without frontmatter",
			content: "# hi\n",
			meta:    Meta{},
			body:    "# hi\n",
			hasFM:   false,
		},
		{
			name:    "full frontmatter",
			content: "---\nid: abc-123\ntitle: My note\ntype: journal\ntags: alpha, beta\ncreated: 2026-01-02T03:04:05Z\nupdated: 2026-01-02T03:04:05Z\n---\nbody text\n",
			meta:    Meta{ID: "abc-123", Title: "My note", Type: "journal", Tags: []string{"alpha", "beta"}, Created: "2026-01-02T03:04:05Z", Updated: "2026-01-02T03:04:05Z"},
			body:    "body text\n",
			hasFM:   true,
		},
		{
			name:    "empty tags and hash-style tags",
			content: "---\nid: x\ntitle: t\ntype: note\ntags: #a, #b, []\ncreated: c\nupdated: u\n---\nend\n",
			meta:    Meta{ID: "x", Title: "t", Type: "note", Tags: []string{"a", "b", "[]"}, Created: "c", Updated: "u"},
			body:    "end\n",
			hasFM:   true,
		},
		{
			name:    "missing title stays empty",
			content: "---\nid: x\n---\n",
			meta:    Meta{ID: "x"},
			body:    "",
			hasFM:   true,
		},
		{
			name:    "unclosed separator treated as no frontmatter",
			content: "---\nid: x\nno terminator\n",
			meta:    Meta{},
			body:    "---\nid: x\nno terminator\n",
			hasFM:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, body, hasFM := Parse(tt.content)
			if hasFM != tt.hasFM {
				t.Fatalf("hasFM = %v, want %v", hasFM, tt.hasFM)
			}
			if meta.ID != tt.meta.ID || meta.Title != tt.meta.Title || meta.Type != tt.meta.Type ||
				meta.Created != tt.meta.Created || meta.Updated != tt.meta.Updated {
				t.Fatalf("meta = %+v, want %+v", meta, tt.meta)
			}
			if len(meta.Tags) != len(tt.meta.Tags) {
				t.Fatalf("tags = %v, want %v", meta.Tags, tt.meta.Tags)
			}
			for i := range meta.Tags {
				if meta.Tags[i] != tt.meta.Tags[i] {
					t.Fatalf("tags = %v, want %v", meta.Tags, tt.meta.Tags)
				}
			}
			if body != tt.body {
				t.Fatalf("body = %q, want %q", body, tt.body)
			}
		})
	}
}

func TestRenderRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		meta    Meta
		body    string
		wantErr bool
	}{
		{name: "ok", meta: Meta{ID: "x", Title: "T", Type: "note", Tags: []string{"a"}, Created: "c1", Updated: "c2"}, body: "b"},
		{name: "defaults applied", meta: Meta{ID: "x", Title: "T"}, body: "b"},
		{name: "missing title", meta: Meta{ID: "x"}, body: "b", wantErr: true},
		{name: "missing id", meta: Meta{Title: "T"}, body: "b", wantErr: true},
		{name: "bad tag", meta: Meta{ID: "x", Title: "T", Tags: []string{"a,b"}}, body: "b", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := Render(tt.meta, tt.body)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Render() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			got, body, ok := Parse(out)
			if !ok {
				t.Fatal("parse of rendered output lost frontmatter")
			}
			if got.ID != tt.meta.ID || got.Title != tt.meta.Title {
				t.Fatalf("round trip mismatch: %+v", got)
			}
			if got.Type == "" && tt.meta.Type == "note" {
				t.Fatal("empty type should default to note")
			}
			if body != tt.body {
				t.Fatalf("body = %q, want %q", body, tt.body)
			}
		})
	}
}

func TestTouch(t *testing.T) {
	t.Parallel()
	m := Meta{ID: "x", Title: "T", Created: "keep"}
	got := m.Touch()
	if got.Updated == "" {
		t.Fatal("Touch did not set updated")
	}
	if got.Created != "keep" {
		t.Fatalf("created changed: %q", got.Created)
	}
	_ = strings.TrimSpace // keep strings import used if test is trimmed
}
