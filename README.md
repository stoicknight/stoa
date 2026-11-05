# stoa

A CLI tool that organizes notes, journal entries and documentation as plain
`.md` files, with an SQLite index for tags, `[[wikilinks]]` and search.

Philosophy: the vault is the source of truth; the database is a disposable,
fully rebuildable index (`stoa reindex`). Edit files with nvim, `vim`, whatever
— stoa never locks anything.

## Install

Requires [Go](https://go.dev/) 1.27 or newer.

```sh
# with just
just install
# or directly — pin a tag for reproducibility (e.g. @v0.1.0)
go install github.com/stoicknight/stoa/cmd/stoa@latest
```

The binary lands in `$GOBIN` (typically `~/go/bin`) — add it to `PATH` if
`stoa` is not found after install.

## Layout

```
~/.stoa/vault/          # the vault (configurable)
├── notes/              # stoa new note
├── journal/            # stoa new journal
├── docs/               # stoa new doc
├── .trash/             # deleted-but-recoverable notes
└── .stoa/index.db      # sqlite index (delete it: reindex rebuilds it)
```

Every file carries a small frontmatter block (id, title, type, tags,
created/updated timestamps). The `id` is a UUIDv7: stable across renames and
edits, sorts chronologically.

## Commands

| command | what it does |
|---|---|
| `stoa init` | create vault, config, empty index |
| `stoa new <note\|journal\|doc> "title" [--tags=a,b]` | create + open `$EDITOR` |
| `stoa edit <ref>` | open in `$EDITOR`, then reindex |
| `stoa show <ref> [--path]` | print content or file path |
| `stoa list [--type=t] [--tag=t] [--limit=n]` | browse |
| `stoa search <terms>` | substring search (titles, tags, bodies) |
| `stoa link <a> <b> [--kind=ref]` | append a `[[b]]` wikilink to a |
| `stoa backlinks <ref>` | which notes link to ref |
| `stoa rm <ref>... [--yes] [--purge]` | move to `.trash` (or delete) |
| `stoa reindex` | rebuild the index from files |
| `stoa gen <file.md> [--pdf] [-o out] [--theme name] [--template path] [--watch]` | generate a CV PDF from a markdown file (type: cv) |
| `stoa config [key [value]]` | show or set `vault`/`editor`/`debug` |

`<ref>` accepts an id, an id prefix, a title, or a file name.

## Links

Relations live in two places on purpose:

1. **In the file** as Obsidian-style `[[wikilinks]]` — nothing locked in.
2. **In the DB** `links` table (`source_id, target_id, kind`) — resolved or
   unresolved targets, ready for graph queries, typed links and, later,
   LLM-generated links.

## Config

Defaults < `~/.config/stoa/stoa.toml` < `STOA_VAULT` env var:

```toml
vault = "/home/you/.stoa/vault"
editor = "nvim"
debug = false
```

## Development

No install step while stoa is MVP under active development. Run it straight
from the repo:

```sh
just check      # gofmt + vet + golangci-lint + tests
just run list   # go run ./cmd/stoa -... (no binary)
just build      # → bin/stoa, then ./bin/stoa <cmd>
just db-gen     # regenerate sqlc code after *.sql edits
```

Installing to `~/.local/bin` is only worth doing once the CLI settles; by
then it will come back as a Justfile recipe.

Go tools (goose, sqlc, golangci-lint) are pinned in `go.mod` under `[tool]`;
no global installs needed.

The current working ledger — repo review findings, prioritized fixes,
roadmap and a feature backlog — lives in [docs/todo.md](docs/todo.md).

## Roadmap (not in MVP)

- FTS5 `MATCH` relevance search (sqlc cannot parse fts5's hidden column yet;
  the table is already populated)
- LLM adapter for summaries/extraction (behind a plain interface, wired in main)
- TUI, sync, import

## CV generation

`stoa gen <file.md>` renders a CV note to PDF through the [Typst](https://typst.app)
CLI (`sudo pacman -S typst` on Arch). It works on any markdown file — no vault
needed. Output defaults to the input's directory; the parsed data schema is
always saved next to the PDF as `<name>.yaml` for preview/debugging.

```
stoa gen cv.md                     # → cv.pdf + cv.yaml (default theme: modern)
stoa gen cv.md --watch             # re-render on file changes (Ctrl-C stops)
stoa gen cv.md -o ~/cv/out.pdf     # custom output path
stoa gen cv.md --theme classic     # accent sans look (themes: modern | classic | minimalist)
stoa gen cv.md --template my.typ   # custom Typst template against the same schema
```

Built-in themes live in `internal/cv/themes/` (`classic` — sans with a
single accent color, `modern` — serif and all-black with underlined
links, `minimalist` — the original plain look). `classic` and `modern`
are faithful reimplementations of rendercv 0.3.0's classic and
engineeringresumes looks against stoa's schema (deliberate deviations:
A4 instead of US Letter, no footer/top note, no connection icons).
`--theme` and `--template` are mutually exclusive.

Reformatting the CV is pure Typst: write a new template against the stable
schema (see `internal/cv/themes/*.typ` for examples) — no stoa changes.

### CV markdown format

The format is intentionally restricted so parsing is predictable; anything
else is an error with a line number. Frontmatter is flat `key: value`
(mirroring stoa's frontmatter style). All contacts except the location
become links (`tel:`, `mailto:`, `https:`).

```markdown
---
id: <uuid>
title: Jane Doe
type: cv
position: Cloud & Automation Engineer
location: London, UK
tel: +1 555 555 5555
email: jane.doe@example.com
linkedin: jane-doe
github: janedoe
website: example.com
---

## Summary

One or more prose lines form a paragraph section.

## Technical Skills

- **Cloud & IaC:** AWS, Terraform, Ansible
- **Programming:** Go, Python

## Professional Experience

### Freelance IT Consultant & Software Engineer

**Self-employed** — Remote
*Aug 2025 – Present*

- Delivered client engagements with **GitOps** CI/CD.

## Projects

### Stoa — CLI Notes Tool *(MVP)*

- Single binary for notes and documentation.

## Certifications

- Red Hat Certified System Administrator (RHCSA)
```

Rules:

- `## Section` — section heading; sections render in file order.
- A section is **entries** (contains `###`), **bullets** (only `- ` lines),
  or **paragraph** (prose). Mixing shapes in one section is an error.
- Inside `### Entry`: the `**bold**` line is the organization (text after
  an em dash is the location or, education-style, a date range); a bare or
  italic line containing a year is the date range; `- ` lines are bullets.
- Inline markup: `**bold**`, `*italic*`, `[text](url)`. Nothing else is
  interpreted; special characters are escaped literally.

## Status

MVP: usable and in daily use, but still under rapid development and
pre-1.0. Not accepting contributions yet — the design phase moves too
fast for anything else. Forks welcome; issues (bugs, questions, feature
requests) are always read.

## License

MIT — see [LICENSE](LICENSE).
