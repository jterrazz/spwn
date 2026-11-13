package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestTheRootPasswordOfAMacKeepsTheContract(t *testing.T) {
	contracttest.Password(t, func(_ *testing.T, asks bool) contract.Password {
		w := &world.World{}
		w.Mac("workspace").Passwordless = !asks
		return fake.PasswordOf(w, "workspace")
	})
}
