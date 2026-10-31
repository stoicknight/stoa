// stoa "modern" theme — serif, all-black ink, full-width section
// rules, underlined body links. A faithful, self-contained
// reimplementation of the "engineeringresumes" look of rendercv 0.3.0
// (Typst package @preview/rendercv, sample
// examples/engineeringresumes.typ) against stoa's data.yaml schema —
// no external Typst packages. Deliberate deviations from the sample:
//   - A4 paper instead of US Letter (EU-style CV)
//   - no footer / top note (stoa's schema has no date field)
//   - no connection icons (they need a Font Awesome font)
//
// rendercv parameter set reproduced here (rendercv option names in
// comments):
//   colors: everything black; links underlined, except header
//     connections (the sample passes if-underline: false per link)
//   typography: XCharter; body 10pt; line-spacing 0.6em; justified
//     with hyphenation; name 25pt regular; headline 10pt; section
//     titles 1.2em bold
//   header: centered; 0.7cm below name, headline and connections;
//     connections "|" separated, 0.5cm apart, hyperlinked
//   section titles: "with_full_line" (a rule under the title), 0.5pt
//     thick, 0.75cm above, 0.45cm below (stoa tweaks for breathing
//     room: rendercv uses 0.5cm / 0.3cm)
//   sections: 0.15cm between text-based entries, 0.42cm between
//     regular entries
//   entries: date column 4.15cm wide, right-aligned; side space 0cm;
//     0.1cm between columns; bold title, comma, organization, " – "
//     location on one line; bullets flush left with a 0.3em hang,
//     0.08cm between items, 13pt bullet glyph
//
// Schema (see internal/cv/yaml.go):
//   position, name: string
//   contacts: list of {label, uri} — uri empty means plain text
//   sections: list of {name, kind, text, bullets, entries} in file order
//     kind: "paragraph" (uses text) | "bullets" (uses bullets)
//         | "entries" (uses entries)
//     entry: {title, organization, location, dateRange, bullets, details}
// Text fields hold Typst markup from stoa's inline converter; md()
// evaluates them — with rendercv's link styling baked in (underlined
// black; typst show rules chain, so a later local rule cannot undo a
// global one — hence no global rule and styling at creation scope).
//
// TODO(schema): rendercv's education degree column and employment
// durations have no stoa fields yet — add them here when the schema
// grows degree/duration.

#let data = yaml("data.yaml")

// Body-text markup: links underlined, black (rendercv links-underline
// true). Header connections stay plain: the show rule lives inside
// md(), so it only touches eval'd body content.
#let md(s) = {
  show link: it => { set text(fill: black); underline(it) }
  eval(s, mode: "markup")
}

#let serif = ("XCharter", "Charter", "Bitstream Charter", "Libertinus Serif", "Tinos", "Liberation Serif")
// rendercv entries-highlights-bullet: 13pt glyph, baseline nudged up
#let glyph = text(size: 13pt, baseline: -0.6pt)[•]

#set document(author: data.name, title: data.name + "'s CV")
#set page(paper: "a4", margin: (x: 0.7in, top: 0.7in, bottom: 0.7in))
#set text(font: serif, size: 10pt, lang: "en", hyphenate: true, fill: black)
#set par(spacing: 0cm)

// Section titles (rendercv section-titles-type "with_full_line"):
// real heading elements styled by a show rule, exactly like rendercv —
// the heading element's own block spacing is part of the look (a plain
// function loses ~6.5pt below the title). Bold 1.2em title, 3pt below
// it a full-width 0.5pt rule; 0.75cm above (weak), 0.45cm − 0.5em
// below.
// The `1em / 1.2` reset and the em in the trailing v are rendercv
// verbatim — they set the title to 12pt and the v to 0.3cm − 6pt
// (rendercv uses 0.3cm; stoa bumps it to 0.45cm).
#show heading.where(level: 2): it => [
  #set text(size: (1em / 1.2))
  #set text(size: 1.2em, weight: 700)
  #v(0.75cm, weak: true)
  #block(breakable: false, width: 100%)[
    #it.body
    #v(10pt * 0.3)
    #box(width: 1fr, height: 0.5pt, fill: black)
  ]
  #v(0.45cm - 0.5em)
]

// Text-based sections (rendercv content-area): justified, 0.6em
// leading, 0.15cm between paragraphs and list items; side space 0cm,
// so no inset — text sits at the margin like the entries.
#let content-area(body) = {
  set par(leading: 0.6em, spacing: 0.15cm + 0.6em, justify: true)
  set list(marker: (glyph, glyph), indent: 0cm, spacing: 0.15cm + 0.6em, body-indent: 0.3em)
  block(
    body,
    breakable: true,
    below: 0.15cm + 0.6em,
    width: 100%,
  )
}

// Entry first line, built in code mode so no markup whitespace sneaks
// in between the pieces (a line break in markup is a space): bold
// title, comma, organization, " – " location — rendercv's
// `#strong[Title], Org – Location`.
#let entryline(e) = {
  let line = text(weight: 700)[#md(e.title)]
  if e.organization != "" { line += [, #md(e.organization)] }
  if e.location != "" { line += [#h(0.3em)–#h(0.3em)#e.location] }
  line
}

// One entry (rendercv regular-entry): the first line above; the date
// alone in a fixed 4.15cm right-aligned column; bullets flush left,
// 0.3em hang, 0.08cm between items, 0.08cm above the list. Unbreakable
// (entries-allow-page-break false), spaced 0.42cm + 0.6em below —
// rendercv puts the entry rhythm on `below` and relies on the default
// block `above`.
#let entry(e) = {
  set par(leading: 0.6em, spacing: 0.6em, justify: true)
  set list(marker: (glyph, glyph), indent: 0cm, spacing: 0.08cm + 0.6em, body-indent: 0.3em)
  block(
    {
      grid(
        columns: (1fr, 4.15cm),
        column-gutter: 0.1cm,
        align: (left, right),
        entryline(e),
        if e.dateRange != "" [#e.dateRange] else { [] },
      )
      if e.bullets.len() > 0 {
        v(0.08cm)
        list(..e.bullets.map(md))
      }
      if e.details.len() > 0 { e.details.map(d => md(d)).join(parbreak()) }
    },
    breakable: false,
    below: 0.42cm + 0.6em,
    width: 100%,
  )
}

// --- header (rendercv h1 + headline + connections) ---
// 25pt regular black name; 0.7cm below it the position as a plain 10pt
// headline; 0.7cm below that the connections: black, hyperlinked, NOT
// underlined, "|" separated, 0.5cm apart. Connections wrap naturally
// between items (weak gaps collapse at line breaks).
#align(center)[
  #set par(spacing: 0pt)
  #text(size: 25pt)[#data.name]
]
#v(0.7cm, weak: true)
#if data.position != "" [
  #block(width: 100%)[#align(center)[#data.position]]
  #v(0.7cm, weak: true)
]
#if data.contacts.len() > 0 [
  #align(center)[
    #set par(spacing: 0pt, leading: 0.6em * 1.7, justify: false)
    #for (i, c) in data.contacts.enumerate() [
      #if i > 0 [#h(0.25cm, weak: true)|#h(0.25cm, weak: true)]
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
