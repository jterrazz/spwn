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
	// value: the most rows a table keeps and serves, and the most parts a
	// figure drawn from what it shows is served with; past it the rest is
	// counted (S31), so a page of tables over a page of blocks stays linear.
	MaxTableRows = 200
	// value: the most columns a table keeps: the widest table of the
	// fixture corpus has 4, so 24 leaves a legitimate page six times room;
	// past it the rest is dropped at the compile and counted (S31), so a
	// table's cells are at most MaxTableRows times MaxTableColumns.
	MaxTableColumns = 24
	// value: the most entries of one list a field or a cell is served with;
	// past it the rest is counted in the field's More.
	MaxCellRefs = 20
	// value: the most sections a page keeps, its opening included: the
	// fixture corpus's pages hold 8 at most; a heading past it is read as
	// the prose of the last section kept, and one S31 counts them.
	MaxSections = 200
	// value: the most bytes a finding's Says keeps of what it quotes: the
	// fixture corpus's longest says 190; past it Says is cut on a
	// character's boundary and marked Cut, so a finding quoting a hostile
	// key or value weighs this at most.
	MaxSaysBytes = 320
	// value: the most bytes of a page's word and line its head is served
	// with, the one part of wiki.read the meter never cuts: a word and a
	// line of the fixture corpus are under 300; past it each is cut on a
	// character's boundary.
	MaxFaceBytes = 4 << 10
	// value: the most bytes of JSON one answer of wiki.read or wiki.map is
	// built to: past it, the items of a page, or the nodes of a map, are
	// left out from the first that does not fit, and counted.
	MaxAnswerBytes = 4 << 20
)

// value: the caps wiki.yaml is held to before it is judged, each a product
// of the schema and a page the check or a serve would otherwise multiply: a
// list past its cap keeps its first entries, a string its first bytes on a
// character's boundary, and one hard H44 on the schema names the cap. The
// fixture's schema is far under each: its longest face template is 9 bytes,
// its 4 seats hold 5 folders, its longest name is 11 bytes and its floors'
// longest question 43.
const (
	// value: a face template of a kind, its word, line or icon: at least 256
	// bytes, four times the fixture's longest at least.
	MaxTemplateBytes = 256
	MaxSeats         = 64
	MaxFields        = 64
	MaxIcons         = 512
	// value: the folders of all seats together, and the folders floor_of
	// names, each.
	MaxPaths = 256
	// value: a floor's name and its question.
	MaxFloorTextBytes = 240
	// value: an id or a name: a floor's, a seat's, a kind's and its icon,
	// state and titles, a field's name, type and target, an enum's value, an
	// icon's, a repository's and a mount's.
	MaxNameBytes = 64
	// value: the values of an enum field.
	MaxValues = 64
	// value: the repositories, and the mounts, each.
	MaxRepos = 64
	// value: a page's id, its file's path less `.md`; a folder or a page
	// wiki.yaml names, and a repository's or a mount's path. A file whose
	// path is longer is served as a page with no content and its one H44.
	MaxPageIDBytes = 240
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
	p.Findings = []Finding{finding("H44", p.ID, "", At{Line: 1, Col: 1}, fmt.Sprintf("%s.md is %d bytes, past the %d a page is compiled from: split it", p.ID, size, MaxPageBytes))}
	return p
}

// value: a page whose path is past MaxPageIDBytes, compiled from nothing
// under its path's first bytes and its pathMark, so two such pages stay two
// and every id is MaxPageIDBytes at most; its one finding says so (H44).
func longPath(file string, schema Schema) Page {
	mark := "~" + pathMark(file)
	kept, _ := clip(file, MaxPageIDBytes-len(mark))
	p := Compile(kept+mark, "", schema)
	p.Findings = []Finding{finding("H44", p.ID, "", At{Line: 1, Col: 1}, fmt.Sprintf("the page's path is %d bytes, past the %d a page id is: shorten it", len(file), MaxPageIDBytes))}
	return p
}

func Past(pages int, bytes int64) bool {
	// value: whether a corpus of these pages and bytes is past a limit.
	return pages > MaxPages || bytes > MaxCorpusBytes
}

type ruleOn struct{ page, rule string }

// value: findings as made, each page's first maxFindings of each rule, then
// one that says how many more there are, where the first of them is; the
// count of each is total's when given, as a checker keeps no finding past
// the first maxFindings+1, else the findings'.
func capped(findings []Finding, total map[ruleOn]int) []Finding {
	if total == nil {
		total = map[ruleOn]int{}
		for _, f := range findings {
			total[ruleOn{f.Page, f.Rule}]++
		}
	}
	seen := map[ruleOn]int{}
	out := make([]Finding, 0, len(findings))
	for _, f := range findings {
		on := ruleOn{f.Page, f.Rule}
		seen[on]++
		if n := seen[on]; n <= maxFindings {
			out = append(out, f)
		} else if n == maxFindings+1 {
			f.Block, f.Cut = "", false
			f.Says = fmt.Sprintf("and %d more like it on this page", total[on]-maxFindings)
			out = append(out, f)
		}
	}
	return out
}
