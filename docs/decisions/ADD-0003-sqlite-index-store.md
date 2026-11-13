# ADD-0003: SQLite index store enforces foreign keys; migration policy is version-scoped

- **Status:** Decided — adopted
- **Date:** 2026-11-12
- **Implemented in:** commit `81e8fba` (FK pragma, race fix), squash commit (this file's history)

## Context

The `store.Open` DSN set `busy_timeout` and `journal_mode(WAL)` but not
`foreign_keys`. SQLite defaults enforcement off per connection, so every
`REFERENCES ... ON DELETE CASCADE` clause in `00001_init.sql` was dead
text: `index.pruneOrphans` deleted note rows while their `tags` and
`links` rows stayed behind as permanent garbage.

## Decision

1. **Foreign keys are always on.** `_pragma=foreign_keys(1)` is part of
   the DSN in `store.Open` (the modernc driver applies DSN pragmas to
   every pooled connection). Any new table must declare FKs that work,
   and tests may rely on `ON DELETE CASCADE`.
2. **Migration policy is version-scoped.** stoa's index is disposable by
   design (the vault files are the source of truth; `stoa reindex`
   rebuilds the DB from scratch), which shapes the rule:
   - **Pre-1.0:** the migration baseline may move — migrations may be
     squashed or edited while the schema evolves and no upgrade paths
     exist to protect. An index.db at an older recorded version is
     simply deleted and rebuilt (`stoa reindex`).
   - **From v1.0 / first real vault usage onward:** migrations are
     append-only and never edited. goose records only the version
     number, so editing an applied migration would silently skip it on
     existing databases, leaving two different schemas claiming the
     same version with nothing to tell them apart.
3. **Data repairs post-1.0 live in new migrations** (`000NN_*.sql`,
   goose Up/Down; Down may be a documented no-op for irreversible
   cleanups).

## Consequences

- Pre-1.0 schema changes do not need upgrade paths; if an existing
  index.db records a version higher than the repo's migrations, delete
  it and `stoa reindex` (documented behavior, not an error path to
  build).
- Post-1.0: every schema change or one-time data repair = new migration
  file.
- goose's process-global setup state is mutexed in `store.Open`;
  concurrent `Open` calls (parallel tests, future side-car processes)
  are safe.
- Opening the index validates the pragma: `PRAGMA foreign_keys` must
  return 1 (pinned by `TestForeignKeysEnforced`).
