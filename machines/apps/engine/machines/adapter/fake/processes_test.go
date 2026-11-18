package fake

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestTheFakeProcessesKeepTheContract(t *testing.T) {
	contracttest.Processes(t, func(*testing.T) contracttest.ProcessesSubject {
		running := machines.Process{PID: 4242, Start: "Fri Nov 13 09:00:00 2026"}
		w := &world.World{}
		w.Mac("mac-mini").Processes = []machines.Process{running}
		return contracttest.ProcessesSubject{Processes: ProcessesOf(w, "mac-mini", nil), Running: running}
	})
}
