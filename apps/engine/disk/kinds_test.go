package disk

import (
	"slices"
	"testing"
)

func kindsOf(amounts []Amount) []Kind {
	out := make([]Kind, 0, len(amounts))
	for _, a := range amounts {
		out = append(out, a.Kind)
	}
	return out
}

func TestAVolumeThatCannotTellItsSystemHasNoSystemAndNoRemainder(t *testing.T) {
	rows := []Row{{App: "go", Group: CanRebuild, Bytes: 40}}
	got, can := KindsOf(1000, 500, Survey{System: -1}, rows, 100, true)
	if want := []Kind{KindApps, KindCaches, KindMachines, KindFree}; !slices.Equal(kindsOf(got), want) {
		t.Fatalf("kinds %v, want %v: your data would hold the system", kindsOf(got), want)
	}
	if can.Bytes != 40 {
		t.Fatalf("can free %d, want the 40 bytes of Can rebuild", can.Bytes)
	}
}

func TestYourDataIsWhatNoOtherKindHoldsAndNeverBelowZero(t *testing.T) {
	s := Survey{System: 100, Rows: []Tally{{Install: 50}}}
	rows := []Row{{App: "go", Group: CanRebuild, Bytes: 40}, {App: "xcode", Group: YourData, Bytes: 7}}
	got, _ := KindsOf(1000, 500, s, rows, 200, true)
	if got[2] != (Amount{Kind: KindYourData, Bytes: 110}) {
		t.Fatalf("your data %+v, want 500 used - 100 system - 50 apps - 40 caches - 200 machines", got[2])
	}
	got, _ = KindsOf(1000, 500, s, rows, 900, true)
	if got[2].Bytes != 0 {
		t.Fatalf("your data %d, want 0 when the other kinds hold more than the volume uses", got[2].Bytes)
	}
	got, _ = KindsOf(1000, 500, s, rows, 0, false)
	if slices.Contains(kindsOf(got), KindMachines) {
		t.Fatalf("kinds %v off a Mac, want no machines and images", kindsOf(got))
	}
}

func TestStationPartLeavesTheMacItselfWhatItsPartsDoNotHold(t *testing.T) {
	h := Held{Machines: []MachineDisk{{Name: "linux-1", Bytes: 50}}, Images: []MachineDisk{{Name: "img", Bytes: 30}}, Unused: 30}
	p := StationOf(1000, 600, 20, h, "mbp")
	if p.Holds[4] != (Holding{Holder: HolderThisMac, Bytes: 300}) || p.Machines[0] != (MachineDisk{Name: "mbp", Bytes: 300}) {
		t.Fatalf("the Mac itself %+v, %+v, want 400 used - 50 - 30 - 20", p.Holds[4], p.Machines[0])
	}
	if p.Verdict.Sentence != "30 B can be freed." {
		t.Fatalf("verdict %q, want the unused image's bytes", p.Verdict.Sentence)
	}
	if none := StationOf(1000, 600, 0, Held{}, "mbp"); none.Verdict.Sentence != "Nothing to free." {
		t.Fatalf("verdict %q, want nothing to free", none.Verdict.Sentence)
	}
}
