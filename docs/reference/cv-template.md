# CV template contract

`stoa gen --theme <file>.typ` renders a CV with a user-provided Typst
template. The template replaces stoa's built-in themes
(`internal/cv/themes/*.typ` — read them for working examples).

## Requirements

1. **Load the schema yourself.** stoa writes `data.yaml` next to the
   template in its workspace and refreshes it on every change (watch
   mode included). The template must read it:

   ```typst
   #let data = yaml("data.yaml")
   ```

   A template that never reads `data.yaml` is rejected before compile:
   it would render an empty PDF with no error.

2. **Consume the schema shape below.** Wrong field access fails at
   `typst compile` with a diagnostic naming your template file and line.

3. **Text fields hold Typst markup.** stoa converts the markdown inline
   markup (`**bold**`, `*italic*`, `[text](url)`) to Typst markup; render
   those strings with `eval(s, mode: "markup")`.

4. **Self-contained.** No external Typst packages; fonts must exist on
   the system or fall back gracefully.

## Schema (`data.yaml`)

Source of truth: `internal/cv/yaml.go`.

| field | shape |
|---|---|
| `position`, `name` | string |
| `contacts` | list of `{label, uri}` — empty `uri` renders as plain text |
| `sections` | list of `{name, kind, text, bullets, entries}`, in file order |

`kind` picks the section renderer:

- `"paragraph"` — uses `text`
- `"bullets"` — uses `bullets` (list of strings)
- `"entries"` — uses `entries` (list of `{title, organization, location, dateRange, bullets, details}`)

## Minimal skeleton

```typst
#let data = yaml("data.yaml")
#let md(s) = eval(s, mode: "markup")

#set page(paper: "a4", margin: 0.7in)
#set text(font: ("Liberation Sans",), size: 10pt)
#set document(author: data.name, title: data.name + "'s CV")

// if/else chains must live in code blocks — `else` cannot start a
// markup expression
#let contacts = data.contacts.map(c => if c.uri == "" { c.label } else { link(c.uri, c.label) })

#let renderSection(s) = {
  heading(level: 2, s.name)
  if s.kind == "paragraph" {
    md(s.text)
  } else if s.kind == "bullets" {
    for b in s.bullets [- #md(b)]
  } else if s.kind == "entries" {
    for e in s.entries {
      heading(level: 3, md(e.title))
      md(e.organization)
      e.location
      e.dateRange
      for b in e.bullets [- #md(b)]
    }
  }
}

#text(size: 24pt, weight: "bold")[#data.name] \
#data.position \
#contacts.join(" · ")

#for s in data.sections {
  renderSection(s)
}
```
