package activity_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
)

func watchedFor2MinAfter3hOfMinutes() (*activity.Series, time.Time) {
	var s activity.Series
	at := time.Unix(1_800_000_000, 0)
	for range 181 {
		s.Add(activity.Sample{At: at})
		at = at.Add(time.Minute)
	}
	at = at.Add(-time.Minute)
	for range 60 {
		at = at.Add(2 * time.Second)
		s.Add(activity.Sample{At: at})
	}
	return &s, at
}

func spacing(points []activity.Sample) time.Duration {
	return points[len(points)-1].At.Sub(points[len(points)-2].At)
}

func TestTheLast6MinDrawTheWatched2sPoints(t *testing.T) {
	s, now := watchedFor2MinAfter3hOfMinutes()

	got := s.Graph(6*time.Minute, now)

	if len(got) != 65 || spacing(got) != 2*time.Second || got[0].At.Before(now.Add(-6*time.Minute)) {
		t.Errorf("Graph(6m) = %d points, the newest %v apart, from %v; want 65 points 2s apart, within 6 min of %v", len(got), spacing(got), got[0].At, now)
	}
}

func TestTheLastHourDrawsItsMinutes(t *testing.T) {
	s, now := watchedFor2MinAfter3hOfMinutes()

	got := s.Graph(time.Hour, now)

	if len(got) != 61 || spacing(got) != time.Minute {
		t.Errorf("Graph(1h) = %d points, the newest %v apart; want 61 points a minute apart", len(got), spacing(got))
	}
}

func TestTheLast24hDrawEveryMinuteKept(t *testing.T) {
	s, now := watchedFor2MinAfter3hOfMinutes()

	if got := s.Graph(24*time.Hour, now); len(got) != 183 {
		t.Errorf("Graph(24h) = %d points; want the 183 minutes kept", len(got))
	}
}

func TestAnUnwatchedMacsLast6MinDrawTheirMinutes(t *testing.T) {
	var s activity.Series
	at := time.Unix(1_800_000_000, 0)
	for range 8 {
		s.Add(activity.Sample{At: at})
		at = at.Add(time.Minute)
	}
	now := at.Add(-time.Minute)

	if got := s.Graph(6*time.Minute, now); len(got) != 7 || spacing(got) != time.Minute {
		t.Errorf("Graph(6m) = %d points, the newest %v apart; want 7 points a minute apart", len(got), spacing(got))
	}
}
