package textblock

import (
	"strconv"
	"strings"
)

func Markers(id string) (start, end string) {
	// value: the marker lines of one block of station's, by its id.
	return "# >>> " + id + " >>>", "# <<< " + id + " <<<"
}

// value: what a text holds of one block: none, one block, or a damage a
// hand cannot be told from (a start with no end, two of one id).

type State int

const (
	Absent State = iota
	Found
	Damaged
)

type Span struct {
	ID         string
	Start, End int
}

func Spans(lines []string) []Span {
	// invariant: a block of station's is found by its two marker lines
	// alone, on whole lines, its id `station:<name>` or `station <name>`;
	// station's main block (`station` alone) and the owner's lines are never
	// part of one. A start met again before its end leaves the first with no
	// end (End -1).
	var out []Span
	for i := 0; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		id, ok := strings.CutPrefix(t, "# >>> ")
		if !ok || !strings.HasSuffix(id, " >>>") {
			continue
		}
		id = strings.TrimSuffix(id, " >>>")
		if rest, ok := strings.CutPrefix(id, "station"); !ok || len(rest) < 2 || rest[0] != ':' && rest[0] != ' ' {
			continue
		}
		start, end := Markers(id)
		s := Span{ID: id, Start: i, End: -1}
		for j := i + 1; j < len(lines); j++ {
			if t := strings.TrimSpace(lines[j]); t == start {
				break
			} else if t == end {
				s.End = j
				break
			}
		}
		out = append(out, s)
		if s.End >= 0 {
			i = s.End
		}
	}
	return out
}

func Of(text, id string) (string, State) {
	// value: the body of one block, without its markers, and whether it is
	// there.
	lines := strings.Split(text, "\n")
	var found []Span
	for _, s := range Spans(lines) {
		if s.ID == id {
			found = append(found, s)
		}
	}
	switch {
	case len(found) == 0:
		return "", Absent
	case len(found) > 1 || found[0].End < 0:
		return "", Damaged
	}
	s := found[0]
	return strings.Join(lines[s.Start+1:s.End], "\n") + "\n", Found
}

func Lines(text, id string) (first, last int) {
	// value: the lines, counted from 1, of the one whole block of id, its
	// markers included; 0, 0 when there is none.
	for _, s := range Spans(strings.Split(text, "\n")) {
		if s.ID == id && s.End >= 0 {
			return s.Start + 1, s.End + 1
		}
	}
	return 0, 0
}

func DamagedLines(text, id string) []string {
	// value: what one hand deletes of a damaged block of id to leave one
	// block, by its lines counted from 1: each start with no end of its own
	// ("line N"), and every whole block after the first ("lines N to M").
	var out []string
	whole := 0
	for _, s := range Spans(strings.Split(text, "\n")) {
		switch {
		case s.ID != id:
		case s.End < 0:
			out = append(out, "line "+strconv.Itoa(s.Start+1))
		default:
			whole++
			if whole > 1 {
				out = append(out, "lines "+strconv.Itoa(s.Start+1)+" to "+strconv.Itoa(s.End+1))
			}
		}
	}
	return out
}

func Remove(text, id string) (updated string, changed bool) {
	// value: text without the block of that id, its marker lines and what
	// is between them, every other byte as it was.
	lines := strings.Split(text, "\n")
	for _, s := range Spans(lines) {
		if s.ID == id && s.End >= 0 {
			return strings.Join(append(append([]string(nil), lines[:s.Start]...), lines[s.End+1:]...), "\n"), true
		}
	}
	return text, false
}
