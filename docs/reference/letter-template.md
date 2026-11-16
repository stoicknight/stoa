# Letter template contract

`stoa gen --theme <file>.typ` renders a cover letter (`type: letter`) with a
user-provided Typst template. The template replaces stoa's built-in letter
theme (`internal/render/themes/letter.typ` — read it for a working example).

The rules are identical to the CV contract
([cv-template.md](cv-template.md)):

1. **Load the schema yourself** — stoa writes `data.yaml` next to the
   template and refreshes it on every change (watch mode included):

   ```typst
   #let data = yaml("data.yaml")
   ```

   A template that never reads `data.yaml` is rejected before compile.

2. **Consume the schema shape below.** Wrong field access fails at
   `typst compile` with a diagnostic naming your template file and line.

3. **`paragraphs` hold Typst markup.** stoa converts the note's inline
   markdown (`**bold**`, `*italic*`, `[text](url)`) to Typst markup; render
   those strings with `eval(s, mode: "markup")`.

4. **Self-contained.** No external Typst packages; fonts must exist on
   the system or fall back gracefully.

## Schema (`data.yaml`)

Source of truth: `internal/render/letter.go`.

| field | shape |
|---|---|
| `name`, `position`, `location`, `tel`, `email` | string — sender identity; everything except `name` may be `""` |
| `company`, `attn`, `address` | string — recipient block; `attn`/`address` may be `""` |
| `subject` | string (from frontmatter `subject:`, else the single `#` heading in the body) |
| `paragraphs` | list of strings (Typst markup), in file order |

Empty sender/recipient fields must render nothing — skip them in the
template (the built-in theme filters them out).

## Letter semantics fixed by stoa (not the template's choice)

- **The prose is the letter.** The salutation and the closing line are
  ordinary paragraphs the user writes; the template adds only the
  sign-off gap and the name after the last paragraph.
- The date line is generated at compile time, not stored in the note.
- Paragraph separation comes from paragraph spacing: join the list with
  `parbreak()` so each string stays its own `<par>`.

## Minimal skeleton

```typst
#let data = yaml("data.yaml")

#set page(paper: "a4", margin: (top: 25mm, bottom: 25mm, left: 25mm, right: 20mm))
#set text(font: ("Source Sans 3", "Liberation Sans"), size: 11pt, hyphenate: false)
#set par(justify: true, leading: 0.65em, spacing: 1.2em)
#set document(author: data.name, title: data.subject)

#let sender = (
  strong(data.name),
  if data.location != "" { text(size: 10pt, data.location) },
  if data.email != "" { text(size: 10pt, link("mailto:" + data.email, data.email)) },
  if data.tel != "" { text(size: 10pt, link("tel:" + data.tel, data.tel)) },
).filter(l => l != none)
#sender.join(linebreak())

#let recipient = (
  strong(data.company),
  if data.attn != "" { data.attn },
  if data.address != "" { data.address },
).filter(l => l != none)
#recipient.join(linebreak())

#text(weight: "bold")[#data.subject]
#v(0.9em, weak: false)

#data.paragraphs.join(parbreak())

#v(1cm, weak: true)
#data.name
```
