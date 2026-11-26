package wiki

import "strings"

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
	// of a link, where it points as written, and its Kind.
	Text   string
	Strong bool     `json:",omitempty"`
	Code   bool     `json:",omitempty"`
	Link   string   `json:",omitempty"`
	Kind   LinkKind `json:",omitempty"`
}

type LinkKind string

// value: what a prose link points at: a page of the corpus by its path from
// the page, an anchor of the page, or an address of the web.
const (
	LinkPage     LinkKind = "page"
	LinkExternal LinkKind = "external"
	LinkAnchor   LinkKind = "anchor"
)

// invariant: a link is served only to where the window may follow it: an
// anchor (#id), an http or https address, or a path with no scheme, which
// is a page's. Any other scheme (javascript:, file:, mailto:), a path to a
// host (//host) and a backslash are no link: the run is its words, plain,
// and the check says so (S30).
func linkKind(target string) (LinkKind, bool) {
	t := strings.TrimSpace(target)
	switch {
	case t == "", strings.HasPrefix(t, "//"), strings.ContainsRune(t, '\\'):
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

func Paragraphs(markdown string) []Paragraph { return paragraphs(markdown, nil) }

// invariant: refused, when set, is told each link target no Kind serves,
// in reading order, so the check judges the very links a reader is served.
func paragraphs(markdown string, refused func(target string)) []Paragraph {
	// invariant: prose is read, never rendered as HTML: a paragraph is the
	// lines up to a blank one, a list item opens on "- " or "1. " and takes
	// the indented lines after it, and its inline marks are **strong**,
	// `code` and [words](target); anything else is its text as written.
	var out []Paragraph
	var form ProseForm
	var lines []string
	flush := func() {
		if len(lines) > 0 {
			out = append(out, Paragraph{Form: form, Runs: runs(strings.Join(lines, " "), refused)})
		}
		lines = nil
	}
	for _, raw := range strings.Split(markdown, "\n") {
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
			for _, cell := range strings.Split(strings.Trim(line, "|"), "|") {
				p.Cells = append(p.Cells, runs(strings.TrimSpace(cell), refused))
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

func runs(text string, refused func(target string)) []Run {
	var out []Run
	var plain strings.Builder
	emit := func(s Run) {
		if plain.Len() > 0 {
			out = append(out, Run{Text: plain.String()})
			plain.Reset()
		}
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
	for i := 0; i < len(text); {
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
				switch {
				case ok:
					emit(Run{Text: words, Link: target, Kind: kind})
				case refused != nil && strings.TrimSpace(target) != "":
					refused(target)
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
		out = append(out, Run{Text: plain.String()})
	}
	return out
}
