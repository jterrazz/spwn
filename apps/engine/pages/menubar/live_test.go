package menubar

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func TestAVMsHeldMemoryIsItsProcessesAsReadWithNoSeries(t *testing.T) {
	const gb = int64(1) << 30
	carrying := activity.Reading{VMs: []activity.VMUsage{{VM: "x", Usage: activity.Usage{Memory: 9 * gb}}}}
	if got := LiveValue("memh:x", []activity.Reading{carrying, carrying}); got.Kind != page.KindSize || got.N != 9*gb || len(got.Items) != 0 {
		t.Errorf("memh:x is %+v; want 9 GB as the process holds it, whatever the VM's size, with no series", got)
	}
	if got := LiveValue("memh:x", []activity.Reading{carrying, {}}); got.Kind != "" {
		t.Errorf("memh:x is %+v once the last reading lacks the VM; want none", got)
	}
	if got := LiveValue("memh:x", nil); got.Kind != "" {
		t.Errorf("memh:x is %+v with no reading; want none", got)
	}
}
