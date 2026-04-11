package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"spwn.sh/examples"
)

// TestBundling_CompiledBinaryShipsExamples builds the real spwn binary
// into a temp directory and invokes `spwn example list --json` on it.
// This is the one test that proves the production build path actually
// bakes the example templates into the binary — it catches the class
// of bug where the package tests pass (because go test runs against
// the local embed) but a shipped binary is hollow (e.g. because some
// upstream build tool sets CGO_ENABLED=0, strips embeds, or runs from
// the wrong working directory).
//
// The test is guarded by a build tag in the release workflow (Tauri
// pre-build-script runs it too) so a broken embed cannot silently
// ship a v28.x release again.
func TestBundling_CompiledBinaryShipsExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping binary build test in -short mode")
	}

	// Resolve the module root: apps/cli/../../
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve caller file")
	}
	cliDir := filepath.Dir(thisFile)
	repoRoot := filepath.Join(cliDir, "..", "..")

	// Build the binary fresh into a tempdir. Uses the same flags as
	// the Makefile `make build` target.
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "spwn")

	build := exec.Command("go", "build", "-o", binPath, "./cmd/spwn")
	build.Dir = cliDir
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	// Run `spwn example list --json` and decode.
	run := exec.Command(binPath, "example", "list", "--json")
	run.Dir = repoRoot
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("spwn example list --json failed: %v\n%s", err, out)
	}

	var payload struct {
		Examples []struct {
			Slug string `json:"slug"`
		} `json:"examples"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, out)
	}

	// Build a set of slugs the binary reports.
	got := make(map[string]bool, len(payload.Examples))
	for _, ex := range payload.Examples {
		got[ex.Slug] = true
	}

	// Every canonical shipped slug MUST appear.
	want := examples.ShippedSlugs()
	for _, slug := range want {
		if !got[slug] {
			t.Errorf("compiled binary is missing example %q — rebuild is producing a hollow binary", slug)
		}
	}
	if len(payload.Examples) != len(want) {
		t.Errorf("compiled binary lists %d examples, want %d (%v)", len(payload.Examples), len(want), want)
	}
}
