# Review: "CLI Design for Humans and AI" — key takeaways and decisions worth exploring in stoa

Companion: [examples.md](examples.md) extracts the talk's code, step by step,
as patterns to follow and build upon in stoa.

Sources reviewed:

- **Talk**: Miki Tebeka (O'Reilly), *CLI Design for Humans and AI: Building Dual-Audience Tools in Go* — YouTube live `NmxqSdN0LCk`
- **Code**: https://github.com/tebeka/talks/tree/main/agent-cmdline (7 progressive steps, one `main.go` each)
- **Notes**: repo's `playbook.md` (talk flow + DuckDB agent-mode lessons), `CLAUDE.md`, `devcon-ro.md`

---

## 1. Core thesis

A CLI is the cheapest way to give a coding agent a tool: no runtime needed (Go static binary), works for humans **and** agents from the same code. Agents differ from humans in three ways that break naive CLIs:

1. **Bounded context** (~200k tokens, performance degrades before the hard limit) — a giant `SELECT *` dump blows the agent's context; humans just Ctrl-C.
2. **Structured output preference** — humans gloss over banner text, spinners, comma-grouped numbers; agents can't reliably separate data from decoration.
3. **Non-interactive execution** — agents run commands as subprocesses without a TTY. An interactive "Are you sure? [y/N]" prompt makes the agent think the command is *hung*.

### Go as the language

- Static single binary, no runtime install, trivial cross-compilation (gorp: mac/windows/linux, arm/x64) via CI + goreleaser.
- Agent-friendly to *modify*: simple, typed, fast compile → fast feedback loop when an agent edits the tool itself.
- Dependencies are supply-chain/security risk; stick to stdlib unless the CLI is complex (then cobra or urfave/cli are fine — the demo uses `urfave/cli/v3`, `lib/pq`, `golang.org/x/term`).

---

## 2. The demo's progression (`taxi query FILE` against 4M-row NYC taxi DB)

Each step adds a small, reviewable diff. Order matters — it's a checklist for any CLI:

| Step | Change | Technique |
|---|---|---|
| 0-initial | bare tool | colors, banner, spinner always on → escape-code garbage when piped |
| 1-tty | TTY check | `term.IsTerminal(int(os.Stdout.Fd()))`; color/highlight only on a real terminal (~5 lines of diff) |
| 2-json | `-format text\|json` | formatter function variables; NDJSON (one JSON object per line); summary line becomes `_meta` JSON too — *never* mix plain text into a JSON stream |
| 3-yes | `-yes` flag | skip the confirmation prompt explicitly; mirrors `apt -y` / `brew -y` non-interactive convention |
| 4-schema | `schema [TABLE]` subcommand | self-discovery: list tables, or columns+types of a table; SQL shipped via `go:embed` (single binary, no sidecar files) |
| 5-errors | actionable errors | `errors.AsType[*pq.Error]` to extract DB *hint*s ("perhaps you meant column `fare_amount`"), else fall back to a next-step hint (`run "taxi schema" to list tables`); errors formatted as compact JSON **on stderr** |
| 6-bounded | `-limit N` | `EXPLAIN (FORMAT JSON)` to estimate rows *before* running; refuse (with an error) if estimate > default cap (1000); count rows and report `count / total, truncated` so the agent knows output is partial; `-limit 0` = everything |

Implementation details worth copying:

- **Formatter indirection**: `formatRow`, `formatTotal`, `formatError` are package vars swapped in a `Before` hook by `-format`. Every new output type needs both text and JSON formatters.
- **Progress/diagnostics → stderr, data → stdout**. An agent redirects stderr to a file (`2>/tmp/err.txt`) and only reads it on non-zero exit.
- **Exit status**: non-zero on failure — agents key off it before reading stderr at all.
- **Truncation honesty**: report "4 of 55 row(s), truncated (use -limit 0 for all)" and in JSON `{"_meta":{"count":4,"total":55,"truncated":true}}`. Agents are *too trusting* of partial data — they won't guess it's incomplete, so the tool must say so.
- **The demo's `hasLimit` (naive `WHERE`/`LIMIT` substring check) is deliberately fake** — noted in `CLAUDE.md` as not a real safety check. Real safety comes from the next point.

### Security: least privilege

Instead of writing "never DELETE" into `AGENTS.md`/`CLAUDE.md` and hoping agents obey (they usually do, sometimes don't — Toyota deleted a prod table anyway), the tool connects with a **read-only DB user**. "The DB enforces permissions, the CLI can't." This is also good hygiene for humans.

### DuckDB agent mode (referenced: duckdb.org blog "agent mode")

> Note: DuckDB itself was evaluated and **rejected** as stoa's engine —
> see [ADD-0001](../../decisions/ADD-0001-duckdb-not-adopted.md) and the
> [evaluation doc](../../duckdb-evaluation.md). Its agent-mode ideas are
> portable and listed as such below.

- Agent detection = known agent env vars + stdout not a terminal.
- Explicit output format chosen by the user **overrides** auto-detection.
- Compact markdown tables for results; column types shown with names.
- Explicit truncation reporting; ability to stop large results early.
- Compact JSON errors on stderr; short one-time startup notice, full instructions in `--help`.
- Measured **59% fewer output tokens**, all answers still correct (but no total-cost/runtime savings claim).

---

## 3. Q&A takeaways (from the video)

1. **TTY vs explicit flags**: TTY detection is fine for *cosmetic* adaptation (colors, spinners). **Destructive** behavior must require an explicit `-yes` — never infer "non-interactive ⇒ safe to proceed" (e.g. piping through `tee` would silently authorize destruction). Better to hang than to destroy without explicit authorization.
2. **CLI vs MCP**: CLIs are easier to write, and both humans and agents can use them. MCPs earn their cost for credential isolation (run as a service holding the credentials; clients never see them). Start with a CLI; move to MCP if credentials/secrets matter.
3. **Skills per CLI**: yes — let the agent itself generate a skill file from `--help` output of the tool and its subcommands. The skill header tells the agent *when* to reach for it. Update the skill when the tool changes by re-running `--help`.
4. **Output format**: JSON is the lingua franca agents already know (YAML/TOML are more human-readable but token-similar; TOON ≈ better CSV, for tabular things; he "hates CSV with passion").
5. **Agents do, humans judge**: agents execute without questioning whether the task is worth doing — push them (or your skills) to *gather evidence first* (e.g. "what % of users is affected?" → 0.001% → skip).

---

## 4. Decisions worth exploring in stoa

stoa already matches the talk better than average (TTY check in `internal/cli/tty_linux.go`, `--yes` on `rm`, `--limit` on `list`, stderr prefix `stoa:`). The items below are the deltas, roughly ordered by value/effort.

### A. Structured output (`-format json`) — highest value, moderate effort

- `stoa list`, `stoa search`, `stoa backlinks`, `stoa show` currently print human text only. An agent querying a vault has to parse prose.
- Adopt the demo's pattern: `formatRow`/`formatTotal`/`formatError` vars, NDJSON (one note per line), and a `_meta` line with `{count, total, truncated}` at the end.
- Decide deliberately whether to expose `--format` or auto-switch on `!isTTY`. The talk's answer: cosmetics (color) auto-switch on TTY; *data format* should be explicit (`-format json`), so piping `stoa list | head` still yields predictable text.
- Watch out: the sqlite/sqlc layer is already the structured side; this is purely an output-layer concern in `internal/cli/*`.

### B. Actionable errors with hints — small effort, high agent payoff

- stoa's `<ref>` resolution (id/prefix/title/filename) is the natural failure point. On ambiguous or not-found refs, mirror the demo's pattern:
  - not found → hint: `run "stoa list" to see available notes`
  - ambiguous title → hint: `disambiguate with an id prefix (stoa show <id-prefix>)`
- Ensure every error goes to **stderr** with a consistent machine-findable prefix (currently `stoa: <err>` — fine), and non-zero exit status everywhere.
- Consider `errors.AsType`-style wrapping around sqlc/sqlite errors so constraint failures can carry a hint too.

### C. Bounded + honest output — verify, then extend

- `stoa list --limit` exists: confirm it reports truncation ("12 of 340 notes, truncated — use --limit 0 for all" or `_meta.truncated` in JSON mode). Unmarked truncation is the "agents trust partial data" trap.
- `stoa search` is the likely context-blower (full note bodies). Consider `search` returning only match snippets/paths by default, with full bodies opt-in (`--full` or in `show`).
- `stoa reindex` on a big vault = progress report. Follow the demo rule: progress/spinners only on TTY, silence (or a single stderr line) when piped.

### D. Self-discovery subcommand — optional

- The talk's `schema` subcommand is a DB-specific idea; stoa's analog would be `stoa list --tags` or a `stoa schema`-like `stoa tags` / `stoa stats` (counts per type/tag) so an agent can orient itself in the vault without opening files.
- Cheaper alternative (matches talk's Q&A): rely on the skill header + `--help`; stoa's README/`docs/` already document commands well.

### E. Safety posture beyond `--yes` — align with the "least privilege" lesson

- `rm --purge` is stoa's destructive edge. The talk's answer for destructive ops: keep the explicit `--yes` (never infer from TTY absence), and consider a "soft by default" design — which stoa already has (`.trash/`). Document that `.trash` is the safety net *and* that `--purge --yes` is the only unrecoverable path.
- The SQLite index is already disposable/rebuildable (`stoa reindex`) — the equivalent of the read-only-user principle applied to the *file* side: the vault is the source of truth, tools never mutate beyond their command's contract. Worth keeping as an explicit design invariant when the LLM-adapter lands in the roadmap.

### F. Small concrete wins

- `internal/cli/tty_other.go` — confirm the non-Linux path is `term.IsTerminal` (stdlib `x/term`), not just build-tag drift.
- `stoa gen --watch` error output currently uses `stoa:` stderr prefix — keep diagnostics there, but make sure the *first* error also sets a non-zero exit at the end of the run.
- Add `go:embed`-style bundling thinking for any future asset (themes already live in `internal/cv/themes/` — good, single binary holds).
- Exit codes: consider distinguishing "usage error" (2) from "operation failed" (1) so agents can retry differently.

---

## 5. Source pointers

- Demo repo: `github.com/tebeka/talks` → `agent-cmdline/0-initial`…`6-bounded` (read them as diffs: `diff 1-tty/main.go 2-json/main.go`)
- `playbook.md` in that repo = talk flow + command-by-command demo script
- DuckDB agent-mode post: https://duckdb.org/2026/10/09/agent-mode
- Slide deck: linked from the repo (`slides.pdf`)
