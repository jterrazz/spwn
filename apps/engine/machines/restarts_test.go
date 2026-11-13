package machines

import (
	"slices"
	"testing"
	"time"
)

func TestAServerWaitsLongerAtEachRestartUntilAMinute(t *testing.T) {
	now := time.Date(2026, 11, 7, 14, 0, 0, 0, time.UTC)
	var r Restarts
	waits := make([]time.Duration, 0, 6)
	for range 6 {
		r = r.After(30*time.Second, now)
		waits = append(waits, r.Wait)
		now = now.Add(3 * time.Minute)
	}
	want := []time.Duration{5 * time.Second, 10 * time.Second, 20 * time.Second, 40 * time.Second, 60 * time.Second, 60 * time.Second}
	for i := range want {
		if waits[i] != want[i] {
			t.Fatalf("waits %v, want %v", waits, want)
		}
	}
	if r.Count != 6 {
		t.Errorf("↺ %d, want 6", r.Count)
	}
}

func TestARunThatStayedUpTenMinutesGoesBackToTheFirstWait(t *testing.T) {
	now := time.Date(2026, 11, 7, 14, 0, 0, 0, time.UTC)
	r := Restarts{}.After(time.Second, now).After(time.Second, now.Add(time.Minute))
	r = r.After(10*time.Minute, now.Add(12*time.Minute))
	if r.Wait != 5*time.Second || r.Count != 3 {
		t.Errorf("after a steady run: wait %v and ↺ %d, want 5s and 3", r.Wait, r.Count)
	}
	if r = (Restarts{}).After(9*time.Minute+59*time.Second, now).After(9*time.Minute+59*time.Second, now.Add(10*time.Minute)); r.Wait != 10*time.Second {
		t.Errorf("a run up just under 10 min kept the backoff at %v, want 10s", r.Wait)
	}
}

func TestTheFifthRestartWithinTenMinutesIsALoopThatWaitsAMinute(t *testing.T) {
	now := time.Date(2026, 11, 7, 14, 0, 0, 0, time.UTC)
	var r Restarts
	for i := range 4 {
		r = r.After(time.Second, now.Add(time.Duration(i)*time.Minute))
		if r.Looping {
			t.Fatalf("restart %d is a loop already", i+1)
		}
	}
	r = r.After(time.Second, now.Add(4*time.Minute))
	if !r.Looping || r.Wait != time.Minute {
		t.Fatalf("the fifth restart: looping %v, wait %v; want a loop waiting 60s", r.Looping, r.Wait)
	}
	if got := r.Words(now.Add(4 * time.Minute)); !slices.Equal(got, []string{"↺ 5 in 10 min", "waiting 60 s"}) {
		t.Errorf("Words = %q", got)
	}
	spread := Restarts{}
	for i := range 5 {
		spread = spread.After(time.Second, now.Add(time.Duration(i)*3*time.Minute))
	}
	if spread.Looping {
		t.Error("five restarts over 12 min are a loop")
	}
	if steady := r.Steady(10*time.Minute, now.Add(15*time.Minute)); steady.Looping {
		t.Error("a loop whose run then stayed up 10 min is still a loop")
	}
}

func TestARebootFromInsideIsCountedWithNoWait(t *testing.T) {
	now := time.Date(2026, 11, 7, 14, 0, 0, 0, time.UTC)
	r := Restarts{}.After(time.Second, now).Rebooted(now.Add(time.Minute))
	if r.Count != 2 || r.Wait != 0 || r.Streak != 1 {
		t.Errorf("after a reboot: ↺ %d, wait %v, streak %d; want 2, 0, 1", r.Count, r.Wait, r.Streak)
	}
	if got := r.Words(now.Add(3*time.Hour + time.Minute)); !slices.Equal(got, []string{"↺ 2", "last 3 h ago"}) {
		t.Errorf("Words = %q", got)
	}
	if got := (Restarts{}).Words(now); !slices.Equal(got, []string{"↺ 0", "none since boot"}) {
		t.Errorf("Words = %q", got)
	}
}
