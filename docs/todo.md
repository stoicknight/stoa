# stoa — repo review todo

Full-repo review, package by package and doc by doc. `just check` passes
(gofmt, vet, golangci-lint 0 issues, tests green) — findings below are
design, edge-case and completeness issues, not gate failures.

Findings are prioritized: **P0** = correctness bug / data-risk, **P1** =
robustness, missing tests, sharp edges, **P2** = simplification /
deduplication / idiom, **P3** = docs & polish. After the findings there is
a suggested roadmap and a feature-idea backlog.

Active findings keep their original numbers even as neighbors are
completed, so references from code, specs and commits stay stable.

---

## Completed

Work from this ledger that is done. Details live in the commit message
and the referenced code/docs; dates are when the fix landed.

| date | finding | commit | details |
|---|---|---|---|
| 2026-11-12 | #5 `gen --theme`/`--template` conflict — fixed by redesign: one `--theme` flag (built-in name or `.typ` file), explicit template contract, default theme `classic` | `2471421` | [ADD-0002](decisions/ADD-0002-cv-theme-template.md), [docs/reference/cv-template.md](reference/cv-template.md) |
| 2026-11-12 | #1 foreign keys were off → orphaned rows poisoned the index (no sweep migration needed: no pre-fix vaults exist, see ADD-0003) | `81e8fba` | `internal/store/store_test.go`, `internal/index/index_test.go`, [ADD-0003](decisions/ADD-0003-sqlite-index-store.md) |
| 2026-11-12 | #23 goose global-state race — migration block mutexed; `just check` now runs `-race` | `81e8fba`, `8404e7c` | `internal/store/store.go`, `Justfile` |
| 2026-11-12 | #25 (partial) `$EDITOR` wording (`new`/`edit`), `stoa.example.toml` header truth | `ccb8ac8` | `README.md`, `internal/cli/cli.go`, `stoa.example.toml` |
| 2026-11-12 | #25 (partial) docs drift pass: config precedence + defaults, `<ref>` resolution semantics, `.trash` wording, Development section | this pass | `README.md` |
| 2026-11-12 | #26 ADD convention resolved — `docs/decisions/README.md` index (option a) | this pass | [docs/decisions/README.md](decisions/README.md) |
| 2026-11-12 | roadmap item 18 (theme/template doc) — superseded by the template contract doc | `2471421` | [docs/reference/cv-template.md](reference/cv-template.md) |

---

## P0 — correctness bugs (fix first)

### 2. `resolveQuery` can only ever see the 10 newest notes
- **Where:** `internal/cli/resolve.go:45` (`candidates`)
- `candidates` calls `st.Q.ListAllNotes(ctx, int64(maxCandidates))`,
  i.e. it loads the **10 most recently created notes** and then
  string-matches them in Go. Any note older than the 10 newest is
  unreachable by id prefix, title, or filename — via *every* command
  (`edit`, `show`, `link`, `backlinks`, `rm`).
- **Fix:** push matching into SQL: a `CandidateNotes` query doing
  `WHERE id LIKE @prefix || '%' OR lower(title) = lower(@title)
  OR lower(replace(path,'.md','')) = lower(@name) LIMIT 11`
  (extra row to keep the "ambiguous" determination exact when hits
  exceed `maxCandidates`). Keep the Go-side ambiguity formatting.

### 3. `stoa link` silently destroys unknown frontmatter keys
- **Where:** `internal/cli/link.go:52` + `internal/frontmatter.Parse`
- `runLink` reads a file, appends the wikilink, then writes
  `frontmatter.Render(meta, updatedBody)` back over the full file.
  `Meta` only knows `id/title/type/tags/created/updated`; any other key
  the user added (`status:`, `project:`, …) is **dropped from the file
  permanently**. `frontmatter.Parse`'s comment promises "swap the parser
  without touching callers", but the render side has no round-trip
  guarantee for unknown keys.
- **Fix options:** (a) `Render` serializes from a parsed map with unknown
  keys preserved in order, or (b) `runLink` does a targeted body
  transformation and keeps the original frontmatter block verbatim
  (cheapest, and in the spirit of "vault is source of truth").
- Include a regression test: render→parse→render with an extra key
  survives.

### 4. A CV note inside the vault bricks `reindex` (and the tags CHECK)
- **Where:** `internal/index/collect` + `internal/domain.ParseNoteType` +
  migration 1's `CHECK (type IN ('note','journal','doc'))`
