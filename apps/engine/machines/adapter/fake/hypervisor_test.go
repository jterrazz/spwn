package fake

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestTheFakeHypervisorKeepsTheContract(t *testing.T) {
	contracttest.Hypervisor(t, func(*testing.T) contracttest.Subject {
		w := &world.World{}
		return contracttest.Subject{
			Hypervisor: HypervisorOf(w, "mac-mini", "vz", nil),
			End: func(r contract.Run, how machines.EndReason) {
				end := machines.RunEnd{Reason: how}
				if how == machines.Crashed {
					end.ExitCode = 1
				}
				End(r, end)
			},
			Restart: func() contract.Hypervisor { return HypervisorOf(w, "mac-mini", "vz", nil) },
		}
	})
}
