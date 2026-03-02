//go:build e2e

package e2e

import (
	"testing"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
	"github.com/jterrazz/universe/internal/mind"
)

func TestDestroy_RemovesContainer(t *testing.T) {
	setup.NewSpawnBuilder(t).
		NoAgent().
		Execute().
		Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(0)
		}).
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.NotExists()
		})
}

func TestDestroy_AgentSurvives(t *testing.T) {
	ctx := setup.NewTestContext(t)
	ctx.InitAgent("survivor-agent")

	u := ctx.Spawn().
		WithAgent("survivor-agent").
		Execute()

	u.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(0)
		})

	// Agent Mind still exists on host
	info, err := mind.Inspect("survivor-agent")
	if err != nil {
		t.Fatalf("Agent should survive after destroy: %v", err)
	}
	if _, ok := info.Layers["personas"]; !ok {
		t.Fatal("Agent Mind should still have personas layer")
	}
}
