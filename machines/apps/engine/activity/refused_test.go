package activity

import (
	"encoding/json"
	"testing"
	"time"
)

func TestABootThatFlipsASecondIsTheSameBoot(t *testing.T) {
	boot := time.Date(2026, 11, 25, 8, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		name  string
		boot  time.Duration
		ticks uint64
		gap   bool
	}{
		{"its boot a second later", time.Second, 2000, false},
		{"its boot a second earlier", -time.Second, 2000, false},
		{"its boot two seconds later", BootJitter, 2000, false},
		{"booted again, its counters from zero", time.Hour, 10, true},
		{"booted again, its counters still higher", time.Hour, 2000, true},
	} {
		var s Series
		at := boot.Add(time.Hour)
		s.Add(Sample{At: at, CPUTotal: 1000, CPUBusy: 100, Boot: boot})
		s.Add(Sample{At: at.Add(WatchedEvery), CPUTotal: c.ticks, CPUBusy: c.ticks / 10, Boot: boot.Add(c.boot)})
		got := s.within(T0, at, at.Add(WatchedEvery))
		if gap := len(got) == 2 && got[1].span == 0; len(got) != 2 || gap != c.gap {
			t.Errorf("%s: the second sample rolls into %+v; want a gap %v", c.name, got, c.gap)
		}
	}
}

func TestASampleAtOrBefore1970IsRefused(t *testing.T) {
	var s Series
	for _, at := range []time.Time{{}, time.Unix(0, 0), time.Unix(-3600, 0)} {
		s.Add(Sample{At: at, CPUTotal: 10})
	}
	if _, ok := s.Last(); ok {
		t.Error("a sample at or before 1970 was kept; want it refused")
	}
}

func TestARowOutOfRangeIsRefusedOnRead(t *testing.T) {
	for _, row := range []string{
		`[1764072000000,-1,2,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]`,
		`[1764072000000,1024,2,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]`,
		`[1764072000000,1,2,4,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]`,
		`[0,1,2,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]`,
	} {
		var s Series
		doc := `{"recent":[{"at":"2025-11-25T12:00:00Z"}],"tiers":[{"every":1000000000,"buckets":[` + row + `]}]}`
		if err := json.Unmarshal([]byte(doc), &s); err == nil {
			t.Errorf("the row %s read back; want it refused", row)
		}
		if _, ok := s.Last(); ok {
			t.Errorf("the row %s was refused and its series kept; want none", row)
		}
	}
}
