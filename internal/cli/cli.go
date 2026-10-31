// Package cli wires subcommands; each command lives in its own file.
package cli

import "fmt"

// commands is the single source of truth for dispatch and usage output.
var commands = []command{
	{name: "init", summary: "create vault, config and empty index", run: runInit},
	{name: "new", args: `<note|journal|doc> "title" [--tags=a,b]`, summary: "create a note and open $EDITOR", run: runNew},
	{name: "edit", args: "<ref>", summary: "open a note in $EDITOR", run: runEdit},
	{name: "show", args: "<ref> [--path]", summary: "print a note (or its file path)", run: runShow},
	{name: "list", args: "[--type=t] [--tag=t] [--limit=n]", summary: "list notes", run: runList},
	{name: "search", args: "<terms>", summary: "substring search over titles, tags and bodies", run: runSearch},
	{name: "link", args: "<a> <b> [--kind=ref]", summary: "append a [[wikilink]] to note a", run: runLink},
	{name: "backlinks", args: "<ref>", summary: "notes linking to ref", run: runBacklinks},
	{name: "rm", args: "<ref>... [--yes] [--purge]", summary: "move to .trash (or delete with --purge)", run: runRemove},
	{name: "reindex", summary: "rebuild the index from vault files", run: runReindex},
	{name: "gen", args: "<file.md> [--pdf] [-o out] [--theme name] [--template path] [--watch]", summary: "generate a CV PDF from a markdown file (type: cv)", run: runGen},
	{name: "config", args: "[key [value]]", summary: "show or set vault, editor, debug", run: runConfig},
}

// Run parses global dispatch and executes one command.
func Run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		printUsage()
		return nil
	}
	for _, c := range commands {
		if c.name == args[0] {
			return c.run(args[1:])
		}
	}
	printUsage()
	return fmt.Errorf("unknown command %q", args[0])
}

// printUsage is generated from the command registry, so it cannot drift.
// col is the width of the "  <command> <args>" column in usage output.
func printUsage() {
	const col = 32
	fmt.Print(`stoa — markdown vault with an SQLite index

usage: stoa <command> [args] [flags]

commands:
`)
	for _, c := range commands {
		spec := c.name
		if c.args != "" {
			spec += " " + c.args
		}
		if 2+len(spec) > col {
			fmt.Println("  " + spec) // too long: wrap summary to next line
			fmt.Printf("%-*s%s\n", col, "", c.summary)
			continue
		}
		fmt.Printf("  %-*s%s\n", col-2, spec, c.summary)
	}
	fmt.Print(`
env/config: STOA_VAULT overrides vault; key = value in ~/.config/stoa/stoa.toml
`)
}
