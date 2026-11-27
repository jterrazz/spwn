package wiki

import (
	"path"
	"strings"
)

type Paragraph struct {
	// value: a paragraph of a page's prose as a reader is served it: plain,
	// an item of a list or a quote, each its runs; or a line of a markdown
	// table, its head or a row, each cell its runs.
	Form  ProseForm
	Runs  []Run
	Cells [][]Run
}

type ProseForm string

const (
	FormPlain ProseForm = "plain"
	FormItem  ProseForm = "item"
	FormHead  ProseForm = "head"
	FormRow   ProseForm = "row"
	FormQuote ProseForm = "quote"
)

type Run struct {
	// value: a run of a paragraph's text: plain, strong, code, or the words
	// of a link, its Kind and where it points: a page link the id of the
	// corpus page it lands on and the Anchor after its #, an anchor or an
	// address as written.
	Text   string
	Strong bool     `json:",omitempty"`
	Code   bool     `json:",omitempty"`
	Link   string   `json:",omitempty"`
	Anchor string   `json:",omitempty"`
	Kind   LinkKind `json:",omitempty"`
}

type LinkKind string

// value: what a prose link points at: a page of the corpus by its id, an
// anchor of the page, or an address of the web.
const (
	LinkPage     LinkKind = "page"
	LinkExternal LinkKind = "external"
	LinkAnchor   LinkKind = "anchor"
)

type Lands func(target string) (page, anchor string, ok bool)

func LandsOn(from string, holds func(id string) bool) Lands {
	// invariant: where a page link lands: the id of a page of the corpus
	// and the anchor after its #, or false when it lands on none. It is read
	// from the folder of the page it is written on, its ./ and ../ resolved
	// and its .md dropped, and lands only on a page the corpus holds: a path
	// from the root (/x), one that climbs out of the corpus, a query (?) and
	// one that names no page land nowhere.
	return func(target string) (string, string, bool) {
		t, anchor, _ := strings.Cut(target, "#")
		if t == "" || strings.HasPrefix(t, "/") || strings.ContainsRune(t, '?') {
			return "", "", false
		}
		id := strings.TrimSuffix(path.Join(path.Dir(from), t), ".md")
		if id == "." || id == ".." || strings.HasPrefix(id, "../") || !holds(id) {
			return "", "", false
		}
		return id, anchor, true
	}
}

// invariant: a link is served only to where the window may follow it: an
// anchor (#id), an http or https address, or a page of the corpus by its
// id (LandsOn). Any other scheme (javascript:, file:, mailto:), a path to
// a host (//host), a backslash, a control character or a tab, a space
// before or after the target, and a path that lands on no page are no
// link: the run is its words, plain, and the check says so (S30).
func linkKind(target string) (LinkKind, bool) {
	t := target
	switch {
	case t == "", t != strings.TrimSpace(t), strings.HasPrefix(t, "//"), strings.ContainsRune(t, '\\'), strings.ContainsFunc(t, func(r rune) bool { return r < 0x20 || r == 0x7f }):
		return "", false
	case strings.HasPrefix(t, "#"):
		return LinkAnchor, true
	}
	if i := strings.IndexAny(t, ":/?#"); i >= 0 && t[i] == ':' {
		switch strings.ToLower(t[:i]) {
		case "http", "https":
			return LinkExternal, true
		}
		return "", false
	}
	return LinkPage, true
}

func Paragraphs(markdown string, lands Lands) []Paragraph {
	return paragraphs(markdown, &linking{lands: lands})
}

// value: how a prose's links are read: where its page links land, and,
// when set, refused, told each link target no Kind serves, in reading
// order, so the check judges the very links a reader is served; and, when
// set, the meter its paragraphs and runs are measured on as they are read,
// reading stopping once it is spent.
type linking struct {
	lands   Lands
	refused func(target string)
	m       *meter
}

func (l *linking) spent() bool { return l != nil && l.m != nil && l.m.spent() }

func (l *linking) run(r Run) {
	if l != nil && l.m != nil {
		l.m.runs([]Run{r})
	}
}

func (l *linking) paragraph() {
	if l != nil && l.m != nil {
		l.m.object(3)
	}
}