- `stoa gen` explicitly works on "any markdown file", and
  `examples/cv-example.md` carries `type: cv`. `stoa new` cannot create
  cv notes, but users *will* drop CV notes into the vault (it's a notes
  tool). `collect` then fails the whole walk with
  `unknown note type "cv"`, so tags/search/list go dead until the file is
  removed by hand.
- **Fix:** decide the policy explicitly. Minimal: allow `type: cv` end to
  end — add CV to `NoteType` (with `Folder()`), widen the CHECK in
  migration 1 (pre-1.0: the baseline may move, see ADD-0003), skip cv
  files in FTS indexing or index them like notes. Zero-code fallback: document
  loudly that gen files must live outside the vault — but validation
  errors on arrival beat doc-only fixes.

---

## P1 — robustness, tests, sharp edges

### 6. CLI commands have no tests (biggest test gap)
- `internal/cli` is tested only for `gen`. The remaining logic-rich code
  is untested: `resolveQuery`/`candidates` (ambiguity output, fuzzyID),
  `loadConfig`/`setConfigKey` merge order (defaults < file < env), `init`
  idempotency, `rm` prompt/purge/trash paths, `link` append + reindex,
  `list` filters/limit 0, `search` input sanitization, `new` filename
  slug. Each command currently constructs its own `Config` from
  `$HOME`, which makes them untestable as written.
- **Fix:** thread config through a small struct or accept
  `Config`/`ctx` params in the `run*` funcs (keeping the thin arg-parsing
  shell), then add table-driven tests using `t.TempDir()` +
  `STOA_VAULT`/`--config`. The `newVault` helper in `index_test` is a
  ready-made pattern to crib.

### 7. `domain`, `store`, `markdown` (parts), `cv` helpers lack tests
- `domain.ParseNoteType`/`Folder` — perfect table-driven target, zero
  tests, package currently shows `[no test files]`.
- ~~`store.Open`~~ — covered since the foreign-keys fix
  (`internal/store/store_test.go`: migration application, FK pragma,
  error paths; committed `81e8fba`).
- `frontmatter` — no test for CRLF files (`\r` handling survives via
  TrimSpace but isn't pinned), title containing `:` or `\n` (a `\n` in a
  title renders a *broken multi-line frontmatter*), tags list containing
  a literal `[]` (currently parsed as a *tag named "[]"* — see the
  `parseTagList` "special-case only applies when the whole value is
  `[]`" bug), BOM, tag `#`-strip mismatch between new (`--tags=#a` is
  NOT stripped — check `runNew` → `New` path) and reindex
  (`#a` IS stripped in `parseTagList`).
- `cv` — `stripMarkup`, `splitOrgLine` en-dash-only case, `emph`/`bold`
  unmatched-marker branches have direct unit tests only via `inline`
  partials; schema `YAML()` empty-section emission ("text" always
  emitted) is fine but untested for an empty CV.

### 8. Search: name lies, wildcards leak, truncation is silent
- **`sanitizeFTS`** builds a `LIKE` pattern, not an FTS5 MATCH query —
  the function should be named for what it does (e.g.
  `likePattern`). Rename costs nothing, the README roadmap already plans
  real FTS5 MATCH.
- `%` and `_` are not neutralized by the replacer, so user input becomes
  a LIKE wildcard (usually benign, occasionally surprising on `_` in
  snake_case searches). Escape or replace them explicitly if you want
  literal semantics.
- Results are capped at `Max: 50` with no "(truncated)" hint and no
  `--limit` flag, while `list` has `--limit`.
- Result ordering is `ORDER BY note_id` — effectively random to a user;
  order by created DESC like every other listing, or by rank later in
  the FTS5 work.

### 9. `stoa rm` trash collisions and self-linkable rm
- `.trash/` target is `filepath.Base(full)` — two notes with the same
  filename in different folders (allowed: UNIQUE is on full path) make
  the second `Move to trash` fail. Prefix trash names with the id
  fragment like `stoa new` does (`id[:8]-name.md`).
- Multiple refs are validated (resolve) fully before any file is touched
  — good — but a mid-loop failure (`os.Remove` errors) stops with some
  files already moved and no report of partial state. Either keep a
  per-target error list and print "removed N, failed: …" or make it
  atomic by pre-checking writability.

