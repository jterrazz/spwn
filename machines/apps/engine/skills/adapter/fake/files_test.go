package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/skills/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/skills/contract"
	"github.com/jterrazz/studio-station/apps/engine/skills/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

// value: the layout of a root the world keeps no project at, as the
// project family's fake reads it: act runs. An adapter's test imports no
// other adapter.
type unkept struct{}

func (unkept) Within(_ string, act func() error) error { return act() }

func TestFiles(t *testing.T) {
	contracttest.Files(t, func(t *testing.T) (contract.Files, string) {
		return fake.Of(&world.World{}, "this-mac", unkept{}), t.TempDir()
	})
}
