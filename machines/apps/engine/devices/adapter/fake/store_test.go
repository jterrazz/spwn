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

func TestAProcessThatIsNotTheEngineHoldsNoBookWhileTheEngineRuns(t *testing.T) {
	w := &world.World{}
	w.Mac("macbook-pro").Engine = &world.Engine{}
	engine := fake.StoreOf(w, "macbook-pro")
	engine.Sole(t.Context())
	if _, _, ok := engine.Held(t.Context()); !ok {
		t.Fatal("the engine that took its start read holds no book")
	}
	if _, _, ok := fake.StoreOf(w, "macbook-pro").Held(t.Context()); ok {
		t.Fatal("a process of the same Mac that never took the start read holds the engine's book")
	}
}

func TestAnchor(t *testing.T) {
	contracttest.Anchor(t, func(*testing.T) contract.Anchor { return fake.AnchorOf(&world.World{}, "macbook-pro") })
}
