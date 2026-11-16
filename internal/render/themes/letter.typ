// Cover letter — stoa built-in "letter" theme.
// Structure + spacing mirror typst-letter-pro (Sematre, DIN 5008):
// sender block -> v(0.9em) -> date -> v(1.2cm) -> recipient block
// -> v(0.9cm) -> subject -> v(0.9em) -> body paragraphs (justified, no
// hyphenation, default par spacing) -> v(1cm) -> name.
// The body prose carries the salutation and the closing line; the
// template adds only the sign-off gap and the sender name. Sender and
// recipient fields left empty in the note render nothing. The date is
// generated at compile time (no date line in the note). Accent color is
// the CV theme accent; the sender block is black per letter convention.
#let data = yaml("data.yaml")
#let accent = rgb("#135996")
#let ink = rgb("#222222")

#set page(paper: "a4", margin: (top: 25mm, bottom: 25mm, left: 25mm, right: 20mm))
#set text(font: ("Source Sans 3", "Liberation Sans"), size: 11pt, lang: "en", fill: ink, hyphenate: false)
#set par(justify: true, leading: 0.65em, spacing: 1.2em)
#set document(author: data.name, title: data.subject)

// --- Sender block (vertical contact lines, letter-pro style) ---
// One paragraph with line breaks — same tight leading as the recipient
// block. Empty fields are skipped.
#let sender = (
  strong(data.name),
  if data.location != "" { text(size: 10pt, data.location) },
  if data.email != "" { text(size: 10pt, link("mailto:" + data.email, data.email)) },
  if data.tel != "" { text(size: 10pt, link("tel:" + data.tel, data.tel)) },
).filter(l => l != none)
#sender.join(linebreak())

#v(0.9em, weak: false)

// --- Date (auto-generated at compile time) ---
#datetime.today().display("[day] [month repr:long] [year]")

#v(1.2cm, weak: false)

// --- Recipient block (empty attn/address lines are skipped) ---
#let recipient = (
  strong(data.company),
  if data.attn != "" { data.attn },
  if data.address != "" { data.address },
).filter(l => l != none)
#recipient.join(linebreak())

#v(0.9cm, weak: false)

// --- Subject line ---
#text(weight: "bold", fill: accent)[#data.subject]

#v(0.9em, weak: false)

// --- Body: the prose is the letter (salutation and closing included) ---
// join(parbreak()) keeps each paragraph its own <par> so par.spacing applies
#data.paragraphs.join(parbreak())

// --- Sign-off gap + name ---
#v(1cm, weak: true)
#data.name
