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

func holding(bytes int64) HeldReading {
	return HeldReading{State: HeldAnswered, Held: Held{Machines: []MachineDisk{{Name: "linux-1", Bytes: bytes}}}}
}

func TestAVolumeThatCannotTellItsSystemHasNoSystemAndNoRemainder(t *testing.T) {
	rows := []Row{{App: "go", Group: CanRebuild, Bytes: 40}}
	got, can := KindsOf(1000, 500, Survey{System: -1}, rows, holding(100))
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
	got, _ := KindsOf(1000, 500, s, rows, holding(200))
	if got[2] != (Amount{Kind: KindYourData, Bytes: 110}) {
		t.Fatalf("your data %+v, want 500 used - 100 system - 50 apps - 40 caches - 200 machines", got[2])
	}
	got, _ = KindsOf(1000, 500, s, rows, holding(900))
	if got[2].Bytes != 0 {
		t.Fatalf("your data %d, want 0 when the other kinds hold more than the volume uses", got[2].Bytes)
	}
	got, _ = KindsOf(1000, 500, s, rows, HeldReading{State: HeldNone})
	if slices.Contains(kindsOf(got), KindMachines) || got[2] != (Amount{Kind: KindYourData, Bytes: 310}) {
		t.Fatalf("kinds %+v with no hypervisor, want no machines and images, your data 500 - 100 - 50 - 40", got)
	}
}

func TestAHypervisorThatHasNotAnsweredLeavesMachinesAndYourDataOut(t *testing.T) {
	s := Survey{System: 100, Rows: []Tally{{Install: 50}}}
	for _, state := range []HeldState{HeldPending, HeldFailed} {
		got, _ := KindsOf(1000, 500, s, nil, HeldReading{State: state})
		if want := []Kind{KindSystem, KindApps, KindCaches, KindFree}; !slices.Equal(kindsOf(got), want) {
			t.Fatalf("kinds %v while the hypervisor is %d, want %v: your data would hold the machines", kindsOf(got), state, want)
		}
	}
}

func TestAReadingHasStationsPartOnlyOnceTheHypervisorAnswered(t *testing.T) {
	for state, want := range map[HeldState]bool{HeldNone: false, HeldPending: false, HeldFailed: false, HeldAnswered: true} {
		r := ReadingOf(1000, 500, Survey{System: -1}, HeldReading{State: state}, "mbp")
		if (r.Station != nil) != want || r.Pending != (state == HeldPending) {
			t.Fatalf("hypervisor %d: part %+v, pending %v; want a part %v and pending only while it is asked", state, r.Station, r.Pending, want)
		}
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
