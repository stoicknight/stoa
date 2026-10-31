package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/stoicknight/stoa/internal/domain"
	"github.com/stoicknight/stoa/internal/store/dbx"
)

// runList lists notes, optionally filtered by type or tag.
func runList(args []string) error {
	fs := fsBase("list")
	typ := fs.String("type", "", "note|journal|doc")
	tag := fs.String("tag", "", "filter by tag")
	limit := fs.Int("limit", 100, "max results")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *typ != "" {
		if _, err := domain.ParseNoteType(*typ); err != nil {
			return err
		}
	}
	cfg := loadConfig(fs.Lookup("config").Value.String())
	ctx := context.Background()
	st, _, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	var (
		rows []dbx.Note
		verr error
	)
	switch {
	case *tag != "":
		rows, verr = st.Q.ListNotesByTag(ctx, dbx.ListNotesByTagParams{
			Tag:   strings.TrimPrefix(*tag, "#"),
			Limit: int64(*limit),
		})
	case *typ != "":
		rows, verr = st.Q.ListNotesByType(ctx, dbx.ListNotesByTypeParams{
			Type:  *typ,
			Limit: int64(*limit),
		})
	default:
		rows, verr = st.Q.ListAllNotes(ctx, int64(*limit))
	}
	if verr != nil {
		return failf("list: %v", verr)
	}
	printNoteRows(rows, "no notes")
	return nil
}

// printNoteRows prints listing rows; shared by list and search. Empty input
// prints the given message.
func printNoteRows(rows []dbx.Note, emptyMsg string) {
	if len(rows) == 0 {
		fmt.Println(emptyMsg)
		return
	}
	for _, n := range rows {
		fmt.Printf("%s  %-7s %s  %s\n", fuzzyID(n.ID, 8), n.Type, shortDate(n.Created), n.Title)
	}
}

// shortDate trims an RFC3339 timestamp to its date part for listings.
func shortDate(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}
