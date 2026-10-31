// Command stoa organizes notes, journal entries and documentation as plain
// .md files with an SQLite index for tags, links and full-text search.
package main

import (
	"fmt"
	"os"

	"github.com/stoicknight/stoa/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "stoa:", err)
		os.Exit(1)
	}
}
