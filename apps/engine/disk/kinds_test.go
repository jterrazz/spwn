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
	s := Survey{System: 100, Apps: 50}
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
	s := Survey{System: 100, Apps: 50}
	for _, state := range []HeldState{HeldPending, HeldFailed} {
		got, _ := KindsOf(1000, 500, s, nil, HeldReading{State: state})
		if want := []Kind{KindSystem, KindApps, KindCaches, KindFree}; !slices.Equal(kindsOf(got), want) {
			t.Fatalf("kinds %v while the hypervisor is %d, want %v: your data would hold the machines", kindsOf(got), state, want)
		}
	}
}

func TestAKindWhoseWalkHasNotAnsweredIsAbsentAndYourDataWithIt(t *testing.T) {
	cases := map[string]struct {
		s    Survey
		rows []Row
		want []Kind
	}{
		"system":          {Survey{System: -1, SystemPending: true, Apps: 50}, nil, []Kind{KindApps, KindCaches, KindFree}},
		"apps":            {Survey{System: 100, Apps: -1}, nil, []Kind{KindSystem, KindCaches, KindFree}},
		"caches":          {Survey{System: 100, Apps: 50, CachesUnknown: true}, nil, []Kind{KindSystem, KindApps, KindFree}},
		"a cache unsized": {Survey{System: 100, Apps: 50}, []Row{{App: "go", Group: CanRebuild, Bytes: -1}}, []Kind{KindSystem, KindApps, KindFree}},
	}
	for name, c := range cases {
		got, can := KindsOf(1000, 500, c.s, c.rows, HeldReading{State: HeldNone})
		if !slices.Equal(kindsOf(got), c.want) {
			t.Errorf("%s unknown: kinds %v, want %v: nothing unknown reads 0", name, kindsOf(got), c.want)
		}
		if cachesAbsent := !slices.Contains(c.want, KindCaches); cachesAbsent != (can.Sentence == "") {
			t.Errorf("%s unknown: can free %+v while caches are absent %v: it is absent with them", name, can, cachesAbsent)
		}
	}
	if r := ReadingOf(1000, 500, Survey{System: -1, SystemPending: true}, HeldReading{}, "linux-1"); !r.Pending {
		t.Error("a reading whose system is still walked is not pending")
	}
}

func TestAReadingHasNoStationPartWhileItsLogsAreUnknown(t *testing.T) {
	if r := ReadingOf(1000, 500, Survey{System: -1, Logs: -1}, holding(10), "mbp"); r.Station != nil {
		t.Fatalf("station's part %+v while its logs are not sized, want none", r.Station)
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
	h := Held{
		Machines: []MachineDisk{{Name: "linux-1", Bytes: 50}}, Images: []MachineDisk{{Name: "img", Bytes: 30}},
		Unused: []Freeable{{Holder: HolderImages, Kind: FreeImage, Name: "img@1", Bytes: 30}, {Holder: HolderLeftovers, Kind: FreeDisk, Name: "gone", Bytes: 40}},
	}
	p := StationOf(1000, 600, 20, h, "mbp")
	if p.Holds[2] != (Holding{Holder: HolderLeftovers, Bytes: 40}) || p.Holds[4] != (Holding{Holder: HolderThisMac, Bytes: 260}) || p.Machines[0] != (MachineDisk{Name: "mbp", Bytes: 260}) {
		t.Fatalf("leftovers %+v, the Mac itself %+v, %+v, want 40 and 400 used - 50 - 30 - 40 - 20", p.Holds[2], p.Holds[4], p.Machines[0])
	}
	if p.Itself != "This Mac itself holds 260 B." {
		t.Fatalf("itself %q", p.Itself)
	}
	if p.Verdict.Sentence != "70 B can be freed." || p.CanFree.Bytes != 70 || p.CanFree.Call != FreeUnusedCall || len(p.CanFree.Frees) != 2 {
		t.Fatalf("verdict %q, can free %+v, want the unused image and the leftover, and the gesture that frees both", p.Verdict.Sentence, p.CanFree)
	}
	if p.CanFree.Sentence != "70 B can be freed: 1 unused image and 1 leftover disk." {
		t.Fatalf("can free says %q", p.CanFree.Sentence)
	}
	h.Unread = "~/.local/state/station/vms/x.json"
	if u := StationOf(1000, 600, 20, h, "mbp"); u.CanFree.Call != "" || u.Verdict.Sentence != "A machine record cannot be read." {
		t.Fatalf("a record unread: can free %+v, verdict %q, want no gesture and the reason", u.CanFree, u.Verdict.Sentence)
	}
	none := StationOf(1000, 600, 0, Held{}, "mbp")
	if none.Verdict.Sentence != "Nothing to free." || none.CanFree.Call != "" {
		t.Fatalf("verdict %q, can free %+v, want nothing to free and no gesture", none.Verdict.Sentence, none.CanFree)
	}
}

func TestTheCachesCanFreeNamesItsGestureOnlyWhenThereIsSomething(t *testing.T) {
	if got := canFree(10); got.Call != FreeCall {
		t.Fatalf("caches of 10 B: %+v, want disk.free", got)
	}
	if got := canFree(0); got.Call != "" {
		t.Fatalf("no cache: %+v, want no gesture", got)
	}
}

func TestATestVMIsToldByItsName(t *testing.T) {
	for name, want := range map[string]bool{"station-e2e-macos-1a2b": true, "station-play": true, "station-player": false, "linux-1": false} {
		if TestVM(name) != want {
			t.Errorf("%s: test VM %v, want %v", name, !want, want)
		}
	}
}

func TestAReadingTellsWhatItsVolumeUses(t *testing.T) {
	if r := ReadingOf(1000, 400, Survey{System: -1}, HeldReading{}, "mbp"); r.Used != 600 {
		t.Fatalf("used %d, want 1000 total less 400 free", r.Used)
	}
	if r := ReadingOf(400, 1000, Survey{System: -1}, HeldReading{}, "mbp"); r.Used != 0 {
		t.Fatalf("used %d on a volume whose free passes its total, want never below 0", r.Used)
	}
}

func TestCanFreeCountsWhatItFreesByKind(t *testing.T) {
	for want, frees := range map[string][]Freeable{
		"1 unused image":                                    {{Kind: FreeImage}},
		"2 unused images and 1 old folder":                  {{Kind: FreeImage}, {Kind: FreeFolder}, {Kind: FreeImage}},
		"1 unused image, 2 leftover disks and 1 old folder": {{Kind: FreeDisk}, {Kind: FreeImage}, {Kind: FreeDisk}, {Kind: FreeFolder}},
	} {
		if got := counted(frees); got != want {
			t.Errorf("counted %q, want %q", got, want)
		}
	}
}
