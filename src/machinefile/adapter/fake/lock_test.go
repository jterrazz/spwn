package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/src/machinefile/adapter/fake"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/src/world"
)

func TestAWorldMacKeepsItsLocks(t *testing.T) {
	contracttest.Locks(t, func(*testing.T) (contract.Locks, func(name, body string)) {
		w := &world.World{ThisMac: "here"}
		put := func(name, body string) {
			w.Mac("here").Files = map[string]string{"machinefiles/" + name + ".lock": body}
		}
		return fake.LocksOf(w, "here"), put
	})
}
