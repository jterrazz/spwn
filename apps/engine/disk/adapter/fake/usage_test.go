package fake

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/disk/contract"
	"github.com/jterrazz/studio-station/apps/engine/disk/contract/contracttest"
)

func TestTheFakeDiskUsageKeepsTheContract(t *testing.T) {
	contracttest.DiskUsage(t, func(_ *testing.T, files map[string]int) (contract.DiskUsage, string) {
		on := map[string]int64{}
		for p, n := range files {
			on["/h/"+p] = int64(n)
		}
		return NewUsage(on, 500<<30, 120<<30), "/h"
	})
}
