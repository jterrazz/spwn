package verdict

import "strconv"

type Tone string

const (
	OK    Tone = "ok"
	Warn  Tone = "warn"
	Bad   Tone = "bad"
	Plain Tone = "plain"
	New   Tone = "new"
)

func (Tone) Enum() []string {
	return []string{string(OK), string(Warn), string(Bad), string(Plain), string(New)}
}

type Verdict struct {
	Tone     Tone
	Sentence string
	// value: the sentence a page draws under it; empty for none.
	Line string `json:",omitempty"`
}

type Shelf struct {
	// value: the head of a catalog shelf: what the owner's machines carry of
	// it, and what the catalog holds of it, each its own sentence.
	Mine    Verdict
	Catalog Verdict
}

func Counted(n int, noun, where string) Verdict {
	// invariant: a plain verdict that counts, in the singular at one.
	if n == 1 {
		return Verdict{Tone: Plain, Sentence: "1 " + noun + " is " + where}
	}
	return Verdict{Tone: Plain, Sentence: strconv.Itoa(n) + " " + noun + "s are " + where}
}
