package activity

import (
	"encoding/json"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

// value: a server's 30 days at a sample a minute, its last 10 min at one
// every 2 s, busy at 95 % for its last 40 min: every tier full, and the
// busy finding reads back past the window's own buckets.
func month() (*sampler, time.Time) {
	from := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	sp := samplerAt(from)
	busyFrom := from.Add(30*24*time.Hour - 40*time.Minute)
	share := func(at time.Time) float64 {
		if at.After(busyFrom) {
			return 95
		}
		return 20 + float64(at.Minute()%7)
	}
	sp.run(30*24*time.Hour-10*time.Minute, time.Minute, share)
	sp.run(10*time.Minute, WatchedEvery, share)
	return sp, sp.at
}

func fleet(s *Series, machines int) *History {
	h := NewHistory(time.Time{})
	kept := map[string]Kept{}
	for i := range machines {
		kept["m"+strconv.Itoa(i)] = Kept{Series: *s}
	}
	h.Take(kept)
	return h
}

func TestAPartReadsAsTheWholeSeriesForWhatItWasCutFor(t *testing.T) {
	sp, now := month()
	server := machine.Ref{Name: "linux-1"}
	for _, want := range []Want{{Graph: 24 * time.Hour}, {Graph: time.Hour}, {Graph: 6 * time.Minute, History: true}, {}} {
		part := sp.s.Part(want, now)
		data, err := json.Marshal(part)
		if err != nil {
			t.Fatal(err)
		}
		var read Series
		if err := json.Unmarshal(data, &read); err != nil {
			t.Fatal(err)
		}
		for name, got := range map[string]*Series{"cut": &part, "read back": &read} {
			whole := sp.s.Clone()
			if want.Graph > 0 {
				if !reflect.DeepEqual(got.Chart(want.Graph, now), whole.Chart(want.Graph, now)) || !reflect.DeepEqual(got.Measures(want.Graph, now), whole.Measures(want.Graph, now)) || !reflect.DeepEqual(got.Graph(want.Graph, now), whole.Graph(want.Graph, now)) {
					t.Errorf("%+v %s: the window's chart, measures or graph differ from the whole series'", want, name)
				}
			}
			if want.History && !reflect.DeepEqual(got.History(now), whole.History(now)) {
				t.Errorf("%+v %s: the 24 h differ from the whole series'", want, name)
			}
			if r, w := Recent(got, now); r != func() Rates { x, _ := Recent(&whole, now); return x }() || w == 0 {
				t.Errorf("%+v %s: the last 10 s rates %+v over %v differ from the whole series'", want, name, r, w)
			}
			direct := sp.last
			direct.At = now.Add(time.Second)
			direct.CPUTotal += 400
			direct.CPUBusy += 380
			got.Add(direct)
			whole.Add(direct)
			if f, w := Findings(server, machine.Server, got, nil), Findings(server, machine.Server, &whole, nil); !reflect.DeepEqual(f, w) || len(w) == 0 {
				t.Errorf("%+v %s: the findings, a sample added, are %+v; want %+v", want, name, f, w)
			}
		}
	}
}

func TestADaysReadMovesUnder40KBWhateverTheFleet(t *testing.T) {
	sp, now := month()
	want := Want{Graph: 24 * time.Hour}
	sizes := map[int]int{}
	for _, machines := range []int{1, 9} {
		h := fleet(&sp.s, machines)
		k, _ := h.Part("m0", want, now)
		data, err := json.Marshal(k)
		if err != nil {
			t.Fatal(err)
		}
		sizes[machines] = len(data)
	}
	whole, err := json.Marshal(Kept{Series: sp.s})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("a day's read of one machine: %d bytes; its whole series: %d bytes", sizes[1], len(whole))
	if sizes[1] > 40_000 {
		t.Errorf("a day's read of one machine moves %d bytes; want at most 40 KB", sizes[1])
	}
	if sizes[9] != sizes[1] {
		t.Errorf("a day's read moves %d bytes in a fleet of 9, %d alone; want the same", sizes[9], sizes[1])
	}
}

func TestADaysReadCopiesNoWholeSeries(t *testing.T) {
	sp, now := month()
	h := fleet(&sp.s, 9)
	want := Want{Graph: 24 * time.Hour}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	k, _ := h.Part("m0", want, now)
	runtime.ReadMemStats(&after)
	cut := after.TotalAlloc - before.TotalAlloc
	data, err := json.Marshal(k)
	if err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	runtime.ReadMemStats(&before)
	var read Kept
	if err := json.Unmarshal(data, &read); err != nil {
		t.Fatal(err)
	}
	runtime.ReadMemStats(&after)
	back := after.TotalAlloc - before.TotalAlloc
	t.Logf("a day's part: %d bytes allocated to cut it, %d to read it back", cut, back)
	if cut > 100_000 || back > 150_000 {
		t.Errorf("a day's part allocates %d bytes to cut, %d to read back; want no whole series (about 650 KB): at most 100 KB and 150 KB", cut, back)
	}
	runtime.KeepAlive(&read)
}

func BenchmarkADaysReadOfOneMachine(b *testing.B) {
	sp, now := month()
	h := fleet(&sp.s, 9)
	want := Want{Graph: 24 * time.Hour}
	for b.Loop() {
		k, _ := h.Part("m0", want, now)
		data, _ := json.Marshal(k)
		var back Kept
		_ = json.Unmarshal(data, &back)
		back.Series.Chart(want.Graph, now)
		back.Series.Measures(want.Graph, now)
	}
}
