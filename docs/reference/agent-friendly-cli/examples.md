# Code examples from `tebeka/talks` — agent-cmdline

Companion to [review.md](review.md). These snippets are extracted verbatim
(minus demo-only bits) from https://github.com/tebeka/talks/tree/main/agent-cmdline
— the code of Miki Tebeka's talk *"CLI Design for Humans and AI"*. They are a
**starting point to follow and build upon**, not code to copy: the demo is a
`urfave/cli/v3` + Postgres tool, stoa is `spf13/pflag` + SQLite — the *patterns*
port, the plumbing doesn't.

Read the snippets in order; each builds on the previous (that's how the talk is
staged: `0-initial` → `6-bounded`, each step a small diff).

---

## 1. TTY gate — decorate only for eyes, never for pipes

Source: `agent-cmdline/1-tty/main.go` (diff vs `0-initial`: 5 lines)

```go
var isTTY = term.IsTerminal(int(os.Stdout.Fd()))

func highlight(s string) string {
	if !isTTY {
		return s
	}

	return "\033[1;35m" + s + "\033[0m"
}
```

Why: agents receive raw captured stdout. ANSI escape codes are noise that
pollutes context and can corrupt parsing (`| cat -v` shows the garbage).

Port to stoa: stoa already does the hard part (`internal/cli/tty_linux.go`,
TIOCGWINSZ ioctl). Route **all** cosmetic decoration (colors, spinners,
progress) through a single `highlight`-style chokepoint so nothing escapes
into piped output. Rule: decoration checks `isTTY`; *data format* never does.

## 2. Formatter indirection — one flag swaps every output shape

Source: `agent-cmdline/2-json/main.go`

```go
Flags: []cli.Flag{
	&cli.StringFlag{Name: "format", Value: "text", Usage: "output format: text or json"},
},
Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
	switch c.String("format") {
	case "json":
		formatRow = formatRowJSON
		formatTotal = formatTotalJSON
	case "text":
	default:
		return ctx, fmt.Errorf("format must be text or json")
	}
	return ctx, nil
},
```

```go
var formatRow = formatRowText
var formatTotal = formatTotalText

func formatRowJSON(columns []string, values []any) error {
	row := make(map[string]any, len(columns))
	for i, col := range columns {
		row[col] = values[i]
	}
	return json.NewEncoder(os.Stdout).Encode(row)
}
```

Two non-obvious decisions in this step:

- **NDJSON** (one JSON object per line), not a top-level array — the agent can
  stream-parse line by line and stay resilient to truncation.
- **Even the summary line goes JSON.** If you append a plain-text
  `"12 row(s)"` line to a JSON stream, the agent tries to parse it as JSON and
  fails. Metadata becomes a separate object with a `_meta` key:

```go
func formatTotalJSON(count, all int) error {
	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"_meta": map[string]any{"count": count, "total": all, "truncated": count < all},
	})
}
```

Port to stoa: add `--format text|json` to `list`, `search`, `backlinks`, and
`show`; notes emit one JSON object per line, `_meta` last. The
`formatRow/formatTotal/formatError` package vars + `Before`-hook swap is the
whole architecture — no per-command branching.

## 3. Non-interactive mode — the TTY-aware prompt

Source: `agent-cmdline/3-yes/main.go`

```go
if !c.Bool("yes") && !hasLimit(statement) {
	if err := confirm(); err != nil {
		return err
	}
}
```

```go
func confirm() error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("query has no WHERE or LIMIT clause, use -yes to run it")
	}

	fmt.Fprint(os.Stderr, "Query has no WHERE or LIMIT clause. Run it? [y/N] ")
	var answer string
	if _, err := fmt.Fscanln(os.Stdin, &answer); err != nil || (!isYes(answer)) {
		return errors.New("query canceled")
	}
	return nil
}
```

Note the three-way behavior: interactive human gets a prompt on **stderr**;
non-interactive caller gets an *error with the workaround in the message*
("use -yes to run it") instead of hanging — better than both `yes |` piping and
a silent hang. (The demo's `hasLimit` — a naive `WHERE`/`LIMIT` substring
check — is deliberately fake; see `CLAUDE.md` in that repo.)

Port to stoa: stoa has `rm --yes` already. The portable rule: destructive or
irreversible actions require an **explicit** `--yes` — never infer "no TTY ⇒
safe to proceed" (a `tee` in the pipeline would silently authorize it).
`stoa new` needs the inverse: a `--no-edit`/stdin path so agents aren't
wedged inside `$EDITOR`.

## 4. Self-discovery — let the agent orient itself

Source: `agent-cmdline/4-schema/main.go`

```go
}, {
	Name:      "schema",
	Usage:     "list tables, or show the columns of TABLE",
	ArgsUsage: "[TABLE]",
	Action: func(ctx context.Context, c *cli.Command) error {
		db, err := sql.Open("postgres", dsn)
		if err != nil { return err }
		defer db.Close()

		if c.NArg() == 0 {
			return query(ctx, db, tablesSQL)
		}
		return query(ctx, db, columnsSQL, c.Args().First())
	},
}},
```

With the queries embedded in the binary — a single distributable artifact, no
sidecar files:

```go
var (
	//go:embed tables.sql
	tablesSQL string

	//go:embed columns.sql
	columnsSQL string
)
```

