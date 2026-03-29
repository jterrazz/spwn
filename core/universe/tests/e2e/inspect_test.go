//go:build e2e

package e2e

import (
	"testing"

	"github.com/jterrazz/spwn/core/universe/tests/e2e/setup"
)

func TestInspect_ShowsDetails(t *testing.T) {
	// GIVEN a universe spawned with the default config
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	// WHEN inspecting the universe
	// THEN it should report the default config
	chain.Inspect().ExpectConfig("default")
}
