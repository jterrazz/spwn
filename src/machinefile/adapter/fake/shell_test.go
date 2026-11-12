package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/src/machinefile/adapter/fake"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/src/world"
)

func TestTheShellOfTheWorldKeepsTheContract(t *testing.T) {
	contracttest.Shell(t, "mkdir -p ~/src", func(_ *testing.T, fails bool) contract.Shell {
		w := &world.World{}
		if fails {
			w.Script = append(w.Script, world.Fault{Call: "Shell.Run"})
		}
		return fake.ShellOf(w)
	})
}
