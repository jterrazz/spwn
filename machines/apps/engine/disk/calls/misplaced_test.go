package calls

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/disk/adapter/fake"
)

func misplacedOn(t *testing.T, files map[string]int64) []disk.Row {
	t.Helper()
	usage := fake.NewUsage(files, 1000, 500)
	apps := []catalog.App{{
		Name:     "hyper",
		Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.CaskInstall{Token: "hyper"}}},
		Disk: catalog.Disk{
			Paths:     []catalog.Stored{{Path: "~/hyper-data", Level: catalog.LevelData}},
			Misplaced: []catalog.Misplaced{{Path: "~/.hyper", Move: []catalog.Move{{To: "~/.local/share/station/vms"}}}},
		},
	}}
	s := NewSurveyor(onMachine{Usage: usage}, apps, SurveyorOptions{Disk: withSystem{usage}, ThisMachine: catalog.Mac, Expand: disk.Expander{Home: "/h"}})
	return last(t, s).Of(disk.Misplaced)
}

func TestAMisplacedFolderThatHoldsNothingIsNoRow(t *testing.T) {
	if got := misplacedOn(t, map[string]int64{"/h/hyper-data/f": 10, "/h/.hyper/cache/.keep": 0, "/h/.hyper/tmp/.keep": 0}); len(got) != 0 {
		t.Fatalf("an empty ~/.hyper reads %+v, want no misplaced row: there is nothing to move", got)
	}
	if got := misplacedOn(t, map[string]int64{"/h/hyper-data/f": 10, "/h/.hyper/cache/blob": 30}); len(got) != 1 || got[0].Bytes != 30 {
		t.Fatalf("a ~/.hyper holding 30 bytes reads %+v, want its misplaced row of 30", got)
	}
}
