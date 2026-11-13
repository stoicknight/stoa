# Spec: `stoa version` / `--version` (todo #17)

Status: revised — review decisions incorporated, awaiting final go-ahead.

## Resolved design decisions (review 2026-11-12)

1. Output follows the `gh version` multi-line layout, fixed at **2 lines**
   (line 1: version info; line 2: a URL).
2. Devel builds include the build timestamp **and** the short commit
   hash in line 1's parenthetical, as in the original example.

## Goal

`stoa version` and `stoa --version` report the version of the running
binary: the module tag for `go install ...@vX.Y.Z` builds, commit info
for local `(devel)` builds. Single source of truth is the build itself
(`runtime/debug`), so no version string can drift from the git tag.

## Scope

- **Create:** `internal/cli/version.go`
- **Create:** `internal/cli/version_test.go`
- **Modify:** `internal/cli/cli.go` — one registry entry in `commands`;
  `--version` alias handled in `Run` before dispatch
- **NOT to touch:** `cmd/stoa/main.go` (alias lives in `cli.Run`, where
  dispatch already happens), store/domain/markdown/cv packages, Justfile.

## Design

New `internal/cli/version.go`:

```go
// versionString renders the human-readable version line from build
// info so the same code path serves tagged and local builds.
func versionString(bi *debug.BuildInfo) string
```

Every output is exactly two lines, `gh version`-style: line 1 is the
version information, line 2 is a URL.

- Tagged build (`"vX.Y.Z"`, `vcs.modified` false):
  ```
  stoa version v0.1.1
  https://github.com/stoicknight/stoa/releases/tag/v0.1.1
  ```
- Tagged with dirty tree (`vcs.modified` true):
  ```
  stoa version v0.1.1+dirty
  https://github.com/stoicknight/stoa/releases/tag/v0.1.1
  ```
- Local build with stamps (built inside a git checkout):
  ```
  stoa version (devel) (commit 1a2b3c4, built 2025-06-01 10:00:12 UTC, modified)
  https://github.com/stoicknight/stoa/commit/<full-40-char-sha>
  ```
  Omit clauses of the parenthetical whose stamps are absent: without
  `vcs.time` drop the `built` clause, without `vcs.modified=true` drop
  `modified`; the `commit` clause is dropped together with the URL line
  when `vcs.revision` is absent (e.g. binary built outside a git
  checkout → single line `stoa version (devel)`).
- `bi == nil` → single line `stoa version (unknown)` (defensive;
  ReadBuildInfo practically never returns nil).

Notes:
- Line 2 for tagged installs is the GitHub release URL; for devel
  builds with a revision it is the commit URL. The host/repo base is
  derived from `bi.Main.Path` (`github.com/stoicknight/stoa`), not
  hardcoded.
- Tagged installs built through the module proxy (`go install
  ...@v0.1.1`) carry no VCS stamps, so they never show `built`/`commit`
  clauses — only the release URL line.
- The `built` timestamp is `vcs.time` verbatim (RFC 3339 UTC), matching
  what Go embeds.

- `runVersion(args []string) error` — uses `fsBase("version")` so
  `--config` is accepted uniformly; stores `debug.ReadBuildInfo()` and
  prints `versionString` to stdout over `fmt.Println`. Always exits 0.

`internal/cli/cli.go`:

- Registry entry (usage output is generated, so it cannot drift):
  `{name: "version", summary: "print the binary version", run: runVersion}`
- In `Run`, before the command loop:
  `if args[0] == "--version" { args[0] = "version" }` — same trick
  already used for `--help`/`-h`. `-v` is NOT an alias (leave room for
  a future `-v` global flag; `--version` is the POSIX-sh form).

## Constraints

- Stdlib only (`runtime/debug`, `fmt`, `strings`) — no new deps.
- No `-ldflags` stamping: unusable for `go install name@tag` binaries
  and silently skipped by bare `go build`.
- Follows the repo rubric: explicit, table-driven tests; `%w` errors
  where errors exist (none expected here).

## Verification

- `internal/cli/version_test.go`:
  - Table-driven `TestVersionString` with synthetic `*debug.BuildInfo`
    values covering the shapes above (tag / tag+dirty / devel full /
    devel without time / devel without any stamps / nil). Golden
    strings asserted exactly, including the 2-line layout and URL
    derivation from `Main.Path`.
  - `TestVersionDispatch`: table over `Run(["version"])`,
    `Run(["--version"])` — both must return nil (not unknown-command);
    runVersion output is not captured (printing is trivial; the pure
    formatting is what carries the risk).
- `just check`.

## Open Questions

1. Output format — is `stoa vX.Y.Z` on a single line acceptable, or do
   you want it to match `gh`'s `gh version …` multi-line layout?
2. Include the `buildtime` in the `stoa version` output at all, or keep
   it minimal (`stoa (devel) commit 1a2b3c4`)?
