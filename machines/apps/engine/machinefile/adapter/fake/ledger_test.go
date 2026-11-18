package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestTheLedgerOfAMacKeepsTheContract(t *testing.T) {
	contracttest.Ledger(t, func(*testing.T) contract.Ledger { return fake.LedgerOf(&world.World{}, "laptop") })
}

func TestTheOwnersFolderOfAMacKeepsWhatAPathNames(t *testing.T) {
	contracttest.Owned(t, func(_ *testing.T, path, body string) (contract.Owned, contract.Source) {
		w := &world.World{}
		w.Mac("laptop").Files = map[string]string{}
		if path != "" {
			w.Mac("laptop").Files[path] = body
		}
		return fake.OwnedOf(w, "laptop"), fake.UserOf(w, "laptop")
	})
}
