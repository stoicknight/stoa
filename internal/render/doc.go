// Package render parses stoa markdown documents into fixed schemas and
// renders them to PDF through the Typst CLI.
//
// Each document type (cv, letter) has its own fixed schema — the
// stability boundary between source and output. Reformatting a document
// means writing a new Typst template against the same schema — no Go
// changes. All text fields hold Typst markup produced by the shared
// inline converter (internal/markdown); templates evaluate them with
// eval().
package render

import (
	"fmt"
	"strings"
)

// Doc is a parsed stoa document handed to Typst templates: the YAML
// data boundary (written as data.yaml into the workspace and next to
// every PDF) plus the frontmatter type, which the CLI uses to resolve
// the default theme.
type Doc interface {
	YAML() string
	Kind() string
}

// Parse converts a stoa markdown note into its fixed schema,
// dispatching on the frontmatter type: cv or letter. Anything else is
// an error naming the valid types.
func Parse(content string) (Doc, error) {
	keys, body, baseLine, err := parseFrontmatter(content)
	if err != nil {
		return nil, err
	}
	switch t := strings.ToLower(strings.TrimSpace(keys["type"])); t {
	case "cv":
		return parseCV(keys, body, baseLine)
	case "letter":
		return parseLetter(keys, body, baseLine)
	case "":
		return nil, fmt.Errorf("not a stoa document: frontmatter must contain type: cv or type: letter")
	default:
		return nil, fmt.Errorf("unknown document type %q: frontmatter type must be cv or letter", t)
	}
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
