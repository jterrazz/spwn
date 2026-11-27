package activity

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestATierThatWrapsStartsItsSlotAgainNeverAddsToTheBucketBefore(t *testing.T) {
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	for _, tier := range []int{T1, T2, T3} {
		every, kept := Tiers[tier].Every, Tiers[tier].Kept
		sp := samplerAt(from)
		sp.s.need(tier)
		sp.run(kept+3*every, every, half)
		now := sp.at
		buckets := sp.s.within(tier, from, now)
		if len(buckets) != int(kept/every) {
			t.Errorf("the tier of %v holds %d buckets after wrapping; want its %d slots", every, len(buckets), kept/every)
		}
		for _, b := range buckets {
			if b.n() != 1 || now.UnixMilli()-b.at >= kept.Milliseconds() {
				t.Fatalf("the tier of %v has a bucket at %v of %d samples; want one sample each, none older than %v", every, time.UnixMilli(b.at).UTC(), b.n(), kept)
			}
		}
	}
}

func TestAPartKeepsTheBucketsMemoryPressureAndSwapAreReadFrom(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	sp := samplerAt(from)
	sp.run(time.Hour, IdleEvery, half)
	sp.last.Pressure = PressureCritical
	for range 12 * time.Minute / WatchedEvery {
		sp.last.SwapUsed += 1 << 20
		sp.step(WatchedEvery, 50)
	}
	now := sp.at
	server := machine.Ref{Name: "linux-1"}
	subjects := func(s *Series) []string {
		found := Findings(server, machine.Server, s, nil)
		out := make([]string, 0, len(found))
		for _, f := range found {
			out = append(out, f.Subject)
		}
		return out
	}
	whole := sp.s.Clone()
	part := sp.s.Part(Want{}, now)
	data, err := json.Marshal(part)
	if err != nil {
		t.Fatal(err)
	}
	var read Series
	if err := json.Unmarshal(data, &read); err != nil {
		t.Fatal(err)
	}
	want := subjects(&whole)
	if !slices.Contains(want, "memory") || !slices.Contains(want, "swap") {
		t.Fatalf("the whole series raises %v; want memory pressure and swap growing", want)
	}
	for name, got := range map[string]*Series{"cut": &part, "read back": &read} {
		if s := subjects(got); !slices.Equal(s, want) {
			t.Errorf("%s with no graph nor history: the findings are %v; want the whole series' %v", name, s, want)
		}
	}
}
