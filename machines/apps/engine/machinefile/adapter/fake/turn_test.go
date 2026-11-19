package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestAMacsTurnKeepsTheContract(t *testing.T) {
	contracttest.Turn(t, func(*testing.T) (contract.Turn, contract.Turn) {
		w := &world.World{}
		return fake.TurnOf(w, "laptop"), fake.TurnOf(w, "laptop")
	})
}
