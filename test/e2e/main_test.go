//go:build integration

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// binaryPath is the path to the compiled universe binary, set by TestMain.
var binaryPath string

func TestMain(m *testing.M) {
	// Build binary once for all CLI tests.
	tmpDir, err := os.MkdirTemp("", "universe-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "universe")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/universe")
	// Build from module root.
	cmd.Dir = findModuleRoot()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "building binary: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	// Sweep orphan test containers.
	sweepTestContainers()

	os.Exit(code)
}

// findModuleRoot walks up from the current directory to find go.mod.
func findModuleRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Fallback: assume two levels up from test/e2e.
			wd, _ := os.Getwd()
			return filepath.Join(wd, "..", "..")
		}
		dir = parent
	}
}
