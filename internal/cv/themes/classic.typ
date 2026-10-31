// stoa "classic" theme — sans, single accent color, partial section
// rules. A faithful, self-contained reimplementation of the "classic"
// look of rendercv 0.3.0 (Typst package @preview/rendercv, sample
// examples/classic.typ) against stoa's data.yaml schema — no external
// Typst packages. Deliberate deviations from the rendercv sample:
//   - A4 paper instead of US Letter (EU-style CV)
//   - no footer / top note (stoa's schema has no date field)
//   - no connection icons (they need a Font Awesome font)
//
// rendercv parameter set reproduced here (rendercv option names in
// comments):
//   colors: name/headline/connections/section-titles/links
//     rgb(0, 79, 144); body black
//   typography: Source Sans 3; body 10pt; line-spacing 0.6em;
//     justified with hyphenation; name 30pt bold; headline 10pt;
//     section titles 1.4em bold
//   header: centered; 0.7cm below name, headline and connections;
//     connections 0.5cm apart, no separator, hyperlinked
//   section titles: "with_partial_line" (a rule fills the rest of the
//     title line), 0.5pt thick, 0.75cm above, 0.45cm below (stoa
//     tweak: rendercv uses 0.5cm above; extra breathing room)
//   sections: 0.3em between text-based entries, 1.2em between regular
//     entries
//   entries: date-and-location column 4.15cm wide, right-aligned;
//     side space 0.2cm; 0.1cm between columns; bold organization,
//     then the title on one line; bullets 0.15cm in, 0.5em to text
//
// Schema (see internal/cv/yaml.go):
//   position, name: string
//   contacts: list of {label, uri} — uri empty means plain text
//   sections: list of {name, kind, text, bullets, entries} in file order
//     kind: "paragraph" (uses text) | "bullets" (uses bullets)
//         | "entries" (uses entries)
//     entry: {title, organization, location, dateRange, bullets, details}
// Text fields hold Typst markup from stoa's inline converter; md()
// evaluates them.
//
// TODO(schema): rendercv's education degree column and employment
// durations have no stoa fields yet — add them here when the schema
// grows degree/duration.

#let data = yaml("data.yaml")

#let md(s) = eval(s, mode: "markup")

#let accent = rgb(0, 79, 144)
#let sans = ("Source Sans 3", "Source Sans Pro", "Inter", "Liberation Sans", "Nimbus Sans", "Noto Sans")

#set document(author: data.name, title: data.name + "'s CV")
#set page(paper: "a4", margin: (x: 0.7in, top: 0.7in, bottom: 0.7in))
#set text(font: sans, size: 10pt, lang: "en", hyphenate: true, fill: black)
#set par(spacing: 0cm)
#show link: it => { set text(fill: accent); it }

// Section titles (rendercv section-titles-type "with_partial_line"):
// real heading elements styled by a show rule, exactly like rendercv —
// the heading element's own block spacing is part of the look (a plain
// function loses ~6.5pt below the title). Bold accent 1.4em title with
// a 0.5pt accent box filling the rest of the same line; 0.75cm above
// (weak, stoa tweak for breathing room), 0.45cm − 0.5em below. The
// `1em / 1.2` reset and the em in the trailing v are rendercv
// verbatim — they set the title to 14pt and the v to 0.3cm − 7pt
// (rendercv uses 0.3cm; stoa bumps it to 0.45cm).
#show heading.where(level: 2): it => [
  #set text(size: (1em / 1.2))
  #set text(size: 1.4em, weight: 700, fill: accent)
  #v(0.75cm, weak: true)
  #block(breakable: false, width: 100%)[
    #it.body
    #box(width: 1fr, height: 0.5pt, fill: accent)
  ]
  #v(0.45cm - 0.5em)
]

