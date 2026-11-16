package markdown

import "testing"

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
			if got := Inline(tt.in); got != tt.want {
				t.Errorf("Inline(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
