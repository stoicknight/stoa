# stoa — repo review todo

Full-repo review, package by package and doc by doc. `just check` passes
(gofmt, vet, golangci-lint 0 issues, tests green) — findings below are
design, edge-case and completeness issues, not gate failures.

Findings are prioritized: **P0** = correctness bug / data-risk, **P1** =
robustness, missing tests, sharp edges, **P2** = simplification /
deduplication / idiom, **P3** = docs & polish. After the findings there is
a suggested roadmap and a feature-idea backlog.

---

## P0 — correctness bugs (fix first)

### 1. Foreign keys are off → orphaned rows poison the index
- **Where:** `internal/store/store.go:32` (Open DSN)
- The DSN sets `busy_timeout` and `journal_mode` but **not**
  `foreign_keys`. Verified: SQLite defaults `foreign_keys = 0` and the
  modernc driver does not change that, so every
  `REFERENCES ... ON DELETE CASCADE` in `00001_init.sql` is dead text.
- Consequence: `index.pruneOrphans` deletes a note row, but its `tags`
  rows (and stale `links` rows on source_id/target_id) silently remain.
  `ListNotesByTag` joins notes so output hides it, but the DB carries
  garbage and the tag filter's counts/dedup assumptions break down the
  line.
- **Fix:** add `&_pragma=foreign_keys(1)` to the DSN in `store.Open`.
  One-line fix, validate by extending `index_test` with a
  "prune leaves no tag rows" assertion.

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
  migration `CHECK (type IN ('note','journal','doc'))`
- `stoa gen` explicitly works on "any markdown file", and
  `examples/cv-example.md` carries `type: cv`. `stoa new` cannot create
  cv notes, but users *will* drop CV notes into the vault (it's a notes
  tool). `collect` then fails the whole walk with
  `unknown note type "cv"`, so tags/search/list go dead until the file is
  removed by hand.
- **Fix:** decide the policy explicitly. Minimal: allow `type: cv` end to
  end — add CV to `NoteType` (with `Folder()`), widen the migration
  CHECK in a new migration (migration 2: `ALTER`/rebuild, or just accept
  the CHECK is advisory and drop it in migration 2), skip cv files in
  FTS indexing or index them like notes. Zero-code fallback: document
  loudly that gen files must live outside the vault — but validation
  errors on arrival beat doc-only fixes.

### 5. `gen --theme modern --template x` silently ignores the conflict
- **Where:** `internal/cli/gen.go:41`
- The exclusivity check is `if *theme != "modern" && *templatePath != ""`.
  Passing the *default* theme plus a template slips through and renders
  with the template with no warning; passing any other theme errors.
  Two different behaviors for the same user mistake.
