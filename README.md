# stoa

A CLI tool that organizes notes, journal entries and documentation as plain
`.md` files, with an SQLite index for tags, `[[wikilinks]]` and search.

Philosophy: the vault is the source of truth; the database is a disposable,
fully rebuildable index (`stoa reindex`). Edit files with nvim, `vim`, whatever
— stoa never locks anything.

## Install

Requires [Go](https://go.dev/) 1.27+ (and [Typst](https://typst.app) only for
`stoa gen`, the CV PDF preview ... TODO check).

```sh
# latest release
just install            # if you have [just](https://github.com/casey/just)
go install github.com/stoicknight/stoa/cmd/stoa@latest

# or a specific version
go install github.com/stoicknight/stoa/cmd/stoa@v0.1.0
```

Put `$GOBIN`/`~/go/bin` on your `PATH` if `stoa` is not found after install.

## Layout

```
~/.stoa/vault/          # the vault (configurable)
├── notes/              # stoa new note
├── journal/            # stoa new journal
├── docs/               # stoa new doc
├── .trash/             # stoa rm target (recoverable; --purge deletes for good)
└── .stoa/index.db      # sqlite index (delete it: reindex rebuilds it)
```

Every file carries a small frontmatter block (id, title, type, tags,
created/updated timestamps). The `id` is a UUIDv7: stable across renames and
edits, sorts chronologically.

## Commands

| command | what it does |
|---|---|
| `stoa init` | create vault, config, empty index |
| `stoa new <note\|journal\|doc> "title" [--tags=a,b]` | create + open in the configured editor |
| `stoa edit <ref>` | open in the configured editor, then reindex |
| `stoa show <ref> [--path]` | print content or file path |
| `stoa list [--type=t] [--tag=t] [--limit=n]` | browse |
| `stoa search <terms>` | substring search (titles, tags, bodies) |
| `stoa link <a> <b> [--kind=ref]` | append a `[[b]]` wikilink to a |
| `stoa backlinks <ref>` | which notes link to ref |
| `stoa rm <ref>... [--yes] [--purge]` | move to `.trash` (or delete) |
| `stoa reindex` | rebuild the index from files |
| `stoa gen <file.md> [--pdf] [-o out] [--theme name\|.typ] [--watch]` | generate a CV PDF from a markdown file (type: cv) |
| `stoa config [key [value]]` | show or set `vault`/`editor`/`debug` |

`<ref>` accepts an id, an id prefix, a title, a file name, or a vault-relative
path (case-insensitive except the id prefix). Ambiguous matches fail with a
numbered candidate list (`<id-prefix>  <title>`); pick one with the id prefix.

## Links

Relations live in two places on purpose:

1. **In the file** as Obsidian-style `[[wikilinks]]` — nothing locked in.
2. **In the DB** `links` table (`source_id, target_id, kind`) — resolved or
   unresolved targets, ready for graph queries, typed links and, later,
   LLM-generated links.

## Config

Precedence: built-in defaults < `~/.config/stoa/stoa.toml` < `STOA_VAULT`
env var. Every command accepts `--config <path>` to select a different
config file.

```toml
vault = "/home/you/.stoa/vault"
editor = "nvim"
debug = false
```

Defaults: `vault` = `~/.stoa/vault`; `editor` = `$EDITOR`, falling back to
`vi` (`stoa init` writes `nvim` into the config file it creates). Set values
with `stoa config key value`.

## Development

Run straight from the repo:

```sh
just check      # gofmt + vet + golangci-lint + tests (-race)
just run list   # go run ./cmd/stoa list (no binary)
just build      # → bin/stoa, then ./bin/stoa <cmd>
just db-gen     # regenerate sqlc code after *.sql edits
```

Go tools (goose, sqlc, golangci-lint) are pinned in `go.mod` under `[tool]`;
no global installs needed.

The current working ledger — repo review findings, prioritized fixes,
roadmap and a feature backlog — lives in [docs/todo.md](docs/todo.md).

Design reference for making stoa a dual-audience (human + agent) CLI — talk
review and portable code examples — lives in
[docs/reference/agent-friendly-cli/](docs/reference/agent-friendly-cli/).

Architecture decisions live in [docs/decisions/](docs/decisions/) — one file
per decision (ADD-NNNN-*.md): status, context, decision, consequences. Agents
must read the relevant ADD before proposing related changes; decisions are
re-opened only by the user's explicit request.

## Roadmap (not in MVP)

- FTS5 `MATCH` relevance search (sqlc cannot parse fts5's hidden column yet;
  the table is already populated)
- LLM adapter for summaries/extraction (behind a plain interface, wired in main)
- TUI, sync, import
