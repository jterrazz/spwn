package machines

import (
	"testing"
	"time"
)

func TestTheLatencyCardIsTheLastRunTheMedianAndThePeak(t *testing.T) {
	var points []LatencyPoint
	for i, ms := range []int{40, 90, 41, 38, 42} {
		points = KeepLatency(points, LatencyPoint{At: at(5, 10, i), Took: time.Duration(ms) * time.Millisecond})
	}
	l, _ := LatencyOf(points)
	if l.Now != 42*time.Millisecond || l.P50 != 41*time.Millisecond || l.Peak != 90*time.Millisecond || !l.PeakAt.Equal(at(5, 10, 1)) {
		t.Errorf("latency = %+v; want now 42 ms, p50 41 ms, peak 90 ms at 10:01", l)
	}
	for range LatencyPoints {
		points = KeepLatency(points, LatencyPoint{Took: time.Millisecond})
	}
	if len(points) != LatencyPoints {
		t.Errorf("kept %d runs; want the last 60", len(points))
	}
}
