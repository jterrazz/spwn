//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/packages/world"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestList_ReturnsSpawnedWorlds(t *testing.T) {
	// Given - two spawned worlds
	ctx := setup.NewTestContext(t)

	u1 := ctx.Spawn().NoAgent().Execute()
	u2 := ctx.Spawn().NoAgent().Execute()

	// When - listing worlds
	// Then - both should appear as idle
	u2.List().
		ExpectCount(2).
		ExpectWorld(0, func(e *setup.ListEntryAssertion) {
			e.StatusIs(world.StatusRunning)
		}).
		ExpectWorld(1, func(e *setup.ListEntryAssertion) {
			e.StatusIs(world.StatusRunning)
		})

	_ = u1
}
