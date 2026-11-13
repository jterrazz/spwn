package activity

import (
	"testing"
	"time"
)

var noon = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

// value: points every gap from the window's start, each measuring the
// CPU share busy gives it since the point before.
func busyPoints(from time.Time, gap time.Duration, busy ...float64) []Sample {
	points := make([]Sample, 1, 1+len(busy))
	points[0] = Sample{At: from}
	for i, b := range busy {
		last := points[len(points)-1]
		points = append(points, Sample{At: from.Add(time.Duration(i+1) * gap), CPUBusy: last.CPUBusy + uint64(b*10), CPUTotal: last.CPUTotal + 1000})
	}
	return points
}

func TestTheDayIs48BarsOfHalfAnHourWithTheirMeanAndBusiestMinute(t *testing.T) {
	from := noon.Add(-24 * time.Hour)
	busy := make([]float64, 24*60)
	for i := range busy {
		busy[i] = 10
	}
	busy[15*60+9] = 94
	w := Fold(busyPoints(from, time.Minute, busy...), 24*time.Hour, noon)
	if len(w.Buckets) != 48 || w.Every != 30*time.Minute {
		t.Fatalf("the day folds into %d bars of %v; want 48 of 30 min", len(w.Buckets), w.Every)
	}
	if w.Peak != 30 || w.Buckets[30].Max != 94 || !w.PeakAt.Equal(from.Add(15*time.Hour+10*time.Minute)) {
		t.Errorf("the peak is bar %d at %v (%+v); want bar 30, 94 %% at 03:10", w.Peak, w.PeakAt, w.Buckets[w.Peak])
	}
	if b := w.Buckets[30]; b.Points != 30 || b.Mean < 12.7 || b.Mean > 12.8 {
		t.Errorf("the peak's bar is %+v; want the mean of its 30 minutes, 12.8 %%", b)
	}
	if b := w.Buckets[0]; b.Mean != 10 || b.Max != 10 || b.Points != 29 {
		t.Errorf("the first bar is %+v; want 10 %% from its 29 shares (its first point starts the series)", b)
	}
}

func TestSixMinutesAre72BarsOfFiveSecondsFromTheTwoSecondPoints(t *testing.T) {
	busy := make([]float64, 180)
	for i := range busy {
		busy[i] = float64(i % 50)
	}
	w := Fold(busyPoints(noon.Add(-6*time.Minute), 2*time.Second, busy...), 6*time.Minute, noon)
	if len(w.Buckets) != 72 || w.Every != 5*time.Second {
		t.Fatalf("6 min folds into %d bars of %v; want 72 of 5 s", len(w.Buckets), w.Every)
	}
	for i, b := range w.Buckets {
		if b.Points < 2 || b.Points > 3 {
			t.Errorf("bar %d holds %d points; want 2 or 3 of the 2 s points", i, b.Points)
		}
	}
	if w.Buckets[w.Peak].Max != 49 {
		t.Errorf("the peak is %+v; want a 49 %% point", w.Buckets[w.Peak])
	}
}

func TestAnHourIs60BarsOfAMinute(t *testing.T) {
	if w := Fold(nil, time.Hour, noon); len(w.Buckets) != 60 || w.Every != time.Minute {
		t.Errorf("an hour folds into %d bars of %v; want 60 of a minute", len(w.Buckets), w.Every)
	}
}

func TestAWindowNothingWasMeasuredInIsGapsWithNoPeak(t *testing.T) {
	w := Fold(busyPoints(noon.Add(-48*time.Hour), time.Minute, 50, 50), 24*time.Hour, noon)
	if w.Peak != -1 || !w.PeakAt.IsZero() {
		t.Errorf("a day of no points peaks at bar %d, %v; want no peak", w.Peak, w.PeakAt)
	}
	for _, b := range w.Buckets {
		if b.Points != 0 || b.Max != 0 {
			t.Fatalf("a bar of no points is %+v; want a gap", b)
		}
	}
}