### 10. `rebuildLinks` silent 10k cap
- `maxList = 10_000` bounds `ListAllNotes` for in-memory link
  resolution; note 10_001+ are simply not link targets and there's no
  signal. At minimum, log/report truncation; longer term, resolve
  links in one SQL UPDATE...FROM or per-target lookups driven by the
  link list (which is typically much smaller than the corpus).

### 11. Reindex is not transactional
- Each upsert/tag/FTS write autocommits. A crash (or second concurrent
  stoa) mid-run leaves a half-updated index. The hash-skip design makes a
  crash cheap to repair (reindex again), but wrapping the whole pass in
  one tx (or per-doc tx) is nearly free and removes the thinking cost.
  `goose` already holds a tx per migration; mirror that.

### 12. Duplicate id or same-id files fail mid-run with opaque errors
- `UpsertNote` conflicts on `path` UNIQUE if two files share the same id
  (copied file); user gets `upsert notes/foo.md: constraint failed`
  with no hint. Detect duplicate ids in `collect` and report both paths.
- Same for two files resolving to the same title only matters for links;
  resolution ambiguity already handled.

### 13. Editor parsing mangles common configs
- `strings.Fields(editor)` breaks quoted paths with spaces
  (`"C:\Program Files\..."`, `code -w --folder-uri ...` is fine, but
  editor wrappers with quoted args are not). Either document "no quoted
  args" or split respecting simple quotes (15-line helper, no
  dependency).
- Non-interactive (`tty_other.go`) hard-skips the editor on all
  non-Linux platforms with no output about *why* beyond one line —
  consider a louder warning, since `stoa new` then creates and returns
  as if the user edited it.

### 14. `markdown.AppendLinkSection` — dead param + surprising order
- The `kind` parameter is never used inside the function (dead input).
- New links are inserted **immediately after the `## Linked` heading**,
  i.e. *before* previously existing links, so the list reads newest-first
  while the rest of the file reads in file order. Either append after the
  last bullet of the section or keep prepending but say so in the test
  name. Both behaviors should be in `links_test` (currently only the
  "existing section appended" case pins this subtly).