func paragraphs(markdown string, links *linking) []Paragraph {
	// invariant: prose is read, never rendered as HTML: a paragraph is the
	// lines up to a blank one, a list item opens on "- " or "1. " and takes
	// the indented lines after it, and its inline marks are **strong**,
	// `code` and [words](target); anything else is its text as written.
	var out []Paragraph
	var form ProseForm
	var lines []string
	flush := func() {
		if len(lines) > 0 && !links.spent() {
			links.paragraph()
			out = append(out, Paragraph{Form: form, Runs: runs(strings.Join(lines, " "), links)})
		}
		lines = nil
	}
	for rest := markdown; rest != "" && !links.spent(); {
		var raw string
		raw, rest, _ = strings.Cut(rest, "\n")
		line := strings.TrimSpace(raw)
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "|"):
			flush()
			if strings.Trim(line, "|-: ") == "" {
				if n := len(out); n > 0 && out[n-1].Form == FormRow {
					out[n-1].Form = FormHead
				}
				continue
			}
			p := Paragraph{Form: FormRow}
			links.paragraph()
			for cells := strings.Trim(line, "|"); !links.spent(); {
				cell, after, more := strings.Cut(cells, "|")
				p.Cells = append(p.Cells, runs(strings.TrimSpace(cell), links))
				if !more {
					break
				}
				cells = after
			}
			out = append(out, p)
		case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* "):
			flush()
			form, lines = FormItem, append(lines[:0], line[2:])
		case numbered(line) > 0:
			flush()
			form, lines = FormItem, append(lines[:0], line[numbered(line):])
		case strings.HasPrefix(line, "> "):
			if form != FormQuote {
				flush()
			}
			form = FormQuote
			lines = append(lines, line[2:])
		default:
			if len(lines) == 0 {
				form = FormPlain
			}
			lines = append(lines, line)
		}
	}
	flush()
	return out
}

func numbered(line string) int {
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == 0 || !strings.HasPrefix(line[i:], ". ") {
		return 0
	}
	return i + 2
}

func plainOf(line string) string {
	// invariant: a line Face is served as the words a reader reads, its
	// marks dropped: a link keeps its words, code its text; a line of no
	// words is served empty, so its block's line face shows its word.
	var b strings.Builder
	for _, r := range runs(line, nil) {
		b.WriteString(r.Text)
	}
	return strings.TrimSpace(b.String())
}

func runs(text string, links *linking) []Run {
	var out []Run
	var plain strings.Builder
	emit := func(s Run) {
		if plain.Len() > 0 {
			r := Run{Text: plain.String()}
			links.run(r)
			out = append(out, r)
			plain.Reset()
		}
		links.run(s)
		out = append(out, s)
	}
	// invariant: the next "](" and the next ")" after it are each found
	// once and reused until reading passes them, and a search that finds
	// none is never made again, so a line of "[" or of "[](" is read once.
	mid, end := -1, -1
	link := func(i int) (int, int) {
		if mid < i {
			mid = len(text)
			if m := strings.Index(text[i:], "]("); m >= 0 {
				mid = i + m
			}
		}
		if end < mid+2 {
			end = len(text)
			if mid+2 <= len(text) {
				if e := strings.IndexByte(text[mid+2:], ')'); e >= 0 {
					end = mid + 2 + e
				}
			}
		}
		return mid, end
	}
	for i := 0; i < len(text) && !links.spent(); {
		rest := text[i:]
		switch {
		case strings.HasPrefix(rest, "**"):
			if end := strings.Index(rest[2:], "**"); end > 0 {
				emit(Run{Text: rest[2 : 2+end], Strong: true})
				i += end + 4
				continue
			}
		case rest[0] == '`':
			if end := strings.IndexByte(rest[1:], '`'); end > 0 {
				emit(Run{Text: rest[1 : 1+end], Code: true})
				i += end + 2
				continue
			}
		case rest[0] == '[':
			if m, e := link(i); m > i && e < len(text) {
				words, target := text[i+1:m], text[m+2:e]
				kind, ok := linkKind(target)
				link, anchor := target, ""
				if ok && kind == LinkPage {
					ok = links != nil && links.lands != nil
					if ok {
						link, anchor, ok = links.lands(target)
					}
				}
				switch {
				case ok:
					emit(Run{Text: words, Link: link, Anchor: anchor, Kind: kind})
				case links != nil && links.refused != nil && strings.TrimSpace(target) != "":
					links.refused(target)
					plain.WriteString(words)
				default:
					plain.WriteString(words)
				}
				i = e + 1
				continue
			}
		}
		plain.WriteByte(text[i])
		i++
	}
	if plain.Len() > 0 {
		r := Run{Text: plain.String()}
		links.run(r)
		out = append(out, r)
	}
	return out
}
