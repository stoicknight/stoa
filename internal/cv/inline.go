package cv

import "strings"

// inline converts restricted markdown inline markup to Typst markup:
// **strong** → #strong[...], *emphasis* → _..._, `code` → #raw("...")
// and [text](url) → #link(...). Everything else is escaped so Typst
// renders it literally. Nesting is deliberately unsupported (no
// bold inside links etc.) to keep the converter small and predictable;
func inline(s string) string {
	var b strings.Builder
	rest := s
	for rest != "" {
		prefix, text, url, after, ok := cutLink(rest)
		if !ok {
			b.WriteString(bold(rest))
			break
		}
		b.WriteString(bold(prefix))
		b.WriteString(`#link("` + escapeString(url) + `")` + "[" + bold(text) + "]")
		rest = after
	}
	return b.String()
}

// cutLink finds the first [text](url). Brackets inside the text or the
// URL are not supported (escaped input is rare in CVs; predictability
// wins). Returns the text before the link, the link text, the URL and
// the remainder after the link.
func cutLink(s string) (prefix, text, url, after string, ok bool) {
	start := strings.Index(s, "[")
	if start < 0 {
		return s, "", "", s, false
	}
	mid := strings.Index(s[start:], "](")
	if mid < 0 {
		return s, "", "", s, false
	}
	mid += start
	end := strings.Index(s[mid+2:], ")")
	if end < 0 {
		return s, "", "", s, false
	}
	end += mid + 2
	return s[:start], s[start+1 : mid], s[mid+2 : end], s[end+1:], true
}

// bold splits on **strong** spans, then handles `code` and *emphasis*
// in the rest.
func bold(s string) string {
	i := strings.Index(s, "**")
	if i < 0 {
		return code(s)
	}
	j := strings.Index(s[i+2:], "**")
	if j < 0 { // unmatched: keep as literal text
		return escapeMarkup(s[:i+2]) + code(s[i+2:])
	}
	inner := s[i+2 : i+2+j]
	return escapeMarkup(s[:i]) + "#strong[" + code(inner) + "]" + bold(s[i+2+j+2:])
}

// code splits on `inline code` spans (checked after bold, so a span
// between ** markers is treated as bold). Code contents become
// #raw("..."): string-escaped, not markup-escaped, so characters like
// * and _ render literally. Text without code falls through to emph.
// The span is non-greedy: a lone backtick stays literal.
func code(s string) string {
	i := strings.IndexByte(s, '`')
	if i < 0 {
		return emph(s)
	}
	j := strings.IndexByte(s[i+1:], '`')
	if j < 0 { // unmatched: keep as literal text
		return emph(s[:i]) + "\\`" + code(s[i+1:])
	}
	inner := s[i+1 : i+1+j]
	return emph(s[:i]) + "#raw(\"" + escapeString(inner) + "\")" + code(s[i+j+2:])
}

// emph splits on *emphasis* spans in text that has no bold or code
// left.
func emph(s string) string {
	i := strings.Index(s, "*")
	if i < 0 {
		return escapeMarkup(s)
	}
	j := strings.Index(s[i+1:], "*")
	if j < 0 {
		return escapeMarkup(s[:i+1]) + escapeMarkup(s[i+1:])
	}
	inner := s[i+1 : i+1+j]
	return escapeMarkup(s[:i]) + "_" + escapeMarkup(inner) + "_" + emph(s[i+j+2:])
}

// escapeMarkup escapes Typst markup specials so text renders literally.
// Leading list/enum/term/heading markers are neutralized too, because a
// converted string is evaluated as its own markup source.
func escapeMarkup(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i == 0 {
			switch r {
			case '-', '+', '=', '/', '~':
				b.WriteRune('\\')
			}
		}
		switch r {
		case '\\', '#', '$', '*', '_', '`', '[', ']', '@', '~', '"', '\'', '<', '>':
			b.WriteRune('\\')
		case '\n':
			b.WriteString("\\ ")
			continue
		case '\t':
			b.WriteString("\\ ")
			continue
		case '\r':
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// escapeString escapes a value used inside a Typst string literal.
func escapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}
