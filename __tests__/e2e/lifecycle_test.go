//go:build e2e

package e2e

import (
	"testing"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/mind"
)

func TestFullLifecycle_SpawnInspectDestroy(t *testing.T) {
	ctx := setup.NewTestContext(t)
	ctx.InitAgent("e2e-agent")

	// Spawn universe with agent
	u := ctx.Spawn().
		WithAgent("e2e-agent").
		Execute()

	u.ExpectState(func(s *setup.StateAssertion) {
		s.UniverseCount(1)
		s.UniverseStatus(config.StatusIdle)
	})

	u.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/universe/physics.md")
		c.HasFile("/universe/faculties.md")
		c.HasMount("/mind")
	})

	// Inspect
	u.Inspect().ExpectConfig("default")

	// List
	u.List().ExpectCount(1)

	// Destroy
	u.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(0)
		}).
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.NotExists()
		})

	// Agent persists after universe destruction
	info, err := mind.Inspect("e2e-agent")
	if err != nil {
		t.Fatalf("Agent should survive after destroy: %v", err)
	}
	if _, ok := info.Layers["personas"]; !ok {
		t.Fatal("Agent Mind should still have personas layer")
	}
}
