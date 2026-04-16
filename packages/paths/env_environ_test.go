package paths

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

// TestEnsureDockerFriendlyPATH_PrependsMissing verifies the happy
// path: a sanitized Finder-style PATH gets the Homebrew + Docker
// Desktop locations prepended.
func TestEnsureDockerFriendlyPATH_PrependsMissing(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("no-op on %s", runtime.GOOS)
	}

	t.Setenv("PATH", "/usr/bin:/bin:/usr/sbin:/sbin")
	changed := EnsureDockerFriendlyPATH()
	if !changed {
		t.Fatal("expected PATH to be modified")
	}

	got := os.Getenv("PATH")
	// The original entries must still be present.
	for _, orig := range []string{"/usr/bin", "/bin", "/usr/sbin", "/sbin"} {
		if !strings.Contains(got, orig) {
			t.Errorf("lost original entry %q in augmented PATH %q", orig, got)
		}
	}
	// At least one platform-specific docker path must appear.
	wantAny := dockerFriendlyPaths()
	if len(wantAny) == 0 {
		t.Fatal("dockerFriendlyPaths empty on supported OS")
	}
	found := false
	for _, w := range wantAny {
		if strings.Contains(got, w) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no docker-friendly path made it into %q", got)
	}
}

// TestEnsureDockerFriendlyPATH_IsIdempotent makes sure a second call
// after all locations are already present is a no-op (no duplicate
// entries, no churn).
func TestEnsureDockerFriendlyPATH_IsIdempotent(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("no-op on %s", runtime.GOOS)
	}

	t.Setenv("PATH", "/usr/bin")
	EnsureDockerFriendlyPATH()

	firstCall := os.Getenv("PATH")
	if changed := EnsureDockerFriendlyPATH(); changed {
		t.Error("second call reported a change (should be no-op)")
	}
	secondCall := os.Getenv("PATH")
	if firstCall != secondCall {
		t.Errorf("second call mutated PATH\nbefore: %s\nafter:  %s", firstCall, secondCall)
	}

	// No duplicate entries.
	seen := make(map[string]int)
	for _, p := range strings.Split(secondCall, string(os.PathListSeparator)) {
		seen[p]++
	}
	for p, n := range seen {
		if n > 1 && p != "" {
			t.Errorf("duplicate entry %q appears %d times", p, n)
		}
	}
}

// TestEnsureDockerFriendlyPATH_PreservesUserPrecedence checks that
// locations already present in PATH are NOT re-prepended, so the
// user's own ordering wins.
func TestEnsureDockerFriendlyPATH_PreservesUserPrecedence(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("no-op on %s", runtime.GOOS)
	}

	// Put one of the docker-friendly paths at the END of the existing
	// PATH - we expect it to STAY there, not get duplicated at the front.
	extras := dockerFriendlyPaths()
	if len(extras) == 0 {
		t.Skip("no docker-friendly paths on this OS")
	}
	alreadyThere := extras[0]
	t.Setenv("PATH", "/usr/bin:"+alreadyThere)

	EnsureDockerFriendlyPATH()
	got := os.Getenv("PATH")

	// alreadyThere must appear exactly once.
	count := 0
	for _, p := range strings.Split(got, string(os.PathListSeparator)) {
		if p == alreadyThere {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected %q to appear exactly once, got %d: %q", alreadyThere, count, got)
	}
}
