package calls

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/disk/adapter/fake"
)

type slowInventory struct {
	release chan struct{}
}

func (i slowInventory) Installed(ctx context.Context, each func(map[string]bool)) {
	each(map[string]bool{"quick": true})
	select {
	case <-i.release:
		each(map[string]bool{"quick": true, "late": true})
	case <-ctx.Done():
	}
}

func TestASizedRowShowsItsSizeBeforeASlowInventoryAnswers(t *testing.T) {
	usage := fake.NewUsage(map[string]int64{"/h/quick/tree/f": 3, "/h/late/tree/f": 5}, 1<<40, 1<<39)
	apps := []catalog.App{
		{Name: "quick", Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.CaskInstall{Token: "quick"}}}, Disk: catalog.Disk{Paths: []catalog.Stored{{Path: "~/quick/*", Level: catalog.LevelCache}}}},
		{Name: "late", Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.CaskInstall{Token: "late"}}}, Disk: catalog.Disk{Paths: []catalog.Stored{{Path: "~/late/*", Level: catalog.LevelCache}}}},
	}
	inventory := slowInventory{release: make(chan struct{})}
	s := NewSurveyor(onMachine{Usage: usage}, apps, SurveyorOptions{Disk: usage, ThisMachine: catalog.Mac, Expand: disk.Expander{Home: "/h"}, Inventory: inventory})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	reports := s.Watch(ctx)
	var rows map[string]disk.Tally
	for r := range reports {
		rows = map[string]disk.Tally{}
		for _, row := range r.Rows {
			rows[row.Entry.Name] = row
		}
		if quick, late := rows["quick"], rows["late"]; !quick.Pending && !late.Pending && quick.State == catalog.StateOK {
			break
		}
	}
	if ctx.Err() != nil {
		t.Fatalf("no row was sized while the inventory had not answered: %+v", rows)
	}
	if quick := rows["quick"]; quick.Bytes[catalog.LevelCache] != 3 {
		t.Errorf("the row whose tool answered is %+v, want its 3 bytes and its mark", quick)
	}
	if late := rows["late"]; late.Bytes[catalog.LevelCache] != 5 || late.State != catalog.StatePending {
		t.Errorf("the row whose tool has not answered is %+v, want its 5 bytes and a pending mark", late)
	}

	close(inventory.release)
	var last disk.Survey
	for r := range reports {
		last = r
	}
	for _, row := range last.Rows {
		if row.Entry.Name == "late" && row.State != catalog.StateOK {
			t.Errorf("once the inventory answered the late row is %+v, want ok", row)
		}
	}
}
