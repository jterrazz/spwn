package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/devices/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestStore(t *testing.T) {
	contracttest.Store(t, func(*testing.T) contract.Store { return fake.StoreOf(&world.World{}, "macbook-pro") })
}

func TestAnchor(t *testing.T) {
	contracttest.Anchor(t, func(*testing.T) contract.Anchor { return fake.AnchorOf(&world.World{}, "macbook-pro") })
}
