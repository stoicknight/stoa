# ADD-0002: CV rendering selects one `--theme` value with an explicit template contract

- **Status:** Decided — adopted
- **Date:** 2026-11-12
- **Implemented in:** commit `2471421` (`feat(cv): single --theme flag with explicit template contract`)
- **Contract:** [docs/reference/cv-template.md](../reference/cv-template.md)

## Context

`stoa gen` originally had two flags answering the same question (which
`.typ` file renders the CV): `--theme` for built-ins, `--template` for
custom files. The exclusivity check guessed user intent by comparing the
theme value to its default, so `--theme modern --template x` silently
rendered with the template while any other theme errored — same input
shape, two behaviors. External templates had no documented contract, and
a template that never read `data.yaml` compiled clean and silently
rendered an empty PDF.

## Decision

1. **One flag, one selection path.** `--theme` takes a built-in theme
   name (`classic` | `modern` | `minimalist`) or a `.typ` file path.
   Exact built-in names win; file paths must end in `.typ`, so a typo'd
   theme name cannot shadow a same-named file in the working directory.
   The `--template` flag is removed.
2. **Explicit template contract** (documented in the reference doc): the
   template loads `data.yaml` itself (`#let data = yaml("data.yaml")`),
   consumes the fixed schema shape, renders text fields as Typst markup
   via `eval(s, mode: "markup")`, and stays self-contained (no external
   Typst packages).
3. **Fail-fast validation at selection.** Empty templates and templates
   that never reference `data.yaml` are rejected before compile with
   actionable errors. Everything else fails at `typst compile`, whose
   diagnostics are relabeled from the temp workspace path to the
   template's origin (user path or built-in theme label).
4. **Default theme is `classic`.**

## Consequences

- No flag-combination logic exists; new selection behavior (e.g. more
  themes, template variants) extends one code path (`cv.NewTheme`).
- Template contract changes require updating
  `docs/reference/cv-template.md` and `cv.validateTemplate` together.
- Template authors read the reference doc; the minimal skeleton in it is
  kept compiling against stoa's schema (verified in review).
- `--theme` for a custom file is a breaking rename of the old
  `--template` flag (v0.1.0/v0.1.1 shipped both flags).
