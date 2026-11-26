package guide_test

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/guide"
)

func statuses(a guide.Act) []guide.Status {
	var out []guide.Status
	for _, s := range guide.Rail(a) {
		out = append(out, s.Status)
	}
	return out
}

func TestARailFoldsTheStepAnActReached(t *testing.T) {
	rail := []guide.Rung{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}, {ID: "c", Label: "C"}}
	for _, tc := range []struct {
		name  string
		act   guide.Act
		wants []guide.Status
	}{
		{"running", guide.Act{Rail: rail, Step: "b"}, []guide.Status{guide.Done, guide.Now, guide.Next}},
		{"failed", guide.Act{Rail: rail, Step: "b", Ended: guide.EndedFailed}, []guide.Status{guide.Done, guide.Failed, guide.Next}},
		{"interrupted", guide.Act{Rail: rail, Step: "c", Ended: guide.EndedInterrupted}, []guide.Status{guide.Done, guide.Done, guide.Failed}},
		{"made", guide.Act{Rail: rail, Step: "b", Ended: guide.EndedDone}, []guide.Status{guide.Done, guide.Done, guide.Done}},
		{"unknown step", guide.Act{Rail: rail, Step: "z"}, []guide.Status{guide.Next, guide.Next, guide.Next}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := statuses(tc.act); !slices.Equal(got, tc.wants) {
				t.Errorf("Rail: %v, want %v", got, tc.wants)
			}
		})
	}
}
