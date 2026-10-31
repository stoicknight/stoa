package cli

import (
	"context"
	"strings"

	"github.com/stoicknight/stoa/internal/store/dbx"
)

// runSearch runs an FTS query across titles, tags and bodies.
func runSearch(args []string) error {
	fs := fsBase("search")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return failf("usage: stoa search <terms>")
	}
	query := strings.Join(fs.Args(), " ")
	cfg := loadConfig(fs.Lookup("config").Value.String())
	ctx := context.Background()
	st, _, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	ids, err := st.Q.SearchNoteIDs(ctx, dbx.SearchNoteIDsParams{
		Pattern: "%" + sanitizeFTS(query) + "%",
		Max:     50,
	})
	if err != nil {
		return failf("search %q: %v", query, err)
	}
	rows, err := st.Q.GetNotesByIDs(ctx, ids)
	if err != nil {
		return failf("search %q: %v", query, err)
	}
	printNoteRows(rows, "no results")
	return nil
}

// sanitizeFTS turns free text into a safe default-conjunction FTS5 query.
func sanitizeFTS(q string) string {
	replacer := strings.NewReplacer("\"", " ", "'", " ", "(", " ", ")", " ", "*", " ", ",", " ")
	fields := strings.Fields(replacer.Replace(q))
	return strings.Join(fields, " ")
}
