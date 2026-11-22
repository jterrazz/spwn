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
	if want := []Kind{KindApps, KindCaches, KindMachineDisks, KindImages, KindOtherVMs, KindLeftovers, KindLogs, KindFree}; !slices.Equal(kindsOf(got), want) {
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
	if slices.Contains(kindsOf(got), KindMachineDisks) || got[2] != (Amount{Kind: KindYourData, Bytes: 310}) {
		t.Fatalf("kinds %+v with no hypervisor, want no machines and images, your data 500 - 100 - 50 - 40", got)
	}
}

func TestAHypervisorThatHasNotAnsweredLeavesMachinesAndYourDataOut(t *testing.T) {
	s := Survey{System: 100, Apps: 50}
	for _, state := range []HeldState{HeldPending, HeldFailed} {
		got, _ := KindsOf(1000, 500, s, nil, HeldReading{State: state})
		if want := []Kind{KindSystem, KindApps, KindCaches, KindLogs, KindFree}; !slices.Equal(kindsOf(got), want) {
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
		"system":          {Survey{System: -1, SystemPending: true, Apps: 50}, nil, []Kind{KindApps, KindCaches, KindLogs, KindFree}},
		"apps":            {Survey{System: 100, Apps: -1}, nil, []Kind{KindSystem, KindCaches, KindLogs, KindFree}},
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
		Unused: []Freeable{{Holder: KindImages, Kind: FreeImage, Name: "img@1", Bytes: 30}, {Holder: KindLeftovers, Kind: FreeDisk, Name: "gone", Bytes: 40}},
	}
	p := StationOf(1000, 600, 20, 0, h, "mbp")
	if p.Machines[0] != (MachineDisk{Name: "mbp", Bytes: 260}) {
		t.Fatalf("the Mac itself %+v, want 400 used - 50 - 30 - 40 - 20", p.Machines[0])
	}
	if p.Sentence != "This Mac itself holds 260 B. Outside any machine, Machines holds 140 B." {
		t.Fatalf("station's part says %q", p.Sentence)
	}
	if p.Verdict.Sentence != "70 B can be freed." || p.CanFree.Bytes != 70 || p.CanFree.Call != FreeUnusedCall || len(p.CanFree.Frees) != 2 {
		t.Fatalf("verdict %q, can free %+v, want the unused image and the leftover, and the gesture that frees both", p.Verdict.Sentence, p.CanFree)
	}
	if p.CanFree.Sentence != "70 B can be freed: 1 unused image and 1 leftover disk." {
		t.Fatalf("can free says %q", p.CanFree.Sentence)
	}
	h.Unread = "~/.local/state/station/vms/x.json"
	if u := StationOf(1000, 600, 20, 0, h, "mbp"); u.CanFree.Call != "" || u.Verdict.Sentence != "A machine record cannot be read." {
		t.Fatalf("a record unread: can free %+v, verdict %q, want no gesture and the reason", u.CanFree, u.Verdict.Sentence)
	}
	none := StationOf(1000, 600, 0, 0, Held{}, "mbp")
	if none.Verdict.Sentence != "Nothing to free." || none.CanFree.Call != "" {
		t.Fatalf("verdict %q, can free %+v, want nothing to free and no gesture", none.Verdict.Sentence, none.CanFree)
	}
}

func TestTheMacItselfIsTheSumOfItsOwnKindsAndMachinesTheRest(t *testing.T) {
	h := Held{
		Machines: []MachineDisk{{Name: "linux-1", Bytes: 50}}, Images: []MachineDisk{{Name: "img", Bytes: 30}},
		Others: []MachineDisk{{Name: "by-hand", Bytes: 10}},
		Unused: []Freeable{{Holder: KindLeftovers, Kind: FreeDisk, Name: "gone", Bytes: 40}},
	}
	s := Survey{System: 5, Apps: 7, Logs: 20, Hypervisor: "hv"}
	rows := []Row{{App: StationName, Group: CanRebuild, Bytes: 20}, {App: "go", Group: CanRebuild, Bytes: 25}, {App: "hv", Group: CanRebuild, Bytes: 113}}
	got, can := KindsOf(1000, 600, s, rows, HeldReading{State: HeldAnswered, Held: h})
	want := []Amount{{KindSystem, 5}, {KindApps, 7}, {KindYourData, 100}, {KindCaches, 25}, {KindMachineDisks, 50}, {KindImages, 30}, {KindOtherVMs, 10}, {KindLeftovers, 40}, {KindImageCache, 113}, {KindLogs, 20}, {KindFree, 600}}
	if !slices.Equal(got, want) {
		t.Fatalf("kinds %+v, want %+v", got, want)
	}
	if can.Bytes != 45 {
		t.Fatalf("caches can free %d, want the 25 of go and the 20 of Machines' logs, which a bare free empties", can.Bytes)
	}
	p := StationOf(1000, 600, s.Logs, 113, h, "mbp")
	if p.Machines[0].Bytes != 5+7+100+25 {
		t.Fatalf("the Mac itself %d, want its own kinds' %d: one reading, never two sums", p.Machines[0].Bytes, 5+7+100+25)
	}
	if p.Sentence != "This Mac itself holds 137 B. Outside any machine, Machines holds 263 B." {
		t.Fatalf("station's part says %q", p.Sentence)
	}
}

func TestEveryDiskInMachinesHomeIsMachinesWhoeverMadeIt(t *testing.T) {
	h := Held{
		Machines: []MachineDisk{{Name: "development-linux", Bytes: 116}},
		Images:   []MachineDisk{{Name: "station-image-station-linux-2026.11.17", Bytes: 5}},
		Others:   []MachineDisk{{Name: "development-macos", Bytes: 116}, {Name: "station-e2e-xcode-e6a1", Bytes: 77}},
	}
	p := StationOf(1000, 300, 0, 0, h, "mbp")
	if p.Machines[0] != (MachineDisk{Name: "mbp", Bytes: 386}) {
		t.Fatalf("the Mac itself %+v, want 700 used - 116 - 5 - 193", p.Machines[0])
	}
	got, _ := KindsOf(1000, 300, Survey{System: 0, Apps: 0}, nil, HeldReading{State: HeldAnswered, Held: h})
	if got[6] != (Amount{Kind: KindOtherVMs, Bytes: 193}) || got[2] != (Amount{Kind: KindYourData, Bytes: 386}) {
		t.Fatalf("kinds %+v, want other VMs 116 + 77 and your data the 386 left", got)
	}
}

func TestTheImageCacheIsMachinesAndApartFromTheCaches(t *testing.T) {
	s := Survey{System: 0, Apps: 0, Hypervisor: "hv"}
	rows := []Row{{App: "go", Group: CanRebuild, Bytes: 9}, {App: "hv", Group: CanRebuild, Bytes: 113}, {App: "hv", Group: Misplaced, Bytes: 4}}
	got, can := KindsOf(1000, 300, s, rows, holding(200))
	want := []Amount{{KindSystem, 0}, {KindApps, 0}, {KindYourData, 378}, {KindCaches, 9}, {KindMachineDisks, 200}, {KindImages, 0}, {KindOtherVMs, 0}, {KindLeftovers, 0}, {KindImageCache, 113}, {KindLogs, 0}, {KindFree, 300}}
	if !slices.Equal(got, want) {
		t.Fatalf("kinds %+v, want %+v: the image cache is neither a cache to free bare nor your data", got, want)
	}
	if can.Bytes != 9 || can.Call != FreeCall || len(can.Apps) != 0 {
		t.Fatalf("caches can free %+v, want the 9 bytes of go alone, freed bare", can)
	}
	r := ReadingOf(1000, 300, Survey{System: 0, Apps: 0, Hypervisor: "hv", Rows: nil}, holding(200), "mbp")
	if r.ImageCache.Call != "" {
		t.Fatalf("image cache %+v with no row, want nothing to free", r.ImageCache)
	}
	bytes, free := imageCacheOf(s, Classify(Survey{}))
	if bytes != 0 || free.Call != "" {
		t.Fatalf("no row: %d, %+v", bytes, free)
	}
	bytes, free = imageCacheOf(s, rows)
	if bytes != 113 || free.Call != FreeCall || !slices.Equal(free.Apps, []string{"hv"}) {
		t.Fatalf("image cache %d, %+v, want 113 freed by hv's name", bytes, free)
	}
	if free.Sentence != "The image cache holds 113 B: the upstream disks Machines downloaded. The next image built or fetched from one downloads it again." {
		t.Fatalf("image cache says %q", free.Sentence)
	}
	if b, _ := imageCacheOf(s, []Row{{App: "hv", Group: CanRebuild, Bytes: -1}}); b != -1 {
		t.Fatalf("an unsized image cache is %d, want -1", b)
	}
	if b, f := imageCacheOf(Survey{}, rows); b != 0 || f.Call != "" {
		t.Fatalf("no hypervisor: %d, %+v, want no image cache", b, f)
	}
	p := StationOf(1000, 300, 0, 113, Held{Machines: []MachineDisk{{Name: "linux-1", Bytes: 200}}}, "mbp")
	if p.Machines[0] != (MachineDisk{Name: "mbp", Bytes: 387}) {
		t.Fatalf("the Mac itself %+v, want 700 - 200 - 113", p.Machines[0])
	}
}

func TestTheImageCachesRowSaysWhatFreeingItCosts(t *testing.T) {
	if got := noteOf(Survey{Hypervisor: "hv"}, "hv", CanRebuild); got != imageCacheNote {
		t.Fatalf("hv's Can rebuild row says %q", got)
	}
	for _, c := range []struct {
		s   Survey
		app string
		g   Group
	}{{Survey{Hypervisor: "hv"}, "go", CanRebuild}, {Survey{Hypervisor: "hv"}, "hv", Misplaced}, {Survey{}, "hv", CanRebuild}} {
		if got := noteOf(c.s, c.app, c.g); got != "" {
			t.Errorf("%s %s says %q, want nothing", c.app, c.g, got)
		}
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
