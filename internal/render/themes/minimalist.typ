// stoa "minimalist" theme — modern, minimalist, print-safe (no color
// fills, works in B&W). The default for `stoa gen` is the "classic"
// theme (themes/classic.typ); this file remains a built-in theme.
// Data comes from data.yaml, the stable schema
// produced by stoa's cv parser:
//
//   position, name: string
//   contacts: list of {label, uri} — uri empty means plain text
//   sections: list of {name, kind, text, bullets, entries} in file order
//     kind: "paragraph" (uses text) | "bullets" (uses bullets)
//         | "entries" (uses entries)
//     entry: {title, organization, location, dateRange, bullets, details}
//
// Text fields contain Typst markup produced by stoa's inline converter;
// md() evaluates them. Reformatting the CV means writing a new template
// against the same schema — no stoa changes needed.

#let data = yaml("data.yaml")

#let md(s) = eval(s, mode: "markup")

#set page(paper: "a4", margin: (x: 1.9cm, top: 1.5cm, bottom: 1.5cm))
#set text(
  font: ("Inter", "Source Sans 3", "Liberation Sans", "Nimbus Sans", "Noto Sans", "Libertinus Serif"),
  size: 9.5pt,
  lang: "en",
)
#set par(leading: 0.6em, spacing: 0.85em, justify: false)
#show link: it => { set text(fill: black); it }

// Section heading: spaced caps with a thin rule. above: 0.75cm,
// below: 0.8em (stoa tweaks for breathing room; was 1.15em / 0.6em).
#let heading(name) = block(above: 0.75cm, below: 0.8em, width: 100%)[
  #text(size: 10pt, weight: 600, tracking: 0.1em)[#upper(name)]
  #v(0.3em)
  #line(length: 100%, stroke: 0.4pt + luma(150))
]

// Tight bullet list with a hanging indent.
#let tightlist(items) = grid(
  columns: (auto, 1fr),
  column-gutter: 0.55em,
  row-gutter: 0.32em,
  ..items.map(b => (align(top)[•], md(b))).flatten(),
)

// One entry: bold title with right-aligned dates, bold organization —
// location line, bullets, then small detail lines (e.g. study focus).
// Grid children are explicit call arguments — inside a trailing content
// block, consecutive inline expressions merge into ONE paragraph and
// thus ONE cell (typst 0.15 behavior).
#let entry(e) = block(above: 0.9em, below: 0.9em, breakable: false)[
  #grid(
    columns: (1fr, auto),
    align: (left, right),
    column-gutter: 1em,
    text(size: 10pt, weight: 600)[#md(e.title)],
    if e.dateRange != "" { text(size: 8.5pt, style: "italic", fill: luma(90))[#e.dateRange] } else { [] },
  )
  #if e.organization != "" or e.location != "" [
    #text(weight: 600)[#if e.organization != "" [#md(e.organization)]]
    #if e.location != "" [#if e.organization != "" [#h(0.35em)—#h(0.35em)]#e.location]
  ]
  #if e.bullets.len() > 0 [#v(-0.1em)#tightlist(e.bullets)]
  #for d in e.details [#block(above: 0.45em)[#text(size: 8.5pt, fill: luma(90))[#md(d)]]]
]

// --- header ---
#align(center)[
  #text(size: 8.5pt, weight: 600, tracking: 0.18em)[#upper(data.position)]
  #v(0.35em)
  #text(size: 19pt, weight: 700)[#data.name]
  #v(0.5em)
  #for (i, c) in data.contacts.enumerate() [
    #if i > 0 [#h(0.4em)·#h(0.4em)]
    #if c.uri == "" [#box(c.label)] else [#box(link(c.uri, c.label))]
  ]
]
#v(0.2em)
#line(length: 100%, stroke: 0.5pt + luma(120))

// --- sections, in file order ---
#for s in data.sections [
  #heading(s.name)
  #if s.kind == "paragraph" [#md(s.text)]
  #if s.kind == "bullets" [#tightlist(s.bullets)]
  #if s.kind == "entries" [#for e in s.entries [#entry(e)]]
]
