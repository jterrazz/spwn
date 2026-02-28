//go:build integration

package e2e

import (
	"testing"
)

func TestCLICreate(t *testing.T) {
	r := MustRunCLI(t, "create", "--image", TestImage)
	AssertContains(t, r.Stdout, "Universe created:")
	id := ExtractUniverseID(t, r.Stdout)
	CLICleanup(t, id)
}

func TestCLICreateAndList(t *testing.T) {
	r := MustRunCLI(t, "create", "--image", TestImage)
	id := ExtractUniverseID(t, r.Stdout)
	CLICleanup(t, id)

	shortID := id[:8]
	r = MustRunCLI(t, "list")
	AssertContains(t, r.Stdout, shortID)
	AssertContains(t, r.Stdout, TestImage)
}

func TestCLIInspect(t *testing.T) {
	r := MustRunCLI(t, "create", "--image", TestImage)
	id := ExtractUniverseID(t, r.Stdout)
	CLICleanup(t, id)

	r = MustRunCLI(t, "inspect", id)
	AssertContains(t, r.Stdout, id)
	AssertContains(t, r.Stdout, TestImage)
}

func TestCLIDestroy(t *testing.T) {
	r := MustRunCLI(t, "create", "--image", TestImage)
	id := ExtractUniverseID(t, r.Stdout)

	r = MustRunCLI(t, "destroy", id)
	AssertContains(t, r.Stdout, "Universe destroyed:")

	// Verify it's gone from list.
	r = MustRunCLI(t, "list")
	AssertNotContains(t, r.Stdout, id[:8])
}

func TestCLIDestroyNotFound(t *testing.T) {
	r := RunCLI(t, "destroy", "00000000-0000-0000-0000-000000000000")
	if r.Err == nil {
		t.Fatal("destroy with bogus ID should return nonzero exit code")
	}
}

func TestCLICreateWithFlags(t *testing.T) {
	r := MustRunCLI(t, "create", "--image", TestImage, "--mind", "test-cli-mind", "--memory", "256m")
	AssertContains(t, r.Stdout, "Universe created:")
	AssertContains(t, r.Stdout, "test-cli-mind")
	id := ExtractUniverseID(t, r.Stdout)
	CLICleanup(t, id)
}

func TestCLIHelpOutput(t *testing.T) {
	r := MustRunCLI(t, "--help")
	AssertContains(t, r.Stdout, "create")
	AssertContains(t, r.Stdout, "spawn")
	AssertContains(t, r.Stdout, "list")
	AssertContains(t, r.Stdout, "inspect")
	AssertContains(t, r.Stdout, "destroy")
	AssertContains(t, r.Stdout, "mind")
}

func TestCLIMindList(t *testing.T) {
	// Create a universe with a mind to ensure a mind directory exists.
	r := MustRunCLI(t, "create", "--image", TestImage, "--mind", "test-mind-list")
	id := ExtractUniverseID(t, r.Stdout)
	CLICleanup(t, id)

	r = MustRunCLI(t, "mind", "list")
	AssertContains(t, r.Stdout, "test-mind-list")
}

func TestCLIMindInspect(t *testing.T) {
	// Create a universe with a mind to ensure a mind directory exists.
	r := MustRunCLI(t, "create", "--image", TestImage, "--mind", "test-mind-inspect")
	id := ExtractUniverseID(t, r.Stdout)
	CLICleanup(t, id)

	r = MustRunCLI(t, "mind", "inspect", "test-mind-inspect")
	AssertContains(t, r.Stdout, "test-mind-inspect")
	AssertContains(t, r.Stdout, "Structure:")
}
