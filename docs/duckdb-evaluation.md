# DuckDB evaluation for stoa — cross-checked against official sources + local measurements

> **Decision:** rejected — see [ADD-0001](decisions/ADD-0001-duckdb-not-adopted.md).
> This doc is the supporting evidence; it is retained so the decision can be
> re-verified without re-measuring.

Follow-up to the agent-friendly CLI review (now at
[reference/agent-friendly-cli/review.md](reference/agent-friendly-cli/review.md)).
Question: is DuckDB worth adapting into the stoa CLI? Every load-bearing claim from the LLM report was
cross-checked against official DuckDB docs/blog/GitHub, and the cost numbers were measured
locally rather than trusted.

## 1. Cross-check of the LLM report

| Report claim | Verdict | Source |
|---|---|---|
| DuckDB v2.0.0 LTS "Cinnamon Teal" with agent mode, VARIANT, Quack server, async I/O, triggers | ✅ Accurate | duckdb.org/2026/10/21/announcing-duckdb-200 (2026-10-21); GitHub release v2.0.0. 2.1 "Islandica" expected 2027 H1 |
| Go driver requires CGO, no pure-Go driver | ✅ Accurate | Official docs: "the client uses cgo to embed DuckDB" (duckdb.org/docs/2.0/clients/go/troubleshoot); pkg.go.dev duckdb-go/v2 v2.20000.x ↔ DuckDB v2.0.0 |
| Read-write = exclusive file lock; concurrent readers need `access_mode='READ_ONLY'` | ✅ Accurate | duckdb.org/docs/2.0/connect/concurrency |
| Agent mode: 59% fewer output tokens, all answers correct | ✅ Accurate, with nuance | Full post fetched: 123.6k → 50.8k tokens over 132 TPC-H runs, all correct. **But**: savings ≈ 0.5% of total input cost (system prompt dominates), slightly *more* turns (237 vs 224), no total-cost/runtime win. The report/playbook hedges this correctly |
| ~40–70+ MB binary cost | ✅ Roughly right, possibly understated | DuckDB 2.0.0 CLI binary itself measures **74 MB** on disk |
| Hybrid `ATTACH 'x.db' (TYPE sqlite)` | ✅ Verified live | Ran it locally: attached an SQLite file from the DuckDB 2.0 CLI, `SELECT count(*), avg(a)` worked (sqlite extension autoloads) |

## 2. Measured locally (this machine, 4 cores, 8 GB RAM)

| Metric | DuckDB 2.0.0 CLI | stoa (modernc SQLite) |
|---|---|---|
| Binary size | **74 MB** | **12 MB** (SQLite + sqlc + config included) |
| Startup (`SELECT 1`) | **~57 ms**, **~50 MB peak RSS** | **~7 ms** warm, **~12 MB** peak RSS |
| Default memory limit | 6.1 GiB (80% of system RAM) | n/a (SQLite allocates on demand, MBs) |
| 100k-row storage (same data) | **780 KB** (columnar, ~2.6× smaller) | **2,052 KB** |
| `count/avg` over 100k rows | 71 ms **including** its own startup | **12.4 ms** warm process |
| Multi-process concurrency | One read-write process holds the file exclusively | WAL: many readers + one writer |

Key insight from the speed row: for a CLI that spawns one process per invocation (exactly how
agents call tools), DuckDB's **~57 ms startup + 50 MB baseline dominates any OLAP speedup**
until datasets are in the millions of rows. On a 100k-row analytical query, warm SQLite wins.

## 3. Verdict for stoa: not now

stoa's workload is the opposite of DuckDB's sweet spot:

- **Write pattern**: `stoa new`/`edit`/`link`/`rm` → frequent small writes + incremental index
  updates. DuckDB optimizes bulk loads (Appender/COPY); single-row mutations are its weakness.
- **Concurrency**: editors, `gen --watch`, and multiple agents may hit the vault/index
  simultaneously → SQLite WAL fits; DuckDB's single-writer exclusive lock would be a
  regression (or force read_only everywhere).
- **Distribution**: stoa is pure Go (`modernc.org/sqlite`, CGO-free) → trivial cross-compilation
  and a 12 MB binary. duckdb-go drags in CGO + a C/C++ toolchain per target + ~60 MB.
