package activity_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
)

func TestARateIsTheCountersMovePerSecondAndTheBusyShareOfTheMachine(t *testing.T) {
	at := time.Unix(1_800_000_000, 0)
	a := activity.Sample{At: at, CPUBusy: 100, CPUTotal: 1000, DiskRead: 1e9, DiskWritten: 2e9, NetIn: 5e9, NetOut: 6e9}
	b := activity.Sample{At: at.Add(10 * time.Second), CPUBusy: 440, CPUTotal: 2000, DiskRead: 1.041e9, DiskWritten: 2.013e9, NetIn: 5.021e9, NetOut: 6.0021e9}

	got := activity.Between(a, b)

	if want := (activity.Rates{CPU: 34, DiskRead: 4.1e6, DiskWrite: 1.3e6, NetIn: 2.1e6, NetOut: 2.1e5}); got != want {
		t.Errorf("Between() = %+v; want %+v", got, want)
	}
}

func TestCountersThatGoBackAfterARebootGiveZeroNeverANegativeRate(t *testing.T) {
	at := time.Unix(1_800_000_000, 0)
	before := activity.Sample{At: at, CPUBusy: 9e6, CPUTotal: 2e7, DiskRead: 182e9, DiskWritten: 96e9, NetIn: 38e9, NetOut: 4.6e9}
	rebooted := activity.Sample{At: at.Add(2 * time.Minute), CPUBusy: 300, CPUTotal: 1200, DiskRead: 1e8, DiskWritten: 5e7, NetIn: 2e6, NetOut: 1e6}

	if got := activity.Between(before, rebooted); got != (activity.Rates{}) {
		t.Errorf("Between() across a reboot = %+v; want every rate 0", got)
	}
}

func TestTwoSamplesAtTheSameInstantGiveNoRate(t *testing.T) {
	at := time.Unix(1_800_000_000, 0)
	if got := activity.Between(activity.Sample{At: at}, activity.Sample{At: at, NetIn: 1e9, CPUBusy: 5, CPUTotal: 10}); got != (activity.Rates{}) {
		t.Errorf("Between() of one instant = %+v; want no rate", got)
	}
}

func TestAWatchedEngineRatesTheLast10sOfItsPoints(t *testing.T) {
	at := time.Unix(1_800_000_000, 0)
	var s activity.Series
	for i := range 30 {
		s.Add(activity.Sample{At: at.Add(time.Duration(i) * 2 * time.Second), NetIn: uint64(i) * 2e6})
	}
	newest := at.Add(58 * time.Second)

	rates, window := activity.Recent(&s, newest.Add(time.Second))

	if window != activity.RateWindow || rates.NetIn != 1e6 {
		t.Errorf("Recent() = %v over %v; want 1 MB/s over the last 10 s", rates.NetIn, window)
	}
}

func TestAnUnwatchedEnginesMinutePointsGiveNoRate(t *testing.T) {
	at := time.Unix(1_800_000_000, 0)
	var s activity.Series
	for i := range 6 {
		s.Add(activity.Sample{At: at.Add(time.Duration(i) * time.Minute), NetIn: uint64(i) * 6e7})
	}

	if _, window := activity.Recent(&s, at.Add(5*time.Minute)); window != 0 {
		t.Errorf("Recent() over minute points spans %v; want no rate, so the reader samples now", window)
	}
}

func TestAWatchEndedLongAgoGivesNoRate(t *testing.T) {
	at := time.Unix(1_800_000_000, 0)
	var s activity.Series
	for i := range 10 {
		s.Add(activity.Sample{At: at.Add(time.Duration(i) * 2 * time.Second)})
	}

	if _, window := activity.Recent(&s, at.Add(18*time.Second+46*time.Second)); window != 0 {
		t.Errorf("Recent() of points 46 s old spans %v; want no rate", window)
	}
}
