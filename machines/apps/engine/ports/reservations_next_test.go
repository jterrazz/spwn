package ports

import (
	"slices"
	"testing"
	"time"
)

func TestTheLifecycleKeepsReservationsAndIdlesOnlyUnpinnedOnes(t *testing.T) {
	t0 := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	st, acts := Next(State{}, EvListen, t0)
	if st.Phase != PhaseShared || !slices.Equal(acts, []Action{ActShare}) {
		t.Fatalf("listen: %+v %v", st, acts)
	}
	st, _ = Next(st, EvUnlisten, t0.Add(time.Second))
	if st, acts = Next(st, EvTick, t0.Add(5*time.Second)); st.Phase != PhaseClosing || acts != nil {
		t.Fatalf("the grace ended early: %+v %v", st, acts)
	}
	if again, more := Next(st, EvListen, t0.Add(5*time.Second)); again.Phase != PhaseShared || more != nil {
		t.Fatalf("a re-listen inside the grace republished: %+v %v", again, more)
	}
	st, acts = Next(st, EvTick, t0.Add(6*time.Second))
	if st.Phase != PhaseWaiting || !slices.Equal(acts, []Action{ActUnpublish}) {
		t.Fatalf("after the grace: %+v %v", st, acts)
	}
	week := t0.Add(7*24*time.Hour + time.Minute)
	if gone, more := Next(st, EvTick, week); gone.Phase != PhaseGone || !slices.Equal(more, []Action{ActRelease}) {
		t.Fatalf("7 idle days: %+v %v", gone, more)
	}
	flagged := State{Phase: PhaseNotAllowed}
	if kept, more := Next(flagged, EvListen, week); kept.Phase != PhaseNotAllowed || more != nil {
		t.Fatalf("a port not allowed was shared by a listen: %+v %v", kept, more)
	}
}

func TestAListenerHeardOnceIdlesOutAfterAnHourWhileOneKeptSeenKeepsItsWeek(t *testing.T) {
	t0 := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	waiting := State{Phase: PhaseWaiting, LastSeenAt: t0}
	hour := t0.Add(time.Hour + time.Minute)
	if gone, acts := Next(State{Phase: waiting.Phase, LastSeenAt: t0, Fleeting: true}, EvTick, hour); gone.Phase != PhaseGone || !slices.Equal(acts, []Action{ActRelease}) {
		t.Fatalf("a fleeting lease after an hour: %+v %v", gone, acts)
	}
	if kept, acts := Next(waiting, EvTick, hour); kept.Phase != PhaseWaiting || acts != nil {
		t.Fatalf("a lease that listened for a while was released after an hour: %+v %v", kept, acts)
	}
}

func TestOnlyALeaseHeardAtOneReportIsFleeting(t *testing.T) {
	t0 := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	for name, c := range map[string]struct {
		l    Reservation
		want bool
	}{
		"made by rights, heard once":         {Reservation{By: ByRights, ReservedAt: t0, LastSeenAt: t0}, true},
		"made by rights, heard again":        {Reservation{By: ByRights, ReservedAt: t0, LastSeenAt: t0.Add(time.Minute)}, false},
		"made for a package's env, unheard":  {Reservation{By: ByRights, ReservedAt: t0}, false},
		"shared by the owner":                {Reservation{By: ByOwner, ReservedAt: t0, LastSeenAt: t0}, false},
		"given an audience by the owner":     {Reservation{By: ByRights, Chosen: "public", ReservedAt: t0, LastSeenAt: t0}, false},
		"made before the by key was written": {Reservation{ReservedAt: t0, LastSeenAt: t0}, false},
	} {
		if got := c.l.Fleeting(); got != c.want {
			t.Errorf("%s: Fleeting = %v, want %v", name, got, c.want)
		}
	}
}
