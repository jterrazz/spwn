//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/core/universe"
	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestList_ReturnsSpawnedUniverses(t *testing.T) {
	// GIVEN two spawned universes
	ctx := setup.NewTestContext(t)

	u1 := ctx.Spawn().NoAgent().Execute()
	u2 := ctx.Spawn().NoAgent().Execute()

	// WHEN listing universes
	// THEN both should appear as idle
	u2.List().
		ExpectCount(2).
		ExpectWorld(0, func(e *setup.ListEntryAssertion) {
			e.StatusIs(universe.StatusIdle)
		}).
		ExpectWorld(1, func(e *setup.ListEntryAssertion) {
			e.StatusIs(universe.StatusIdle)
		})

	_ = u1
}