- **Query shape**: point lookups by id/ref, tag joins, FTS5. No aggregations over millions of
  rows — the notes corpus is thousands, not millions.

Adopting DuckDB as the index engine would pay the CGO/size/startup/locking tax and get nothing
back at stoa's data scale.

## 4. What *is* worth taking from DuckDB (zero-cost, engine-agnostic)

The agent-mode ideas are portable to stoa's SQLite-based CLI, and stoa's FTS5/`_meta` plans
already cover most of them. From the official post, the extras worth considering:

1. **Result hash in the `_meta` footer** — "has the result changed since my last call?" is a
   cheap win for agents comparing two runs without printing data twice.
2. **Cost estimate before the work** — `stoa reindex` on a huge vault could print
   `reindex: ~4,200 files, est. 8s` to stderr before starting; long `stoa gen --watch` renders
   likewise.
3. **Early-stop with a lower bound** — DuckDB stops runaway queries after the cap and reports
   "at least N rows". `stoa search` (bodies) could adopt the same "count only after cap" idea.
4. **Explicit-format-wins rule** — `-format json` beats agent auto-detection (DuckDB treats a
   format flag as deliberate and stays out of the way). Same rule already proposed for stoa.
5. **One-line startup notice on stderr** explaining why agent mode is on + how to disable —
   DuckDB found 3 lines (634 B) was *worse* than 1 line; models keep context between calls.

## 5. Down the line — where DuckDB genuinely fits stoa

Not as the index engine, but as an **optional companion**, in ascending order of commitment:

1. **Now-ish: `stoa export` to Parquet/NDJSON** — stoa stays pure Go (writes Parquet via a
   small pure-Go lib or just NDJSON/CSV); the user/agents query it with the *standalone*
   `duckdb` CLI (which has its own agent mode). Zero stoa dependencies added; "my vault as an
   analytics dataset" for free. This mirrors the talk's DuckDB-seed pattern (DuckDB loading
   data into Postgres).
2. **Later: read-only analytics sidecar** — `duckdb -c "ATTACH '~/.stoa/vault/.stoa/index.db'
   AS idx (TYPE sqlite)"` for occasional heavy reporting over the index. Verified working; no
   schema migration, no stoa code changes. Only worth it if a reporting need actually appears.
3. **If RAG ever happens** (see the threshold analysis in agent-friendly-cli-review.md §4):
   DuckDB 2.0's vector-similarity features (`APPROX NEAREST`) could serve embeddings — but only
   inside the *companion* role (a `stoa export`-based pipeline), never as the hot path, or the
   CGO tax infects the main binary.

## 6. Decision

Summarized in [ADD-0001](decisions/ADD-0001-duckdb-not-adopted.md). TL;DR: keep SQLite;
DuckDB only as a future companion (`stoa export` → Parquet + standalone `duckdb` CLI).
Re-opening requires the user's explicit decision.

## 7. Summary table

| Question | Answer |
|---|---|
| Adapt DuckDB into the CLI now? | **No.** Wrong workload (OLTP-ish), wrong concurrency model, CGO tax, 6× binary size, 8× startup latency for zero benefit at stoa's scale |
| Real cost if we did | ~60+ MB binary, CGO toolchain per OS/arch in CI, ~50 MB RAM + ~57 ms per process spawn, single-writer file lock |
| Storage | DuckDB columnar is ~2.6× smaller than SQLite at 100k rows — the only metric where it wins, irrelevant at stoa's scale |
| Speed | Only wins at multi-million-row scans/aggregations; loses outright in per-invocation CLI usage below that |
| Real agentic benefit of DuckDB itself? | The *agent mode* is the benefit, and 100% of it is portable to stoa's SQLite CLI without adopting DuckDB |
| Down the line? | Yes: `stoa export` → Parquet, queried by the standalone `duckdb` CLI (companion role); optional `ATTACH (TYPE sqlite)` analytics; vector features only inside that companion, if RAG ever clears its threshold |
| Status | **Rejected** as engine — [ADD-0001](decisions/ADD-0001-duckdb-not-adopted.md) |
