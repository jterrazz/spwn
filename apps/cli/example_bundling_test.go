package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"spwn.sh/examples"
)

// TestBundling_CompiledBinaryShipsExamples builds the real spwn binary
// into a temp directory and invokes `spwn example list` on it.
// This is the one test that proves the production build path actually
// bakes the example templates into the binary — it catches the class
// of bug where the package tests pass (because go test runs against
// the local embed) but a shipped binary is hollow (e.g. because some
// upstream build tool sets CGO_ENABLED=0, strips embeds, or runs from
// the wrong working directory).
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

	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "spwn")

	build := exec.Command("go", "build", "-o", binPath, "./cmd/spwn")
	build.Dir = cliDir
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	// Run `spwn example list` — non-zero exit means empty/broken embed.
	run := exec.Command(binPath, "example", "list")
	run.Dir = repoRoot
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("spwn example list failed: %v\n%s", err, out)
	}
	text := string(out)

	// Every canonical shipped slug MUST appear in the human output.
	want := examples.ShippedSlugs()
	for _, slug := range want {
		if !strings.Contains(text, slug) {
			t.Errorf("compiled binary is missing example %q — rebuild is producing a hollow binary", slug)
		}
	}
}
