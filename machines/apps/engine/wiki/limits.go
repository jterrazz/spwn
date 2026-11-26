package wiki

import "fmt"

const (
	// value: the most findings of one rule a page shows; past it one more
	// finding counts the rest, so a hostile page cannot flood a report.
	maxFindings = 100
	// value: the largest page file compiled; a larger one is read for its
	// size only and refused (H44).
	MaxPageBytes = 1 << 20
	// value: the most pages, and the most bytes of all its files, a corpus
	// is read with; a larger corpus is refused whole.
	MaxPages       = 2000
	MaxCorpusBytes = 64 << 20
)

type File struct {
	// value: a file of a corpus as its folder holds it: its size, and its
	// bytes unless it is past MaxPageBytes, when it is never read.
	Size int64
	Body []byte
}

type TooLarge struct {
	// value: what a corpus refused whole holds, counted until it passed a
	// limit: its pages and the bytes of its files.
	Pages int
	Bytes int64
}

func (e *TooLarge) Error() string {
	return fmt.Sprintf("the wiki holds more than %d pages or %d MiB: %d pages and %d bytes read", MaxPages, MaxCorpusBytes>>20, e.Pages, e.Bytes)
}

// value: a page past MaxPageBytes, compiled from nothing: what it holds is
// not read, and its one finding says so (H44).
func tooLong(id string, size int64, schema Schema) Page {
	p := Compile(id, "", schema)
	p.Findings = []Finding{finding("H44", id, "", At{Line: 1, Col: 1}, fmt.Sprintf("%s.md is %d bytes, past the %d a page is compiled from: split it", id, size, MaxPageBytes))}
	return p
}

// value: whether a corpus of these pages and bytes is past a limit.
func Past(pages int, bytes int64) bool { return pages > MaxPages || bytes > MaxCorpusBytes }

type ruleOn struct{ page, rule string }

// value: findings as made, each page's first maxFindings of each rule, then
// one that says how many more there are, where the first of them is.
func capped(findings []Finding) []Finding {
	total := map[ruleOn]int{}
	for _, f := range findings {
		total[ruleOn{f.Page, f.Rule}]++
	}
	seen := map[ruleOn]int{}
	out := make([]Finding, 0, len(findings))
	for _, f := range findings {
		on := ruleOn{f.Page, f.Rule}
		seen[on]++
		if n := seen[on]; n <= maxFindings {
			out = append(out, f)
		} else if n == maxFindings+1 {
			f.Block = ""
			f.Says = fmt.Sprintf("and %d more like it on this page", total[on]-maxFindings)
			out = append(out, f)
		}
	}
	return out
}
