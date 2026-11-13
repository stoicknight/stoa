# Spec: vault layout & `init` rework (todo #27)

Status: **draft v1 — awaiting approval.** The multi-vault/registry
decisions are deliberately **not** part of this spec: they are parked in
[docs/todo.md #28](../todo.md) (marked critical) and listed under
"Deferred decisions" below. Nothing in that list may be implemented
here.

## Problem

- The default vault root `~/.stoa/vault` produces `~/.stoa/vault/.stoa/`
  — two different things named `.stoa` in one path.
- `stoa init` invents a location implicitly; CLI convention (cf.
  `git init`) is to act on pwd or an explicitly given path.
- What belongs in app config vs per-vault state was never stated.

## Decided

### 1. Default vault root is pwd

- `stoa init` with no argument targets the current working directory.
- Guard: pwd == `$HOME` → refuse:
  `refusing to create a vault in $HOME — pass a path`.
- pwd applies to `init` only; every other command keeps resolving the
  configured vault.

### 2. `stoa init [path]` matrix

| state of `path` | behavior |
|---|---|
| missing | `MkdirAll` (parents incl.) → create `notes/`, `journal/`, `docs/`, `.trash/`, `.stoa/` → create index → print `vault ready: <path> (config: <path>)` |
| exists, is dir | adopt: create missing subdirs only, ensure index, same output |
| exists, is file | error: `path exists and is not a directory` |
| already a stoa vault (`.stoa/index.db` present) | no-op: print `already initialized: <path>`, exit 0 |
| no arg, pwd == `$HOME` | refuse (guard above) |

### 3. Config division (verified against Obsidian — same split)

- `~/.config/stoa/stoa.toml` — app-level: `editor`, `debug`, and the
  vault pointer (`vault` key, singular until the registry lands).
- `<vault>/.stoa/` — per-vault state: `index.db` today; the directory is
  a reserved namespace for future per-vault state (templates etc.).
- A vault is therefore self-contained and portable: copying the folder
  moves everything.

### 4. `STOA_VAULT` stays — runtime override only (Model A)

- It never affects what `init` creates or where; init's target is
  arg-or-pwd, full stop.
- Today its value is a raw path. After the registry lands it becomes
  name-or-path (todo #28).

### 5. No implicit vault anymore

- `Default()` no longer pre-fills `~/.stoa/vault`. A fresh install with
  no config gets, for every vault command:
  `no vault configured — run stoa init [path]` (exit 1).
- Interim rule (respects "never overwrite"): `stoa init` writes the
  `vault` key **only when the config has none**; otherwise it prints
  `config already points to <path> (multi-vault registry pending)` and
  changes nothing. Manual switch stays `stoa config vault <path>` until
  the registry design lands.

## Deferred decisions (see todo #28 — critical)

Recorded so this design pass can be revisited on its own merits:

1. **Multi-vault registry** in `stoa.toml` (`[vaults.list]`, entries
   named by folder basename — decided; collisions error with paths).
2. **The `default` pointer** — Obsidian's `open: true` is an exclusive
   GUI notion; for a CLI it can only mean "the fallback when neither
   `--vault` nor `STOA_VAULT` specifies". Name, necessity, and semantics
   still to be confirmed.
3. **Vault targeting UX** — the `gh` analogy: each vault ≈ a repo, but
   vaults have no remote URL; they *do* have a self-identification
   marker (`.stoa/`). Options to weigh: config pointer only, walk-up
   discovery from pwd (git-style, via the `.stoa/` marker), registry +
   explicit targeting (`--vault <name|path>`, `STOA_VAULT` name-or-path,
   maybe a `stoa vault list`/`stoa use` surface). Must treat agent
   ergonomics as first-class: agents retarget vaults frequently and
   without tty assumptions.
4. **`stoa init` on an already-registered vault** — re-default or leave
   untouched (Obsidian switches; undecided).
5. **Future `.stoa/` contents** — namespace reserved; contents open.

## Implementation plan

- `internal/cli/init.go` — arg parsing (`[path]`), the matrix, guard,
  no-op path, interim config rule.
- `internal/cli/config.go` — `Default()` drops the `~/.stoa/vault`
  pre-fill; vault commands error with the actionable message when
  unset.
- `internal/cli/cli.go` — usage string: `init [path]`.
- Tests (`internal/cli/init_test.go`, new): each matrix row, `$HOME`
  guard, STOA_VAULT-ignored-by-init, no-implicit-vault error, interim
  config rule, `--config` respected.
- `README.md` — Layout section (vault root is user-chosen/pwd),
  Config section (no implicit default), `init` row in the commands
  table.

## Acceptance

- `just check` green including the new tests.
- All spec behaviors above demonstrable from the CLI.
- `docs/todo.md` #27 closed with this spec's commit when implemented.
