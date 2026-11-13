# stoa — project rules for agents and contributors

Beyond the global principles, this project adds:

## Pre-1.0 freedom (until version 1.0.0+)

Anything may be restructured, including destructively: file layout,
command surface, config schema, flag names, docs. No compatibility
shims, no deprecation notices. After 1.0.0 these freedoms end and
breaking changes follow a changelog/notice policy (to be defined at
1.0.0).

## The database is only an index

The SQLite file is a derived, disposable index over the vault files —
nothing authoritative lives in it. Consequences:

- Pre-1.0, migrations may be reworked, squashed or rewritten freely;
  an index.db at an older recorded version is deleted and rebuilt with
  `stoa reindex` (see docs/decisions/ADD-0003-sqlite-index-store.md).
- Never store data in the DB that cannot be regenerated from the vault.

## Obsolete docs may be removed with approval

If an ADD, spec, reference doc or other doc becomes obsolete, and
removing it changes nothing for current or future decisions, it can be
deleted — **with the user's explicit approval**. Prefer deletion over
leaving stale text that agents may misread. The decision index in
docs/decisions/README.md must stay accurate.

## Documentation conventions

Docs state reality, deterministically and briefly. The docs listed in
README.md (todo ledger, decisions, references) are the map; keep them
in sync in the same commit as code changes they describe.
