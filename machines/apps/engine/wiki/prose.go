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
	// of a link and where it points.
	Text   string
	Strong bool   `json:",omitempty"`
	Code   bool   `json:",omitempty"`
	Link   string `json:",omitempty"`
}

func Paragraphs(markdown string) []Paragraph {
	// invariant: prose is read, never rendered as HTML: a paragraph is the
	// lines up to a blank one, a list item opens on "- " or "1. " and takes
	// the indented lines after it, and its inline marks are **strong**,
	// `code` and [words](target); anything else is its text as written.
	var out []Paragraph
	var form ProseForm
	var lines []string
	flush := func() {
		if len(lines) > 0 {
			out = append(out, Paragraph{Form: form, Runs: runs(strings.Join(lines, " "))})
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
				p.Cells = append(p.Cells, runs(strings.TrimSpace(cell)))
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
	// marks dropped: a link keeps its words, code its text.
	var b strings.Builder
	for _, r := range runs(line) {
		b.WriteString(r.Text)
	}
	return b.String()
}

func runs(text string) []Run {
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
				emit(Run{Text: text[i+1 : m], Link: text[m+2 : e]})
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
