package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stoicknight/stoa/internal/store"
)

const maxCandidates = 10

// noteRef is a DB note row slimmed down for resolution output.
type noteRef struct {
	ID    string
	Title string
}

// resolveQuery finds one note by exact id, exact path or fuzzy title/name.
func resolveQuery(ctx context.Context, st *store.Store, q string) (string, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return "", failf("empty note reference")
	}
	if note, err := st.Q.GetNoteByID(ctx, q); err == nil {
		return note.ID, nil
	}
	cands := candidates(ctx, st, q)
	switch len(cands) {
	case 1:
		return cands[0].ID, nil
	case 0:
		return "", failf("no note matches %q", q)
	default:
		var b strings.Builder
		fmt.Fprintf(&b, "ambiguous reference %q, candidates:\n", q)
		for _, c := range cands {
			fmt.Fprintf(&b, "  %s  %s\n", fuzzyID(c.ID, 8), c.Title)
		}
		return "", fmt.Errorf("%s", b.String())
	}
}

// candidates returns notes matching id-prefix, exact title, file base name
// or path fragment; on ties the note created most recently wins.
func candidates(ctx context.Context, st *store.Store, q string) []noteRef {
	rows, err := st.Q.ListAllNotes(ctx, int64(maxCandidates))
	if err != nil {
		return nil
	}
	var out []noteRef
	lower := strings.ToLower(strings.TrimSuffix(q, ".md"))
	for _, n := range rows {
		base := strings.TrimSuffix(filepath.Base(n.Path), ".md")
		switch {
		case strings.HasPrefix(n.ID, q),
			strings.EqualFold(n.Title, q),
			strings.EqualFold(base, lower),
			strings.EqualFold(n.Path, q):
			out = append(out, noteRef{ID: n.ID, Title: n.Title})
		}
	}
	return out
}

// fuzzyID cuts a long id down for display.
func fuzzyID(id string, n int) string {
	if len(id) <= n {
		return id
	}
	return id[:n] + "…"
}

// promptContinue reads stdin for a y/N confirmation.
func promptContinue(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return false
	}
	v := strings.TrimSpace(strings.ToLower(sc.Text()))
	return v == "y" || v == "yes"
}
