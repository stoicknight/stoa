# ADD-0001: DuckDB is not adopted as stoa's storage engine

- **Status:** Decided — rejected (not adopted)
- **Date:** 2026-11
- **Evidence:** [docs/duckdb-evaluation.md](../duckdb-evaluation.md) (measured locally, cross-checked against official DuckDB sources)

## Context

stoa needs an embedded index for search, tags and backlinks. DuckDB 2.0
(embedded OLAP engine, agent-mode CLI) was evaluated as a candidate.

## Decision

Keep **SQLite** (pure-Go `modernc.org/sqlite`) as the only index engine.

Reasons, in priority order:

1. stoa's workload is OLTP (small writes, point lookups, ref resolution) —
   DuckDB is columnar OLAP; it wins only on multi-million-row scans.
   Measured: SQLite aggregates 100k rows in 12 ms warm; DuckDB pays ~57 ms
   startup + ~50 MB RSS per process spawn before query execution begins.
2. DuckDB's single-writer exclusive file lock conflicts with stoa's
   multi-process reality (editors, concurrent agents, `gen --watch`).
3. CGO dependency kills CGO-free cross-compilation; binary grows
   ~12 MB → ~74 MB.
4. Every agent-friendly idea in DuckDB's agent mode is portable to a
   SQLite-based CLI at zero cost (see
   [docs/reference/agent-friendly-cli/review.md](../reference/agent-friendly-cli/review.md)).

## Consequences

- Do not add `github.com/duckdb/duckdb-go` or any CGO dependency to stoa.
- If analytics needs arise later, use the **companion** pattern (see
  evidence doc §5): `stoa export` → Parquet/NDJSON queried by the
  standalone `duckdb` CLI, or `ATTACH ... (TYPE sqlite)` read-only.
- Re-opening this decision requires: user's explicit request, plus new
  evidence that stoa's data scale or query shape has outgrown SQLite.
  Agents must not adopt DuckDB on their own initiative.
