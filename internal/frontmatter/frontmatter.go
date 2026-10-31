// Package frontmatter parses and renders the small YAML-like metadata block
// stoa stores at the top of every vault file. The format is intentionally
// restricted to flat "key: value" scalars and a comma-separated tags list,
// so no YAML library is needed. If the format ever outgrows this, swap the
// parser without touching callers.
package frontmatter

import (
	"fmt"
	"strings"
	"time"
)

type Meta struct {
	ID      string
	Title   string
	Type    string
	Tags    []string
	Created string // RFC3339 (or plain date) as written in the file
	Updated string
}

// Parse splits file content into metadata and body. Content without a
// frontmatter block yields a zero Meta and the content as body.
func Parse(content string) (Meta, string, bool) {
	trimmed := strings.TrimLeft(content, "\ufeff")
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 || !isSep(lines[0]) {
		return Meta{}, content, false
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if isSep(lines[i]) {
			end = i
			break
		}
	}
	if end < 0 {
		return Meta{}, content, false
	}

	m := Meta{}
	for _, line := range lines[1:end] {
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "id":
			m.ID = v
		case "title":
			m.Title = v
		case "type":
			m.Type = v
		case "tags":
			m.Tags = parseTagList(v)
		case "created":
			m.Created = v
		case "updated":
			m.Updated = v
		}
	}
	body := strings.Join(lines[end+1:], "\n")
	return m, body, true
}

func isSep(line string) bool {
	return strings.TrimSpace(line) == "---"
}

func parseTagList(v string) []string {
	if v == "" || v == "[]" {
		return nil
	}
	parts := strings.Split(v, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tags = append(tags, strings.TrimPrefix(t, "#"))
		}
	}
	return tags
}

// Render returns the full file content: frontmatter block + body.
func Render(m Meta, body string) (string, error) {
	if strings.TrimSpace(m.Title) == "" {
		return "", fmt.Errorf("frontmatter render: title is required")
	}
	if m.ID == "" {
		return "", fmt.Errorf("frontmatter render: id is required")
	}
	for _, t := range m.Tags {
		if strings.Contains(t, ",") || strings.Contains(t, "\n") {
			return "", fmt.Errorf("frontmatter render: invalid tag %q", t)
		}
	}
	if m.Type == "" {
		m.Type = "note"
	}
	if m.Created == "" {
		m.Created = time.Now().UTC().Format(time.RFC3339)
	}
	if m.Updated == "" {
		m.Updated = m.Created
	}
	head := fmt.Sprintf("---\nid: %s\ntitle: %s\ntype: %s\ntags: %s\ncreated: %s\nupdated: %s\n---\n",
		m.ID, m.Title, m.Type, strings.Join(m.Tags, ", "), m.Created, m.Updated)
	return head + body, nil
}

// New creates a Meta with fresh timestamps.
func New(id, title, typ string, tags []string) Meta {
	now := time.Now().UTC().Format(time.RFC3339)
	return Meta{ID: id, Title: title, Type: typ, Tags: tags, Created: now, Updated: now}
}

// Toucher updates the updated timestamp preserving created.
func (m Meta) Touch() Meta {
	m.Updated = time.Now().UTC().Format(time.RFC3339)
	return m
}
