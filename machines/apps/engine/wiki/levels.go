package wiki

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Detail string

const (
	// value: the four levels of detail one page is read at, the most
	// abstract first, and the face a block, a group or prose takes at one: a
	// block's word, its line, its card, and its full face. A level never
	// hides what a section holds; it changes how much of it shows.
	DetailWord Detail = "word"
	DetailLine Detail = "line"
	DetailCard Detail = "card"
	DetailFull Detail = "full"
)

func Details() []Detail { return []Detail{DetailWord, DetailLine, DetailCard, DetailFull} }

func (Detail) Enum() []string {
	return []string{string(DetailWord), string(DetailLine), string(DetailCard), string(DetailFull)}
}

func (d Detail) rank() int { return slices.Index(Details(), d) }

type ServedReading struct {
	// value: how long one level of the page takes to read: the words it
	// shows, the seconds they take at the reading speed, and the label a
	// reader is shown. Same is true when the level shows exactly what the
	// one before it shows, all at the same faces, so a slider skips it.
	Words   int
	Seconds int
	Label   string
	Same    bool
}

// value: words a minute, the reading speed of every reading time.
const readingSpeed = 230

// invariant: the four levels are read off the three budgets' table of S11: a
// block shows a line at word when 30 s shows it, a card at line when 2 min
// shows it a card or a tile, at card what 10 min shows, a tile its full face,
// and at full a card at least; a level never shows a block less than the
// level before it does, nor more than the block can open.
func blockRanks(t Tier, grouped bool, most Detail) []Detail {
	opened := func(s Shown) bool { return s == ShownLine || s == ShownCard || s == ShownTile }
	f30, f2, f10 := shown(t, Budget30s, grouped), shown(t, Budget2min, grouped), shown(t, Budget10min, grouped)
	r := [4]int{0, 1, 1, 2}
	if opened(f30) {
		r[0] = 1
	}
	if f2 == ShownCard || f2 == ShownTile {
		r[1] = 2
	}
	switch f10 {
	case ShownTile:
		r[2], r[3] = 3, 3
	case ShownCard:
		r[2] = 2
	case ShownLine, ShownCounted, ShownNone:
	}
	return ranked4(r, most)
}

// invariant: prose and a group open no further than a card: prose shows its
// first sentence at word and line and all of itself from card, a group its
// blocks' words at word, its blocks at line, and its lead too from card.
func openRanks() []Detail { return ranked4([4]int{0, 1, 2, 3}, DetailCard) }

func ranked4(r [4]int, most Detail) []Detail {
	out := make([]Detail, 0, len(r))
	reached := 0
	for _, n := range r {
		reached = max(reached, n)
		out = append(out, Details()[min(reached, most.rank())])
	}
	return out
}

// invariant: a prose's first sentence ends at the first ".", "!", "?" or ":"
// that a space or the end of its text follows, outside a code run, unless
// that "." closes an abbreviation (e.g., i.e., cf., vs.) or an initial; its
// paragraphs read as one text, each joined to the next by a space, a table
// line its cells so. More is true when the prose says more than that
// sentence.
func firstSentenceOf(paras []Paragraph) (first []Run, more bool) {
	var all []Run
	sep := func() {
		if len(all) > 0 {
			all = append(all, Run{Text: " "})
		}
	}
	for _, p := range paras {
		if p.Cells != nil {
			for _, c := range p.Cells {
				sep()
				all = append(all, c...)
			}
			continue
		}
		sep()
		all = append(all, p.Runs...)
	}
	whole := strings.TrimSpace(runsText(all))
	for i, r := range all {
		if r.Code {
			continue
		}
		for j := range len(r.Text) {
			if !strings.ContainsRune(".!?:", rune(r.Text[j])) || !spaceAfter(all, i, j) || abbreviation(all, i, j) {
				continue
			}
			first = append(slices.Clone(all[:i]), Run{Text: r.Text[:j+1], Strong: r.Strong, Link: r.Link, Kind: r.Kind})
			return trimRuns(first), len([]rune(strings.TrimSpace(runsText(first)))) < len([]rune(whole))-2
		}
	}
	return trimRuns(all), false
}

func spaceAfter(all []Run, i, j int) bool {
	if j+1 < len(all[i].Text) {
		return unicode.IsSpace(rune(all[i].Text[j+1]))
	}
	for _, r := range all[i+1:] {
		if r.Text != "" {
			return unicode.IsSpace(rune(r.Text[0]))
		}
	}
	return true
}

func abbreviation(all []Run, i, j int) bool {
	if all[i].Text[j] != '.' {
		return false
	}
	var before strings.Builder
	for _, r := range all[:i] {
		before.WriteString(r.Text)
	}
	before.WriteString(all[i].Text[:j+1])
	fields := strings.Fields(before.String())
	word := strings.TrimLeft(fields[len(fields)-1], "([\"'“‘")
	switch strings.ToLower(word) {
	case "e.g.", "i.e.", "cf.", "vs.":
		return true
	}
	initial, _ := utf8.DecodeRuneInString(word)
	return utf8.RuneCountInString(word) == 2 && unicode.IsUpper(initial)
}

func trimRuns(rs []Run) []Run {
	out := slices.Clone(rs)
	for len(out) > 0 && strings.TrimSpace(out[0].Text) == "" {
		out = out[1:]
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1].Text) == "" {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return nil
	}
	out[0].Text = strings.TrimLeftFunc(out[0].Text, unicode.IsSpace)
	out[len(out)-1].Text = strings.TrimRightFunc(out[len(out)-1].Text, unicode.IsSpace)
	return out
}

func runsText(rs []Run) string {
	var b strings.Builder
	for _, r := range rs {
		b.WriteString(r.Text)
	}
	return b.String()
}

func paragraphsText(paras []Paragraph) string {
	parts := make([]string, 0, len(paras))
	for _, p := range paras {
		if p.Cells != nil {
			for _, c := range p.Cells {
				parts = append(parts, runsText(c))
			}
			continue
		}
		parts = append(parts, runsText(p.Runs))
	}
	return strings.Join(parts, " ")
}

func wordCount(texts ...string) int {
	n := 0
	for _, t := range texts {
		n += len(strings.Fields(t))
	}
	return n
}

// invariant: a reading time's label is round: its seconds to the nearest 5 s
// under a minute (5 s at least, and a value of 57.5 s or more is 1 min), to
// the nearest 15 s from a minute to five, and to the nearest minute past
// five; the words only grow from one level to the next, so the labels never
// fall, and two levels may say the same.
func readings(words []int, same []bool) []ServedReading {
	out := make([]ServedReading, len(words))
	for i, w := range words {
		exact := float64(w) * 60 / readingSpeed
		out[i] = ServedReading{Words: w, Seconds: int(math.Round(exact)), Label: labelOf(rounded(exact)), Same: same[i]}
	}
	return out
}

// value: the seconds a reading label shows for a value of seconds: under 60
// the value to the nearest 5 s, 5 s at least; from 60 up to 300 to the
// nearest 15 s; from 300 on to the nearest 60 s. 57.5 and up is 60.
func rounded(seconds float64) int {
	grain := 60.0
	switch {
	case seconds < 60:
		grain = 5
	case seconds < 300:
		grain = 15
	}
	return max(5, int(math.Round(seconds/grain)*grain))
}

func labelOf(seconds int) string {
	if seconds < 60 {
		return "≈ " + strconv.Itoa(seconds) + " s"
	}
	label := "≈ " + strconv.Itoa(seconds/60) + " min"
	if s := seconds % 60; s > 0 {
		label += " " + strconv.Itoa(s) + " s"
	}
	return label
}
