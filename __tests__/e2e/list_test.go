//go:build e2e

package e2e

import (
	"testing"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
	"github.com/jterrazz/universe/internal/config"
)

func TestList_ReturnsSpawnedUniverses(t *testing.T) {
	ctx := setup.NewTestContext(t)

	u1 := ctx.Spawn().NoAgent().Execute()
	u2 := ctx.Spawn().NoAgent().Execute()

	u2.List().
		ExpectCount(2).
		ExpectUniverse(0, func(e *setup.ListEntryAssertion) {
			e.StatusIs(config.StatusIdle)
		}).
		ExpectUniverse(1, func(e *setup.ListEntryAssertion) {
			e.StatusIs(config.StatusIdle)
		})

	// Suppress unused variable warnings
	_ = u1
}
