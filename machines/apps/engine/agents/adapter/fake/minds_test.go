package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

// value: the layout of a root the world keeps no project at, as the
// project family's fake reads it: act runs. An adapter's test imports no
// other adapter.
type unkept struct{}

func (unkept) Within(_ string, act func() error) error { return act() }

func TestMinds(t *testing.T) {
	contracttest.Minds(t, func(t *testing.T) (contract.Minds, string) {
		return fake.Of(&world.World{}, "this-mac", unkept{}), t.TempDir()
	})
}
