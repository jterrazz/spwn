package wiki

import "testing"

// value: a label's seconds, from the exact seconds s = words * 60 / 230:
// under 60, round(s / 5) * 5, 5 at least; from 60 under 300, round(s / 15) * 15;
// from 300, round(s / 60) * 60; shown as minutes and seconds.
func TestAReadingLabelIsRoundAtEveryBoundary(t *testing.T) {
	for _, c := range []struct {
		words   int
		seconds int
		label   string
	}{
		{0, 0, "≈ 5 s"},
		{9, 2, "≈ 5 s"},
		{15, 4, "≈ 5 s"},
		{40, 10, "≈ 10 s"},
		{143, 37, "≈ 35 s"},
		{220, 57, "≈ 55 s"},
		{221, 58, "≈ 1 min"},
		{229, 60, "≈ 1 min"},
		{230, 60, "≈ 1 min"},
		{250, 65, "≈ 1 min"},
		{260, 68, "≈ 1 min 15 s"},
		{345, 90, "≈ 1 min 30 s"},
		{1100, 287, "≈ 4 min 45 s"},
		{1140, 297, "≈ 5 min"},
		{1150, 300, "≈ 5 min"},
		{1200, 313, "≈ 5 min"},
		{1400, 365, "≈ 6 min"},
		{1500, 391, "≈ 7 min"},
	} {
		got := readings([]int{c.words}, []bool{false})[0]
		if got.Label != c.label || got.Seconds != c.seconds || got.Words != c.words {
			t.Errorf("%d words: %+v; want %d s labeled %q", c.words, got, c.seconds, c.label)
		}
	}
}

// invariant: the label of a level never falls below the label of the one
// before it when its words do not.
func TestLabelsNeverFallWithTheWords(t *testing.T) {
	var was int
	for w := range 3001 {
		now := rounded(float64(w) * 60 / readingSpeed)
		if now < was {
			t.Fatalf("%d words: %d s after %d s", w, now, was)
		}
		was = now
	}
}
