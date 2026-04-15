package base

import "os"

// Default runtime constants.
const (
	DefaultMaxProcs = 128
	DefaultBackend  = "docker"
	WorldImage      = "spwn/world:latest"
)

// Architect daemon constants.
const (
	architectContainerNameBase = "spwn-architect"
	ArchitectImage             = "spwn/architect:latest"
)

// TestArchitectContainerEnv is the env var the test framework sets to
// Scope the architect container name per test run. Mirrors the
// SPWN_TEST_LABEL pattern used for world containers — when it's set,
// Each test run addresses its own architect via a unique container
// Name, so parallel runs never collide on the "spwn-architect"
// Singleton. Unset in production, so behavior is unchanged.
const TestArchitectContainerEnv = "SPWN_TEST_LABEL"

// ArchitectContainerName returns the architect daemon container name,
// Scoped by SPWN_TEST_LABEL when set. Production: "spwn-architect".
// Under a test run: "spwn-architect-<label>".
func ArchitectContainerName() string {
	if label := os.Getenv(TestArchitectContainerEnv); label != "" {
		return architectContainerNameBase + "-" + label
	}
	return architectContainerNameBase
}

// Image versioning constants.
const (
	WorldImageVersion     = "1.1.0"
	ArchitectImageVersion = "1.1.0"
	ImageVersionLabel     = "sh.spwn.image-version"
)

// MindLayers defines the five-layer Mind structure.
var MindLayers = []string{"identity", "skills", "knowledge", "playbooks", "journal"}
