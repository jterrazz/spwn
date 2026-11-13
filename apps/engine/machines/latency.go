package machines

import (
	"slices"
	"strconv"
	"time"
)

const (
	// value: SPEC B5.4: the check runs a server's engine keeps, one a minute
	// for an hour.
	LatencyPoints = 60
	// value: SPEC B5.11: a restart walks the four health steps.
	RestartSteps = 4
)

type LatencyPoint struct {
	At   time.Time     `json:"at"`
	Took time.Duration `json:"took"`
}

func KeepLatency(points []LatencyPoint, p LatencyPoint) []LatencyPoint {
	// invariant: each check that passed adds its run, the oldest beyond 60 dropped.
	points = append(slices.Clone(points), p)
	if n := len(points); n > LatencyPoints {
		points = points[n-LatencyPoints:]
	}
	return points
}

type Latency struct {
	Now, P50, Peak time.Duration
	PeakAt         time.Time
}

func LatencyOf(points []LatencyPoint) (Latency, bool) {
	// invariant: the card's latency is the last run, the median of the runs
	// kept (the lower of the two middle ones), and the slowest with when it ran.
	if len(points) == 0 {
		return Latency{}, false
	}
	l := Latency{Now: points[len(points)-1].Took}
	sorted := make([]time.Duration, 0, len(points))
	for _, p := range points {
		sorted = append(sorted, p.Took)
		if p.Took > l.Peak {
			l.Peak, l.PeakAt = p.Took, p.At
		}
	}
	slices.Sort(sorted)
	l.P50 = sorted[(len(sorted)-1)/2]
	return l, true
}

type RestartWalk struct {
	// value: the step running, 1 to 4, and when it began.
	Step int       `json:"step"`
	At   time.Time `json:"at"`
}

func WalkStep(running, address, ssh, checked bool) int {
	// invariant: the step a restart runs is the first of Running, Has an
	// address, SSH answers and Check passes not passed yet; 0 once all
	// passed. A server with no check passes its fourth with its third.
	for i, passed := range []bool{running, address, ssh, checked} {
		if !passed {
			return i + 1
		}
	}
	return 0
}

func (w RestartWalk) Name() string {
	return [RestartSteps]string{StepRunning, StepAddress, StepSSH, StepCheck}[min(max(w.Step, 1), RestartSteps)-1]
}

func (w RestartWalk) Counted() string {
	return strconv.Itoa(w.Step) + " of " + strconv.Itoa(RestartSteps)
}
