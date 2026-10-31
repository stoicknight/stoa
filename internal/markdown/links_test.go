package markdown

import (
	"reflect"
	"strings"
	"testing"
)

func TestExtractLinks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want []Link
	}{
		{name: "none", in: "plain [md](link)"},
		{name: "simple", in: "see [[Target]] ok", want: []Link{{Target: "Target"}}},
		{name: "duplicate dropped", in: "[[A]] and [[A]]", want: []Link{{Target: "A"}}},
		{name: "display text stripped", in: "[[target|display]]", want: []Link{{Target: "target"}}},
		{name: "surrounding spaces", in: "[[ Target Two ]]", want: []Link{{Target: "Target Two"}}},
		{name: "empty skipped", in: "[[]] [[A]]", want: []Link{{Target: "A"}}},
		{name: "multiple", in: "[[A]]\n[[b c]]\n[[D|alias]]", want: []Link{{Target: "A"}, {Target: "b c"}, {Target: "D"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractLinks(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ExtractLinks() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAppendLinkSection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    string
		target  string
		want    string
		wantFlg bool
	}{
		{
			name:    "creates section",
			body:    "body\n",
			target:  "Other",
			want:    "body\n\n## Linked\n- [[Other]]\n",
			wantFlg: true,
		},
		{
			name:    "existing section appended",
			body:    "body\n\n## Linked\n- [[One]]\n",
			target:  "Other",
			want:    "body\n\n## Linked\n- [[Other]]\n- [[One]]\n",
			wantFlg: true,
		},
		{
			name:    "already present",
			body:    "body\n\n## Linked\n- [[Other]]\n",
			target:  "Other",
			want:    "body\n\n## Linked\n- [[Other]]\n",
			wantFlg: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := AppendLinkSection(tt.body, tt.target, "ref")
			if changed != tt.wantFlg {
				t.Fatalf("changed = %v, want %v", changed, tt.wantFlg)
			}
			if got != tt.want {
				t.Fatalf("body = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSlug(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
	}{
		{in: "Simple Title", want: "simple-title"},
		{in: "Pässwörter übertragen", want: "p-ssw-rter-bertragen"},
		{in: "a_b & c!?", want: "a-b-c"},
		{in: "💖", want: ""},
		{in: "  trim  me  ", want: "trim-me"},
		{in: "[[weird]] title", want: "weird-title"},
	}
	for _, tt := range tests {
		if got := Slug(tt.in); got != tt.want {
			t.Fatalf("Slug(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSlugLengthCap(t *testing.T) {
	t.Parallel()
	got := Slug(strings.Repeat("word ", 30))
	if len(got) > 64 {
		t.Fatalf("slug too long: %d", len(got))
	}
}