### 15. `stoa link` doesn't bump `updated`
- The file content changes but the frontmatter `updated` stays stale;
  listings and future "sort by updated" features will disagree with
  reality. Apply `meta.Touch()` before re-render (after fixing #3).

### 16. `runNew` writes the file before validation the editor would
assume
- Minor: if the slug collides (same title created twice within the same
  millisecond id-prefix — essentially impossible) it overwrites.
  Acceptable, but a `os.WriteFile` on a **pre-existing** target would
  silently clobber; cheap `os.OpenFile(O_EXCL)` rename-guard makes intent
  explicit.

### 17. No `version` command / `--version` flag
- `stoa --version` (and `stoa version`) treat it as a command-miss and
  print the full usage with exit 1 — users can't tell which local build
  they have, and bug reports lose the single most useful datum.
- **Fix:** add `stoa version` as a first-class command (also accept
  `--version` as an alias before arg routing). Print module version from
  `runtime/debug.ReadBuildInfo()` so it stays in lockstep with the git
  tag with zero manual bookkeeping (see the roadmap, Phase 4 item 16).
  Bounded spec: `docs/specs/version-command.md`.

---

## P2 — simplification, needless abstraction, idioms

### 18. koanf stack for three keys
- `Config` has 3 fields but the config layer pulls in koanv + TOML
  parser + providers (five imports). A ~40-line hand-rolled TOML-flat
  read/write (you already wrote the same thing twice: `frontmatter` and
  `cv.parseFrontmatter`) covers vault/editor/debug, drops two direct
  deps and matches the rest of the codebase's "restricted flat parser"
  philosophy. Alternatively keep koanf but drop the providers you don't
  use. Also `_ = k.Unmarshal(...)` silently swallows malformed values —
  surface those.

### 19. `failf` is `fmt.Errorf` with a defined name and no added semantics
- Either inline it (`fmt.Errorf`) or make it meaningful (e.g. mark user
  errors so main prints them without the usage banner). Right now it
  reads as an abstraction with zero payoff.

### 20. Command boilerplate is copy-pasted 11 times
- Every command does: `fsBase` → parse → `loadConfig` → `openStore` →
  resolve ref → `GetNoteByID` → `defer st.Close`. Extract
  `- one `withStore(cfg, func(ctx, st, vault) error) error` helper
  (tx/store lifecycle, signal-safe ctx),
- `- one `resolveNotes(ctx, st, refs...)` helper returning typed notes.
  The diff shrinks every command file by 30–40% and gives #6's tests one
  seam.

### 21. Serial duplicate of frontmatter parsing in `cv` / duplicated
`parseFrontmatter`
- `internal/cv` re-implements flat frontmatter parsing with richer
  errors and `baseLine`; justified, but its README comment ("mirrors
  internal/frontmatter's philosophy") means it's the second restricted
  parser — when the CV schema grows, consider a shared
  `internal/flatyaml`-ish helper both call. Not urgent; only if a third
  parser appears (the config work in #18 would be it).

### 22. `cli/gen.go` — repeated `schemaFor` vs `Compile`'s internally
written schema
- One-shot mode: engine writes the YAML (`Renderer.Compile` does it for
  you). Watch mode: `schemaFor` writes it. Two write paths, same artifact;
  either have `Renderer` own "write YAML next to output" or let callers
  always pass. Also `_ = pdf` (a no-op flag defined then discarded) is
  honest but could just be a comment in `fsBase("gen")` until a second
  format exists.

### 24. Small idiom nits
- `internal/cli/rm.go` anonymous `struct{ id, path string }` — a named
  `refTarget` type reads better and admits the helper in #20.
- `internal/store/store.go` `Open` ignores ctx for migrations error text;
  wrap as `goose: migrate: %w` consistently across all error paths.
- `internal/cli/cli.go` hardcodes the `col = 32` usage width; could be
  derived from command specs on init so additions can't overflow it.
- `shortDate` slices a string assuming RFC3339 — after #3's render
  changes, dates written as `2026-01-02` (date-only, per README "plain
  date") render fine, but `shortDate("2026-01")` returns "2026-01", a
  range is "2026"... acceptable; keep, but comment it or parse to a
  `time.Time` at the boundary instead.
- `cv.escapeMarkup` escapes **every** `~`, not just leading-position
  list/heading markers. The leading-marker block (`i == 0` switch over
  `- + = / ~`) is redundant with the main switch for `~`, so the intent
  (neutralize leading markers only) is lost and strings like
  `cost ~100` become `cost \~100`. Typst renders `\~` as `~` so output
  is correct today (only cosmetic noise in intermediate markup), but
  fix before the escape rules grow: make the leading switch use its own
  escaping call that does not run the general pass (or drop `~` from the
  main switch), and pin either behavior in `TestInline`.
- `cv.escapeMarkup` escapes **every** `~`, not just leading-position
  list/heading markers. The leading-marker block (`i == 0` switch over
  `- + = / ~`) is redundant with the main switch for `~`, so the intent
  (neutralize leading markers only) is lost and strings like
  `cost ~100` become `cost \~100`. Typst renders `\~` as `~` so output
  is correct today (only cosmetic noise in intermediate markup), but
  fix before the escape rules grow: make the leading switch use its own
  escaping call that does not run the general pass (or drop `~` from the
  main switch), and pin either behavior in `TestInline`.

---

## P3 — docs

### 25. README drift and gaps (partially fixed — see Completed)
- Document that gen/CV files may live **outside** the vault (or fix #4
  and reverse this).
- `stoa rm` docs: nothing about `.trash` naming collisions (todo #9) or
  `--purge` irrecoverable; the Layout comment now covers `--purge`, a
  collision note should follow the #9 fix.
- `stoa search` README says "substring search (titles, tags, bodies)" —
  correct; consider a note that it's case-insensitive LIKE under the
  hood with FTS5 MATCH planned.
- Roadmap section: fine as-is; suggestions for the public roadmap live
  below ("Feature ideas").

### 26. Formalize the ADD convention — resolved

Option (a): `docs/decisions/README.md` is the index (one line per ADD:
id, title, status, links). The per-file format stays as established by
ADD-0001: bold-key header, status first, terse bullets — no MADR
frontmatter unless an agent misreads the format.

### 27. Vault layout + `init` rework (user-flagged, spec drafted)

User flagged the vault layout on 2026-11-12: the default
`~/.stoa/vault/.stoa/` double-naming and implicit init location.
Decided: default vault root is **pwd** (refuse `$HOME`), init takes an
explicit path (exists+dir → adopt, exists+file → error, missing →
create), `.stoa/` stays the reserved per-vault namespace (Obsidian
`.obsidian` pattern; currently only `index.db`), config stays in
`~/.config/stoa/`, no implicit vault anymore, `STOA_VAULT` is runtime-
only (Model A). Draft spec: `docs/specs/vault-init.md` — awaiting
approval.

### 28. Multi-vault registry + vault targeting (parked — CRITICAL, blocks future design)

**Priority: critical.** These decisions shape how every command finds
its vault and how agents drive stoa; decide before more surface grows.
The `gh` analogy (user): each vault ≈ a repo — but vaults have no
remote URL; they self-identify via the `.stoa/` marker, so walk-up
discovery from pwd (git-style) is a live option alongside registry +
explicit targeting. Open questions (all user decisions): registry
schema (`[vaults.list]`, basename names), the `default` pointer (CLI
meaning of Obsidian's `open: true`), switching UX (`stoa use` vs
`stoa config`) with agent ergonomics first-class (`--vault <name|path>`,
`STOA_VAULT` name-or-path — keep, per spec A §4), init on an
already-registered vault (re-default or leave), future `.stoa/`
contents. Full context: "Deferred decisions" in
`docs/specs/vault-init.md`. Do not implement until designed.

Blocked by/related to #27. Open questions, **user's decisions all**:
- Registry in `~/.config/stoa/stoa.toml` (`[vaults.list]` + `default`
  pointer); vault names = folder basename (decided); collisions →
  error listing paths.
- Whether `default` (Obsidian's `open: true`) is the right concept for
  a CLI — proposed reading: "the default when nothing else specifies",
  not exclusive. Not yet confirmed.
- How vault switching works (`stoa use` vs `stoa config`), with agent
  ergonomics as a first-class concern (agents retarget vaults
  frequently; `--vault <name|path>` flag and `STOA_VAULT` env both
  feed this; env var fate — keep with name-or-path semantics is the
  current lean — undecided).
- `stoa init` on an already-registered vault: re-default or leave
  untouched (Obsidian switches; undecided).
- Future `.stoa/` contents (templates etc.) — decided to keep the
  namespace reserved; contents still open.
Do not implement until the design is approved.

---

# Suggested roadmap

Ordered so each step is shippable on its own; group sizes assume
afternoon-sized chunks.

## Phase 1 — harden the core (P0 + core P1)
1. SQL-backed `CandidateNotes` + rewrite `candidates` (#2). Tests:
   out-of-top-10 resolution, prefix ties, ambiguity message limit.
2. `link`: keep frontmatter verbatim (targeted body splice) or
   round-trip-preserving renderer (#3, #15). Test: unknown keys survive.
3. Policy + plumbing for `type: cv` inside the vault (#4): domain entry +
   migration (CHECK widening) + collect/FTS decision. Test: vault with a
   cv note still reindexes.
4. `rm`: collision-safe trash names, partial-failure reporting (#9).
5. Unify command scaffolding (#20) — mechanical refactor unlocking tests.
6. CLI test battery (#6): per-command table-driven tests against temp
   vaults.

## Phase 2 — quality of search and index
7. Reindex in a single transaction; detect duplicate ids in `collect`
 (#11, #12).
8. Address `maxList` cap visibility (#10); add `doctor`-style report of
    warnings (dangling links count, skipped files) to `reindex` output.
9. Search (pre-FTS5): `--limit`, escape wildcards, order by created,
    truncate notice (#8); rename `sanitizeFTS`.
10. Frontmatter edge cases test battery (#7): CRLF, BOM, `:` in title,
    `[]` tag, `#` normalization in tags between `new` and `reindex`
    (pick one path and document it).

## Phase 3 — the planned big rocks (already on README roadmap)
11. **FTS5 MATCH**: sqlc can't parse the hidden column — sidestep with a
    hand-written `QueryContext` method in `internal/store` beside dbx or
    a `queries_raw.go` with `// sqlc:skip`. Ship `stoa search` (ranked)
    behind it; keep old behavior as `--substring`.
12. **LLM adapter**: plain interface (`Summarizer`), an OpenAI-compatible
    adapter behind it, and features on top: `stoa summarize <ref>` plus
    auto-link suggestions (`stoa suggest <ref>` → proposed `[[links]]`
    for review). This is exactly what the two-place link design (file +
    typed DB rows) was built for.
13. **Sync / import**: `stoa import <dir>` bulk-imports and reports
    frontmatter errors per file instead of failing the walk (feed
    findings back into #12's error UX).

Phase 3.5 — agent-facing features (from docs/reference/agent-friendly-cli/):
see the review doc's §4 for the list (JSON output, error hints, truncation
reporting, discovery commands, skill file). Note: DuckDB was evaluated and
rejected as an engine — [ADD-0001](decisions/ADD-0001-duckdb-not-adopted.md);
only the agent-mode *ideas* carry over (portable to SQLite at zero cost).

## Phase 4 — polish
14. `just release` (goreleaser or a pinned go build + checksums), and
    CI on GitHub Actions running `just check`.
15. Config simplification (#18) if it still bothers you post-#8.
16. `stoa version` / `--version` (#17, spec:
    `docs/specs/version-command.md`): read `runtime/debug.BuildInfo`
    (no `-ldflags` version stamping). For `go install ...@v0.x.y` builds
    `Main.Version` is the tag itself, so the binary reports exactly what
    was installed — the git tag is the single source of truth and no
    drift is possible. Local builds report `(devel)` plus the
    `vcs.revision`/`vcs.time`/`vcs.modified` stamps; print those as
    fallback. Test: snapshot each mode.
19. `stoa version` / `--version` (#17, spec:
    `docs/specs/version-command.md`): read `runtime/debug.BuildInfo`
    (no `-ldflags` version stamping). For `go install ...@v0.x.y` builds
    `Main.Version` is the tag itself, so the binary reports exactly what
    was installed — the git tag is the single source of truth and no
    drift is possible. Local builds report `(devel)` plus the
    `vcs.revision`/`vcs.time`/`vcs.modified` stamps; print those as
    fallback. Test: snapshot each mode.

---

# Feature ideas (new capability backlog)

Each is deliberately small-batch and consistent with "vault is the source
of truth; DB is disposable".

| # | Idea | Why it fits now |
|---|------|------------------|
| 1 | **`stoa rename <ref> <new title>`** | Right now editing the title in $EDITOR doesn't rename the file. Rename = rewrite frontmatter + move file + reindex (which dedups). This is THE papercut for a title-driven UX. |
| 2 | **`stoa tag <ref> +tag -tag`** | Mutating tags without editing the file by hand; complements `--tags=` on new. |
| 3 | **`stoa open <ref>`** | `xdg-open`/`open` the note or its containing folder. Two lines on Linux; the vault lives in a filesystem anyway. |
| 4 | **`stoa journal` / `stoa daily`** | Opens (idempotently creates) today's journal note, appending to it rather than creating a new file per invocation. This turns the journal type from "just another folder" into a routine. |
| 5 | **`stoa links <ref>`** (outgoing) | Symmetric with backlinks; the sqlc query already exists (`GetLinksForNote` is dead-in-the-water code right now). Also show `kind` and unresolved targets. |
| 6 | **`stoa doctor`** | Read-only vault health: files missing frontmatter, duplicate ids, unknown types, unresolved wikilinks report, stale `updated` fields. Turns #10/#12's silent failure classes into a first-class, user-friendly surface. |
| 7 | **`--json` on `list`/`search`/`show`** | Plain scriptability; the schema is already flat and typed. Concrete patterns for this (formatter indirection, NDJSON, `_meta` truncation) are in `docs/reference/agent-friendly-cli/examples.md`, decisions in [review.md](reference/agent-friendly-cli/review.md). |
| 8 | **`stoa version`** | Trivial, but missing; go-install users currently have no way to check. |
| 9 | **Templates for `stoa new`** | `stoa new doc --template meeting.md` — a `templates/` folder in the vault, with `{{title}}`-style placeholder substitution. Consistent with "plain files, no magic". |
| 10 | **`stoa config set vault --test/--verify`** | After `config vault <path>` validates that path is a directory; safer `init`-adjacent UX. |
| 11 | **Append-to-note (`stoa in <ref>`)** | Quick capture: append a line/bullet at the end of a note from stdin. Great for journaling-as-a-firehose and pairs with #4. |
| 12 | **Watch the vault (daemon-lite)** | A long-running `stoa watch` that reindexes on fsnotify events. The hash-skip write path already makes this nearly free; unlocks editors other than nvim, and any editor at all (no more "run `stoa reindex` after editing"). |
