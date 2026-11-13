# Architecture Decision Records

One ADD per file: `ADD-NNNN-short-name.md`. Format: bold-key header
(Status, Date), then Context, Decision, Consequences — terse, no MADR
frontmatter. Agents must read the relevant ADD before proposing related
changes; decisions are re-opened only by the user's explicit request.

| ADD | decision | status | evidence/details |
|---|---|---|---|
| [ADD-0001](ADD-0001-duckdb-not-adopted.md) | DuckDB is not adopted as stoa's storage engine — SQLite (modernc) only | Decided — rejected (not adopted), 2026-11 | [duckdb-evaluation.md](../duckdb-evaluation.md) |
| [ADD-0002](ADD-0002-cv-theme-template.md) | CV rendering: one `--theme` flag (built-in name or `.typ` file) with an explicit template contract; default theme `classic` | Decided — adopted, 2026-11-12 | commit `2471421`, [reference/cv-template.md](../reference/cv-template.md) |
| [ADD-0003](ADD-0003-sqlite-index-store.md) | SQLite index store: foreign keys always on; migration policy version-scoped (pre-1.0 baseline may move, post-1.0 append-only) | Decided — adopted, 2026-11-12 | commits `81e8fba` + squash, `internal/store/` |
