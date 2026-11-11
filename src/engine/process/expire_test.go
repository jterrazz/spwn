package process

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"

	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
)

type expired struct {
	vm    string
	names []string
}

type fakeShares struct {
	mu      sync.Mutex
	held    map[string][]accounts.Share
	fail    error
	expires chan expired
}

func (f *fakeShares) Held() (map[string][]accounts.Share, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string][]accounts.Share{}
	for vm, s := range f.held {
		out[vm] = slices.Clone(s)
	}
	return out, nil
}

func (f *fakeShares) Expire(_ context.Context, vm string, names []string) error {
	f.mu.Lock()
	err := f.fail
	if err == nil {
		f.held[vm] = slices.DeleteFunc(f.held[vm], func(s accounts.Share) bool { return slices.Contains(names, s.Name) })
	}
	f.mu.Unlock()
	f.expires <- expired{vm: vm, names: names}
	return err
}

func (f *fakeShares) Forget(_ context.Context, vm string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.held, vm)
	return nil
}

func (f *fakeShares) failing(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fail = err
}

type toldNotifier struct {
	mu   sync.Mutex
	told []engine.Notification
}

func (n *toldNotifier) Notify(_ context.Context, x engine.Notification) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.told = append(n.told, x)
}

func (n *toldNotifier) count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.told)
}

type sharing struct {
	harness
	shares *fakeShares
	told   *toldNotifier
}

func newSharing(t *testing.T, held map[string][]accounts.Share) sharing {
	t.Helper()
	h := newHarness(t,
		model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning},
		model.VMRecord{Name: "web", Lifetime: model.Persistent, Want: model.WantRunning},
	)
	s := sharing{harness: h, shares: &fakeShares{held: held, expires: make(chan expired, 8)}, told: &toldNotifier{}}
	h.d.o.AccountShares, h.d.notifier = s.shares, s.told
	for name, guest := range map[string]string{"api": engine.GuestConnected, "web": engine.GuestConnecting} {
		v, _ := h.d.find(name)
		v.state, v.guest = engine.StateRunning, guest
	}
	return s
}

// value: sweep runs the engine's periodic expire job once and waits for the
// take-back it starts; a sweep that takes nothing back fails the test.
func (s sharing) sweep(t *testing.T) expired {
	t.Helper()
	s.d.sweepShares(t.Context())
	wait, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case e := <-s.shares.expires:
		s.d.shareMu.Lock()
		defer s.d.shareMu.Unlock()
		return e
	case <-wait.Done():
		t.Fatal("the expire job took nothing back")
		return expired{}
	}
}

func (s sharing) sweepNothing(t *testing.T) {
	t.Helper()
	s.d.sweepShares(t.Context())
	wait, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	select {
	case e := <-s.shares.expires:
		t.Fatalf("the expire job took back %v from %s, want nothing", e.names, e.vm)
	case <-wait.Done():
	}
}

func TestAShareEndedOnAVMThatStaysConnectedIsTakenBackByTheExpireJobWithNoNotification(t *testing.T) {
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	s := newSharing(t, map[string][]accounts.Share{
		"api": {{Name: "session:github", Until: now}, {Name: "operator:claude:oauth-token", Until: now.Add(2 * time.Hour)}},
		"web": {{Name: "session:github", Until: now.Add(-time.Hour)}},
	})
	got := s.sweep(t)
	if got.vm != "api" || !slices.Equal(got.names, []string{"session:github"}) {
		t.Errorf("the expire job took back %v from %s, want session:github from api alone", got.names, got.vm)
	}
	held, _ := s.shares.Held()
	if len(held["api"]) != 1 || held["api"][0].Name != "operator:claude:oauth-token" || len(held["web"]) != 1 {
		t.Errorf("shares left %v, want api's claude still running and web's, still connecting, kept for its connect", held)
	}
	if s.told.count() != 0 {
		t.Errorf("told %v, want no one told of a share that ended", s.told.told)
	}
}

func TestTheExpireJobReadsTheSharesOnceAMinute(t *testing.T) {
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	s := newSharing(t, map[string][]accounts.Share{
		"api": {{Name: "session:github", Until: now}, {Name: "operator:claude:oauth-token", Until: now.Add(20 * time.Second)}},
	})
	s.sweep(t)
	s.clock.At = now.Add(30 * time.Second)
	s.sweepNothing(t)
	s.clock.At = now.Add(time.Minute)
	if got := s.sweep(t); !slices.Equal(got.names, []string{"operator:claude:oauth-token"}) {
		t.Errorf("a minute on, the expire job took back %v, want the claude share that ended meanwhile", got.names)
	}
}

func TestAFailedTakeBackKeepsItsShareAndIsToldOncePerHour(t *testing.T) {
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	s := newSharing(t, map[string][]accounts.Share{"api": {{Name: "session:github", Until: now}}})
	s.shares.failing(errors.New("deleting the copies failed for \"api\": exit 1"))
	steps := []struct {
		after time.Duration
		told  int
	}{{0, 1}, {time.Minute, 1}, {59 * time.Minute, 1}, {time.Hour, 2}, {61 * time.Minute, 2}}
	for _, step := range steps {
		s.clock.At = now.Add(step.after)
		s.sweep(t)
		if got := s.told.count(); got != step.told {
			t.Fatalf("%s after the share ended: told %d times, want %d", step.after, got, step.told)
		}
	}
	n := s.told.told[0]
	if n.Kind != engine.NoticeTakeBackFailed || n.Subject != "api share" || !strings.Contains(n.Text, "taking GitHub back failed: deleting the copies") {
		t.Errorf("told %+v, want the GitHub take-back failure of api", n)
	}
	held, _ := s.shares.Held()
	if len(held["api"]) != 1 {
		t.Errorf("shares left %v, want the share kept while its copy stays in the VM", held)
	}

	s.shares.failing(nil)
	s.clock.At = now.Add(62 * time.Minute)
	s.sweep(t)
	s.shares.mu.Lock()
	s.shares.held["api"] = []accounts.Share{{Name: "session:github", Until: now}}
	s.shares.mu.Unlock()
	s.shares.failing(errors.New("exit 1"))
	s.clock.At = now.Add(63 * time.Minute)
	s.sweep(t)
	if got := s.told.count(); got != 3 {
		t.Errorf("a take-back failing again after one succeeded: told %d times, want it told at once (3)", got)
	}
}
