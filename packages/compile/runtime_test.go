package compile

import (
	"sort"
	"testing"
)

// fakeRuntime is a test-only Runtime that records nothing.
type fakeRuntime struct{ name string }

func (f *fakeRuntime) Name() string                { return f.name }
func (f *fakeRuntime) Render(Input) (*Tree, error) { return New(), nil }

// TestRegisteredRuntimes locks the shape of RegisteredRuntimes:
// every registered name shows up, sorted. Used by the CLI to print
// an honest "known runtimes" hint on typos instead of hardcoding
// "claude-code".
func TestRegisteredRuntimes(t *testing.T) {
	// Snapshot + restore so we don't pollute the package-global
	// registry for parallel tests.
	saved := runtimes
	runtimes = map[string]Runtime{}
	t.Cleanup(func() { runtimes = saved })

	Register(&fakeRuntime{name: "zeta"})
	Register(&fakeRuntime{name: "alpha"})
	Register(&fakeRuntime{name: "mu"})

	got := RegisteredRuntimes()
	want := []string{"alpha", "mu", "zeta"}
	if !sort.StringsAreSorted(got) {
		t.Fatalf("RegisteredRuntimes must be sorted, got %v", got)
	}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

// TestRegisteredRuntimes_empty ensures the helper returns an empty
// slice (not nil) when no runtimes are registered.
func TestRegisteredRuntimes_empty(t *testing.T) {
	saved := runtimes
	runtimes = map[string]Runtime{}
	t.Cleanup(func() { runtimes = saved })

	got := RegisteredRuntimes()
	if got == nil {
		t.Fatal("want non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("want empty, got %v", got)
	}
}
