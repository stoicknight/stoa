# Spec: `stoa version` / `--version` (todo #17)

Status: draft — awaiting approval.

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

- Tagged build (`"vX.Y.Z"` + no local mods): `stoa vX.Y.Z`
- Tagged with dirty tree (`vcs.modified`): `stoa vX.Y.Z+dirty`
- Local build: `stoa (devel) (commit 1a2b3c4, built 2025-06-01 10:00:12 UTC, modified)`,
  omitting `built`/`modified` clauses when the corresponding stamps are
  absent (e.g. binary built outside a git checkout → just `stoa (devel)`)
- `bi == nil` → `stoa (unknown)` (defensive; ReadBuildInfo practically
  never returns nil)

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
    values covering the four shapes above (tag / tag+dirty / devel with
    and without stamps / nil). Golden strings asserted exactly.
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
