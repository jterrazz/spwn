package activity

import (
	"encoding/json"
	"testing"
	"time"
)

func TestThePeakIsTheTimeOfItsOwnSampleNotOfItsBucket(t *testing.T) {
	from := time.Date(2026, 11, 25, 8, 0, 0, 0, time.UTC)
	peak := from.Add(37 * time.Minute)
	sp := samplerAt(from)
	sp.run(2*time.Hour, time.Minute, func(at time.Time) float64 {
		if at.Equal(peak) {
			return 90
		}
		return 10
	})
	now := sp.at
	for _, window := range []time.Duration{time.Hour * 2, 24 * time.Hour} {
		if got := sp.s.Chart(window, now).PeakAt; !got.Equal(peak) {
			t.Errorf("the %v chart's peak is at %v; want %v, the busiest sample's own time", window, got, peak)
		}
	}
	data, err := json.Marshal(&sp.s)
	if err != nil {
		t.Fatal(err)
	}
	var back Series
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if got := back.Chart(24*time.Hour, now).PeakAt; !got.Equal(peak) {
		t.Errorf("read back, the day's peak is at %v; want %v", got, peak)
	}
}
