package activity

import (
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
)

func TestAPointsMemoryShareIsHeldAt100AfterTheVMsMemoryShrinks(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	var s activity.Series
	for i := range 10 {
		total, used := int64(16<<30), int64(12<<30)
		if i >= 5 {
			total, used = 8<<30, 4<<30
		}
		s.Add(activity.Sample{At: from.Add(time.Duration(i) * time.Minute), CPUTotal: uint64(i) * 60_000, MemTotal: total, MemUsed: used})
	}
	got := series(s.Graph(time.Hour, from.Add(9*time.Minute)))["memory"]
	if want := []int{100, 100, 100, 100, 100, 50, 50, 50, 50, 50}; !slices.Equal(got, want) {
		t.Errorf("the memory line after a 16 GB VM shrank to 8 GB is %v; want %v: the minutes of 12 GB used held at 100 %%", got, want)
	}
}
