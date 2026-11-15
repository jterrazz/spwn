package process

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

type retried struct {
	vm       string
	programs []accounts.Program
}

type fakeOwed struct {
	mu      sync.Mutex
	owed    map[string][]accounts.Program
	fail    error
	retries chan retried
}

func (f *fakeOwed) Owed() (map[string][]accounts.Program, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string][]accounts.Program{}
	for vm, p := range f.owed {
		out[vm] = slices.Clone(p)
	}
	return out, nil
}

func (f *fakeOwed) Retry(_ context.Context, vm string, programs []accounts.Program) error {
	f.mu.Lock()
	err := f.fail
	if err == nil {
		delete(f.owed, vm)
	}
	f.mu.Unlock()
	f.retries <- retried{vm: vm, programs: programs}
	return err
}

func (f *fakeOwed) Forget(_ context.Context, vm string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.owed, vm)
	return nil
}

func (f *fakeOwed) failing(err error) {
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

type owing struct {
	harness
	owed *fakeOwed
	told *toldNotifier
}

func newOwing(t *testing.T, owed map[string][]accounts.Program) owing {
	t.Helper()
	h := newHarness(t,
		model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning},
		model.VMRecord{Name: "web", Lifetime: model.Persistent, Want: model.WantRunning},
	)
	s := owing{harness: h, owed: &fakeOwed{owed: owed, retries: make(chan retried, 8)}, told: &toldNotifier{}}
	h.d.o.GrantsOwed, h.d.notifier = s.owed, s.told
	for name, guest := range map[string]string{"api": engine.GuestConnected, "web": engine.GuestConnecting} {
		v, _ := h.d.find(name)
		v.state, v.guest = engine.StateRunning, guest
	}
	return s
}

// value: sweep runs the engine's periodic retry job once and waits for the
// retry it starts; a sweep that retries nothing fails the test.
func (s owing) sweep(t *testing.T) retried {
	t.Helper()
	s.d.sweepOwed(t.Context())
	wait, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case r := <-s.owed.retries:
		s.d.owedMu.Lock()
		defer s.d.owedMu.Unlock()
		return r
	case <-wait.Done():
		t.Fatal("the retry job retried nothing")
		return retried{}
	}
}

func (s owing) sweepNothing(t *testing.T) {
	t.Helper()
	s.d.sweepOwed(t.Context())
	wait, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	select {
	case r := <-s.owed.retries:
		t.Fatalf("the retry job retried %v on %s, want nothing", r.programs, r.vm)
	case <-wait.Done():
	}
}

func TestARemovalOwedToAConnectedVMIsTakenOutByTheRetryJobWithNoNotification(t *testing.T) {
	s := newOwing(t, map[string][]accounts.Program{"api": {"wrangler"}, "web": {"acme"}})
	got := s.sweep(t)
	if got.vm != "api" || !slices.Equal(got.programs, []accounts.Program{"wrangler"}) {
		t.Errorf("the retry job retried %v on %s, want wrangler on api alone", got.programs, got.vm)
	}
	owed, _ := s.owed.Owed()
	if len(owed["api"]) != 0 || len(owed["web"]) != 1 {
		t.Errorf("owed left %v, want web's, still connecting, kept for its connect", owed)
	}
	if s.told.count() != 0 {
		t.Errorf("told %v, want no one told of a removal taken out", s.told.told)
	}
}

func TestTheRetryJobReadsWhatIsOwedOnceAMinute(t *testing.T) {
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	s := newOwing(t, map[string][]accounts.Program{"api": {"wrangler"}})
	s.clock.At = now
	s.sweep(t)
	s.owed.mu.Lock()
	s.owed.owed["api"] = []accounts.Program{"acme"}
	s.owed.mu.Unlock()
	s.clock.At = now.Add(30 * time.Second)
	s.sweepNothing(t)
	s.clock.At = now.Add(time.Minute)
	if got := s.sweep(t); !slices.Equal(got.programs, []accounts.Program{"acme"}) {
		t.Errorf("a minute on, the retry job retried %v, want the acme removal owed meanwhile", got.programs)
	}
}

func TestAFailedRetryStaysOwedAndIsToldOncePerHour(t *testing.T) {
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	s := newOwing(t, map[string][]accounts.Program{"api": {"wrangler"}})
	s.owed.failing(errors.New("exit 1"))
	steps := []struct {
		after time.Duration
		told  int
	}{{0, 1}, {time.Minute, 1}, {59 * time.Minute, 1}, {time.Hour, 2}, {61 * time.Minute, 2}}
	for _, step := range steps {
		s.clock.At = now.Add(step.after)
		s.sweep(t)
		if got := s.told.count(); got != step.told {
			t.Fatalf("%s after the removal failed: told %d times, want %d", step.after, got, step.told)
		}
	}
	n := s.told.told[0]
	if n.Kind != engine.NoticeGrantRemovalFailed || n.Subject != "api grant" || !strings.Contains(n.Text, "removing the grant of Cloudflare failed: exit 1") {
		t.Errorf("told %+v, want the Cloudflare removal failure of api", n)
	}
	owed, _ := s.owed.Owed()
	if len(owed["api"]) != 1 {
		t.Errorf("owed left %v, want the removal still owed", owed)
	}

	s.owed.failing(nil)
	s.clock.At = now.Add(62 * time.Minute)
	s.sweep(t)
	s.owed.mu.Lock()
	s.owed.owed["api"] = []accounts.Program{"wrangler"}
	s.owed.mu.Unlock()
	s.owed.failing(errors.New("exit 1"))
	s.clock.At = now.Add(63 * time.Minute)
	s.sweep(t)
	if got := s.told.count(); got != 3 {
		t.Errorf("a removal failing again after one succeeded: told %d times, want it told at once (3)", got)
	}
}
