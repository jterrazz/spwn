package kernel_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/kernel"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
)

func TestTheKernelsProcessesKeepTheContract(t *testing.T) {
	contracttest.Processes(t, func(t *testing.T) contracttest.ProcessesSubject {
		self, err := kernel.Processes{}.Self()
		if err != nil {
			t.Fatal(err)
		}
		return contracttest.ProcessesSubject{Processes: kernel.Processes{}, Running: self}
	})
}