- **Fix:** compute the error from whether the flags were *explicitly
  set* (`fs.Changed("theme") && fs.Changed("template")`), then pick the
  renderer with a plain if/else:
  ```go
  var r *cv.Renderer
  if *templatePath != "" { r, err = cv.NewRenderer(*templatePath) } else { r, err = cv.NewTheme(*theme) }
  ```
  (also removes the current awkward "construct, maybe overwrite, then
  check error" double-assignment).

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
- `store.Open` — no test even with a migrated in-tmp-dir DB (migration
  application + WAL/hashing behavior untested end-to-end).
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

---

## P2 — simplification, needless abstraction, idioms

### 17. koanf stack for three keys
- `Config` has 3 fields but the config layer pulls in koanv + TOML
  parser + providers (five imports). A ~40-line hand-rolled TOML-flat
  read/write (you already wrote the same thing twice: `frontmatter` and
  `cv.parseFrontmatter`) covers vault/editor/debug, drops two direct
  deps and matches the rest of the codebase's "restricted flat parser"
  philosophy. Alternatively keep koanf but drop the providers you don't
  use. Also `_ = k.Unmarshal(...)` silently swallows malformed values —
  surface those.

### 18. `failf` is `fmt.Errorf` with a defined name and no added semantics
- Either inline it (`fmt.Errorf`) or make it meaningful (e.g. mark user
  errors so main prints them without the usage banner). Right now it
  reads as an abstraction with zero payoff.

### 19. Command boilerplate is copy-pasted 11 times
- Every command does: `fsBase` → parse → `loadConfig` → `openStore` →
  resolve ref → `GetNoteByID` → `defer st.Close`. Extract
  `- one `withStore(cfg, func(ctx, st, vault) error) error` helper
  (tx/store lifecycle, signal-safe ctx),
- `- one `resolveNotes(ctx, st, refs...)` helper returning typed notes.
  The diff shrinks every command file by 30–40% and gives #6's tests one
  seam.

### 20. Serial duplicate of frontmatter parsing in `cv` / duplicated
`parseFrontmatter`
- `internal/cv` re-implements flat frontmatter parsing with richer
  errors and `baseLine`; justified, but its README comment ("mirrors
  internal/frontmatter's philosophy") means it's the second restricted
  parser — when the CV schema grows, consider a shared
  `internal/flatyaml`-ish helper both call. Not urgent; only if a third
  parser appears (the config work in #17 would be it).

### 21. `cli/gen.go` — repeated `schemaFor` vs `Compile`'s internally
written schema
- One-shot mode: engine writes the YAML (`Renderer.Compile` does it for
  you). Watch mode: `schemaFor` writes it. Two write paths, same artifact;
  either have `Renderer` own "write YAML next to output" or let callers
  always pass. Also `_ = pdf` (a no-op flag defined then discarded) is
  honest but could just be a comment in `fsBase("gen")` until a second
  format exists.

### 22. `store.Open` global goose state
- `goose.SetBaseFS` / `SetDialect` are process-global with no mutex and
  would misbehave in a server context (fine for a CLI, but wrap in one
  `sync.Once` or init-time call if you ever reuse `store` from a long
  process, e.g. a watch/LSP-mode side-car).

### 23. Small idiom nits
- `internal/cli/rm.go` anonymous `struct{ id, path string }` — a named
  `refTarget` type reads better and admits the helper in #19.
- `internal/store/store.go` `Open` ignores ctx for migrations error text;
  wrap as `goose: migrate: %w` consistently across all error paths.
- `internal/cli/cli.go` hardcodes the `col = 32` usage width; could be
  derived from command specs on init so additions can't overflow it.
- `shortDate` slices a string assuming RFC3339 — after #3's render
  changes, dates written as `2026-01-02` (date-only, per README "plain
  date") render fine, but `shortDate("2026-01")` returns "2026-01", a
  range is "2026"... acceptable; keep, but comment it or parse to a
  `time.Time` at the boundary instead.

---

## P3 — docs

### 24. README drift and gaps
- `stoa new ...` — README says "create + open `$EDITOR`" but it uses
  `config.editor` (`$EDITOR` only as default). One word, worth fixing:
  docs are the first thing a new user reads.
- Document that gen/CV files may live **outside** the vault (or fix #4
  and reverse this).
- `stoa rm` docs: nothing about `.trash` naming or `--purge`
  irrecoverable; a sentence prevents surprise.
- `stoa search` README says "substring search (titles, tags, bodies)" —
  correct; consider a note that it's sub-case-insensitive LIKE under the
  hood with FTS5 MATCH planned.
- `stoa.example.toml` duplicates the header comment `< ~/.config/stoa/...`
  but there's no copy mechanism; either `stoa init` prints the path for
  that file (it does print it — good) and the example file notes that
  `stoa config key value` is the safer write path.
- Roadmap section: fine as-is; suggestions for the public roadmap live
  below ("Feature ideas").

---

# Suggested roadmap

Ordered so each step is shippable on its own; group sizes assume
afternoon-sized chunks.

## Phase 1 — harden the core (P0 + core P1)
1. `store`: add `foreign_keys(1)` DSN pragma. Test: prune drops tag/links
   rows (#1).
2. SQL-backed `CandidateNotes` + rewrite `candidates` (#2). Tests:
   out-of-top-10 resolution, prefix ties, ambiguity message limit.
3. `link`: keep frontmatter verbatim (targeted body splice) or
   round-trip-preserving renderer (#3, #15). Test: unknown keys survive.
4. Policy + plumbing for `type: cv` inside the vault (#4): domain entry +
   migration 2 for the CHECK + collect/FTS decision. Test: vault with a
   cv note still reindexes.
5. `gen`: real flag `Changed()` exclusivity + clean renderer selection
   (#5, fold in #21 simplifications).
6. `rm`: collision-safe trash names, partial-failure reporting (#9).
7. Unify command scaffolding (#19) — mechanical refactor unlocking tests.
8. CLI test battery (#6): per-command table-driven tests against temp
   vaults.

## Phase 2 — quality of search and index
9. Reindex in a single transaction; detect duplicate ids in `collect`
 (#11, #12).
10. Address `maxList` cap visibility (#10); add `doctor`-style report of
    warnings (dangling links count, skipped files) to `reindex` output.
11. Search (pre-FTS5): `--limit`, escape wildcards, order by created,
    truncate notice (#8); rename `sanitizeFTS`.
12. Frontmatter edge cases test battery (#7): CRLF, BOM, `:` in title,
    `[]` tag, `#` normalization in tags between `new` and `reindex`
    (pick one path and document it).

## Phase 3 — the planned big rocks (already on README roadmap)
13. **FTS5 MATCH**: sqlc can't parse the hidden column — sidestep with a
    hand-written `QueryContext` method in `internal/store` beside dbx or
    a `queries_raw.go` with `// sqlc:skip`. Ship `stoa search` (ranked)
    behind it; keep old behavior as `--substring`.
14. **LLM adapter**: plain interface (`Summarizer`), an OpenAI-compatible
    adapter behind it, and features on top: `stoa summarize <ref>` plus
    auto-link suggestions (`stoa suggest <ref>` → proposed `[[links]]`
    for review). This is exactly what the two-place link design (file +
    typed DB rows) was built for.
15. **Sync / import**: `stoa import <dir>` bulk-imports and reports
    frontmatter errors per file instead of failing the walk (feed
    findings back into #12's error UX).

## Phase 4 — polish
16. `just release` (goreleaser or a pinned go build + checksums), and
    CI on GitHub Actions running `just check`.
17. Config simplification (#17) if it still bothers you post-#8.
18. Theme/template docs: a short `examples/custom-template.md` walking
    the schema (the YAML artifact makes this genuinely easy to document
    with before/after snippets).

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
| 7 | **`--json` on `list`/`search`/`show`** | Plain scriptability; the schema is already flat and typed. |
| 8 | **`stoa version`** | Trivial, but missing; go-install users currently have no way to check. |
| 9 | **Templates for `stoa new`** | `stoa new doc --template meeting.md` — a `templates/` folder in the vault, with `{{title}}`-style placeholder substitution. Consistent with "plain files, no magic". |
| 10 | **`stoa config set vault --test/--verify`** | After `config vault <path>` validates that path is a directory; safer `init`-adjacent UX. |
| 11 | **Append-to-note (`stoa in <ref>`)** | Quick capture: append a line/bullet at the end of a note from stdin. Great for journaling-as-a-firehose and pairs with #4. |
| 12 | **Watch the vault (daemon-lite)** | A long-running `stoa watch` that reindexes on fsnotify events. The hash-skip write path already makes this nearly free; unlocks editors other than nvim, and any editor at all (no more "run `stoa reindex` after editing"). |