The `schema` subcommand answers "what can I ask about?" in one call; a second
call (`schema trips`) answers "what columns exist?". This is why agents stop
popping questions mid-task — the discovery surface is in the tool.

Port to stoa: the analog is a vault-orientation command, not a DB schema —
`stoa stats` (counts per type/tag, date range, newest ids) and/or
`stoa tags`. Cheaper alternative per the talk's Q&A: a good skill file + help
text. stoa's README is already strong; the gap is machine-oriented listings.

## 5. Actionable errors — hint on stderr, JSON or text

Source: `agent-cmdline/5-errors/main.go`

```go
func errorHint(err error) string {
	pqErr, ok := errors.AsType[*pq.Error](err)
	if !ok {
		return ""
	}

	if pqErr.Hint != "" {
		return pqErr.Hint
	}

	switch pqErr.Code {
	case "42P01", "42703": // undefined_table, undefined_column
		return `run "taxi schema" to list tables, "taxi schema TABLE" to list columns`
	}
	return ""
}

func formatErrorJSON(err error) {
	out := map[string]string{"error": err.Error()}
	if hint := errorHint(err); hint != "" {
		out["hint"] = hint
	}
	json.NewEncoder(os.Stderr).Encode(out)
}

func formatErrorText(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	if hint := errorHint(err); hint != "" {
		fmt.Fprintln(os.Stderr, "hint:", hint)
	}
}
```

The pattern has four layers:

1. Prefer a **hint from the underlying system** (Postgres suggests the correct
   column — pass it through).
2. Else, map **known error codes** to a *next action*, not an apology:
   "run `taxi schema` to list tables".
3. Errors are a **third formatter** (`formatError`), JSON on stderr when
   `-format=json` — the agent's stderr dump stays machine-readable.
4. Exit status non-zero, so the agent can key off it before reading stderr.

Port to stoa: stoa's `<ref>` resolution is the analog failure point. Not
found → `run "stoa list" to see available notes`; ambiguous title →
`disambiguate with an id prefix (stoa show <id-prefix>)`. Wrap sqlite errors
(`errors.As`) so constraint failures can carry hints too.

## 6. Bounded output — estimate, cap, and admit truncation

Source: `agent-cmdline/6-bounded/main.go`

Estimate *before* running, refuse with an actionable message:

```go
const maxRows = 1000

if !c.IsSet("limit") {
	n, err := estimateRows(ctx, db, statement)
	if err != nil { return err }
	if n > maxRows {
		return fmt.Errorf(
			"query returns about %d rows, use -limit N to get the first N rows (-limit 0 for all) or update your query", n)
	}
}
```

```go
func estimateRows(ctx context.Context, db *sql.DB, statement string) (int, error) {
	var data []byte
	if err := db.QueryRowContext(ctx, "EXPLAIN (FORMAT JSON) "+statement).Scan(&data); err != nil {
		return 0, err
	}

	var plans []struct {
		Plan struct {
			Rows int `json:"Plan Rows"`
		}
	}
	if err := json.Unmarshal(data, &plans); err != nil {
		return 0, err
	}
	return plans[0].Plan.Rows, nil
}
```

Then count *every* row, print only `limit`, and confess the difference:

```go
count, all := 0, 0
for rows.Next() {
	all++
	if limit > 0 && count == limit {
		continue
	}
	// ... scan, formatRow ...
	count++
}
// ...
return formatTotal(count, all)
```

```go
func formatTotalText(count, all int) error {
	total := fmt.Sprintf("%d row(s)", count)
	if count < all {
		total = fmt.Sprintf("%d of %d row(s), truncated (use -limit 0 for all)", count, all)
	}
	fmt.Println(highlight(total))
	return nil
}
```

Two talk points behind this step: a full dump can blow the agent's context
(~200k tokens, degraded well before), and **agents trust partial data** — they
won't guess the output is incomplete, so the tool must say it (`_meta.truncated`
in JSON mode). Also note the default cap is honored even for humans piping
output; `-limit 0` is the explicit escape hatch.

Port to stoa: verify `list --limit` already reports truncation honestly; make
`search` snippet-only by default (`--full` to opt into bodies) — full note
bodies are stoa's equivalent of the unbounded query.

---

## Anti-patterns the demo starts from (step `0-initial`)

Worth remembering what the sequence fixes — this is the "before" state:

- ANSI colors emitted unconditionally → escape-code garbage when piped.
- Progress/connection messages mixed into stdout → agents try to parse
  "connecting to database…" as data.
- Comma-grouped big numbers (`1,234`) → LLMs misread them.
- Spinners and progress bars → meaningless to a non-TTY consumer.
- Interactive prompts without a non-interactive escape → agent sees a hang.
- Unbounded result dumps → context blowup.

## Where to read more

- Step-by-step diffs: `git clone --depth 1 https://github.com/tebeka/talks`
  then `diff 1-tty/main.go 2-json/main.go` etc. — each diff is slide-sized.
- `agent-cmdline/playbook.md` — talk flow, demo commands, DuckDB agent-mode
  lessons (59% fewer output tokens).
- `agent-cmdline/CLAUDE.md` — the repo's own agent instructions, including the
  "deliberately naive `hasLimit`" disclaimer.
- https://duckdb.org/2026/10/09/agent-mode — production agent-mode design.

For what this means for stoa specifically, see the decisions list in
[review.md](review.md).
