package activity

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/page"
)

func hot(kind string) View {
	at := time.Unix(1_800_000_000, 0)
	s := activity.Sample{At: at, CPUs: 10, MemTotal: 16 << 30, Boot: at.Add(-time.Hour), Thermal: activity.ThermalSerious, Battery: &activity.Battery{Percent: 40, OnBattery: true}}
	return View{Name: "macos-1", Label: "macos-1", Kind: kind, Mac: "mac-mini", On: "mac-mini", Report: activity.Report{Source: activity.FromEngine, Last: activity.Reading{Sample: s}, Window: 10 * time.Second}}
}

func lines(p page.Page) map[string]bool {
	out := map[string]bool{}
	for _, s := range p.Sections {
		for _, it := range s.Items {
			if it.Row != nil {
				out[it.Row.ID] = true
			}
		}
	}
	return out
}

func TestAMacOSWorkspaceShowsNoThermalNorBatteryOfItsOwn(t *testing.T) {
	vm := hot(workspace)
	if got := lines(Page(vm, page.Context{})); got["thermal"] || got["battery"] {
		t.Errorf("Page(macOS workspace) lines = %v; want no Thermal and no Battery: a VM's are its Mac's", got)
	}
	if b := Contract(vm, time.Unix(1_800_000_000, 0)); b.Thermal != nil || b.Battery != nil {
		t.Errorf("Contract(macOS workspace) thermal = %+v, battery = %+v; want both null", b.Thermal, b.Battery)
	}
	here := hot(computer)
	here.ThisMac = true
	if got := lines(Page(here, page.Context{})); !got["thermal"] || !got["battery"] {
		t.Errorf("Page(this Mac) lines = %v; want its Thermal and Battery", got)
	}
}
