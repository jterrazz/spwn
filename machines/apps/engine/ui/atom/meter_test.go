package atom

import "testing"

func TestGraphColumnIsTheMeanOfItsPoints(t *testing.T) {
	// value: SPEC §6.4: four points on two columns draw each column at the
	// mean of its two points, not at its last one.
	if got, want := Graph([]int{0, 100, 100, 0}, 2), Spark([]int{50, 50}); got != want {
		t.Fatalf("Graph = %q, want %q", got, want)
	}
}