// Text-based sections (rendercv content-area): justified, 0.6em
// leading, 0.3em between paragraphs and list items, inset 0.2cm left
// and right (rendercv entries side space) so section text lines up
// with entry text.
#let content-area(body) = {
  set par(leading: 0.6em, spacing: 0.3em + 0.6em, justify: true)
  set list(marker: ("•", "•"), indent: 0cm, spacing: 0.3em + 0.6em, body-indent: 0.5em)
  block(
    body,
    breakable: true,
    below: 0.3em + 0.6em,
    inset: (left: 0.2cm, right: 0.2cm),
    width: 100%,
  )
}

// Right-hand column (rendercv date-and-location column): location
// above date, right-aligned, regular body color.
#let datecell(e) = {
  if e.location != "" or e.dateRange != "" {
    if e.location != "" [#e.location]
    if e.dateRange != "" [
      #if e.location != "" [#parbreak()]
      #e.dateRange
    ]
  } else { [] }
}

// One entry (rendercv regular-entry): bold organization, comma, plain
// title on one line; location and date stacked in a fixed 4.15cm
// right-aligned column; bullets 0.15cm in with a 0.5em hang (0cm
// between items, only line spacing; a small v(0.4em) gap separates the
// location / date right column from the bullets). The entry block is unbreakable
// (entries-allow-page-break false), inset 0.2cm (side space) and
// spaced 1.2em + 0.6em below — rendercv puts the entry rhythm on
// `below` and relies on the default block `above`.
#let entry(e) = {
  set par(leading: 0.6em, spacing: 0.6em, justify: true)
  set list(marker: ("•", "•"), indent: 0.15cm, spacing: 0.6em, body-indent: 0.5em)
  block(
    {
      grid(
        columns: (1fr, 4.15cm),
        column-gutter: 0.1cm,
        align: (left, right),
        [
          #if e.organization != "" [
            #text(weight: 700)[#md(e.organization)], #md(e.title)
          ] else [
            #text(weight: 700)[#md(e.title)]
          ]
        ],
        datecell(e),
      )
      if e.bullets.len() > 0 { v(0.4em) + list(..e.bullets.map(md)) }
      if e.details.len() > 0 { e.details.map(d => md(d)).join(parbreak()) }
    },
    breakable: false,
    below: 1.2em + 0.6em,
    inset: (left: 0.2cm, right: 0.2cm),
    width: 100%,
  )
}

// --- header (rendercv h1 + headline + connections) ---
// 30pt bold accent name; 0.7cm below it the position as a plain 10pt
// accent headline; 0.7cm below that the connections: accent-colored,
// hyperlinked, 0.5cm apart (rendercv separator is empty). Connections
// wrap naturally between items (weak gaps collapse at line breaks).
#align(center)[
  #set par(spacing: 0pt)
  #text(size: 30pt, weight: 700, fill: accent)[#data.name]
]
#v(0.7cm, weak: true)
#if data.position != "" [
  #block(width: 100%)[#align(center)[#text(fill: accent)[#data.position]]]
  #v(0.7cm, weak: true)
]
#if data.contacts.len() > 0 [
  #align(center)[
    #set par(spacing: 0pt, leading: 0.6em * 1.7, justify: false)
    #set text(fill: accent)
    #for (i, c) in data.contacts.enumerate() [
      #if i > 0 [#h(0.5cm, weak: true)]
      #if c.uri == "" [#box(c.label)] else [#box(link(c.uri, c.label))]
    ]
  ]
  // rendercv: header-space-below-connections (0.7cm) minus
  // section-titles-space-above (0.5cm)
  #v(0.7cm - 0.5cm)
]

// --- sections, in file order ---
// Level-2 headings pick up the show rule above; section content is
// wrapped in a breakable block (rendercv sections-allow-page-break);
// entries skip the content-area wrapper, text-based sections go
// through it.
#for s in data.sections [
  == #s.name
  #block(breakable: true)[
    #if s.kind == "paragraph" [#content-area[#md(s.text)]]
    #if s.kind == "bullets" [#content-area[#list(..s.bullets.map(md))]]
    #if s.kind == "entries" [#for e in s.entries [#entry(e)]]
  ]
]
