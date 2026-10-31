// Package markdown contains the tiny amount of markdown awareness stoa needs:
// [[wikilink]] extraction and link-section appends. Wiki syntax matches
// Obsidian, so	stoa vaults stay compatible with it.
package markdown

import (
	"regexp"
	"strings"
)

var wikilinkRe = regexp.MustCompile(`\[\[([^\[\]]+?)\]\]`)

// Link is one [[target]] occurrence found in a document.
type Link struct {
	Target string // text between the brackets, display text stripped
	Kind   string
}

// ExtractLinks returns all wikilink targets in order of appearance,
// de-duplicated (first occurrence wins).
func ExtractLinks(content string) []Link {
	var (
		seen  = map[string]bool{}
		links []Link
	)
	for _, m := range wikilinkRe.FindAllStringSubmatch(content, -1) {
		target := strings.TrimSpace(m[1])
		if t, _, ok := strings.Cut(target, "|"); ok {
			target = strings.TrimSpace(t)
		}
		if target == "" || seen[target] {
			continue
		}
		seen[target] = true
		links = append(links, Link{Target: target})
	}
	return links
}

// AppendLinkSection adds "- [[target]]" under a "## Linked" heading,
// creating the section on first use. It returns the updated body.
func AppendLinkSection(body, target, kind string) (string, bool) {
	if linkAlreadyPresent(body, target) {
		return body, false
	}
	headline := "## Linked"
	line := "- [[" + target + "]]"
	if i := strings.Index(body, headline); i >= 0 {
		end := i + len(headline)
		return body[:end] + "\n" + line + body[end:], true
	}
	trimmed := strings.TrimRight(body, "\n")
	return trimmed + "\n\n" + headline + "\n" + line + "\n", true
}

func linkAlreadyPresent(body, target string) bool {
	return strings.Contains(body, "[["+target+"]]")
}

// SlugChars turns a title into a filesystem-safe slug.
// Non-[a-z0-9] characters collapse to '-', matching stable expectations.
func Slug(title string) string {
	lower := strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	prevDash := false
	for _, r := range lower {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == '[' || r == ']':
			// wikilink-ish punctuation removed entirely
		case !prevDash:
			b.WriteByte('-')
			prevDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > 64 {
		slug = strings.Trim(slug[:64], "-")
	}
	return slug
}
