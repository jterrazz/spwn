//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestInspect_ShowsDetails(t *testing.T) {
	// Given - a world spawned with the default config
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	// When - inspecting the world
	// Then - it should report the default config
	chain.Inspect().ExpectConfig("default")
}
