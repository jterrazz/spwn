package contracttest

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/agents/contract"
	"github.com/jterrazz/studio-station/src/catalog"
)

type JournalSubject struct {
	// value: a journal over a clock the suite moves.
	Journal contract.Journal
	Now     *time.Time
}

func Morning() time.Time { return time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC) }

func stepsOf(n int) []agents.Step {
	out := make([]agents.Step, 0, n)
	for i := range n {
		out = append(out, agents.Step{At: Morning().Add(time.Duration(i) * time.Minute), Verb: catalog.AgentEdit, Object: "src/" + string(rune('a'+i)) + ".ts"})
	}
	return out
}

func sessions(as []agents.Agent) []string {
	out := make([]string, 0, len(as))
	for _, a := range as {
		out = append(out, a.Session)
	}
	return out
}

func Journal(t *testing.T, open func(t *testing.T) JournalSubject) {
	ctx := context.Background()

	t.Run("a session written is read back with its last steps alone", func(t *testing.T) {
		s := open(t)
		a := agents.Agent{App: "claudecode", Session: "s-1", State: agents.Working, Task: "build the landing page", Steps: stepsOf(5), Since: Morning()}
		if err := s.Journal.Write(ctx, a); err != nil {
			t.Fatalf("Write: %v", err)
		}
		got, err := s.Journal.Read(ctx)
		if err != nil || len(got) != 1 {
			t.Fatalf("Read = %+v, %v; want one session", got, err)
		}
		if want := stepsOf(5)[5-catalog.AgentStepsKept:]; !slices.EqualFunc(got[0].Steps, want, func(a, b agents.Step) bool { return a.Object == b.Object && a.At.Equal(b.At) }) {
			t.Errorf("steps %+v, want the last %d", got[0].Steps, catalog.AgentStepsKept)
		}
		if got[0].Task != a.Task || got[0].State != a.State || !got[0].Since.Equal(a.Since) {
			t.Errorf("read %+v, want %+v", got[0], a)
		}
	})

	t.Run("writing a session again replaces it and leaves the others", func(t *testing.T) {
		s := open(t)
		for _, a := range []agents.Agent{{Session: "s-1", State: agents.Working}, {Session: "s-2", State: agents.Working}, {Session: "s-1", State: agents.Waiting}} {
			if err := s.Journal.Write(ctx, a); err != nil {
				t.Fatalf("Write: %v", err)
			}
		}
		got, err := s.Journal.Read(ctx)
		if err != nil || !slices.Equal(sessions(got), []string{"s-1", "s-2"}) || got[0].State != agents.Waiting {
			t.Errorf("Read = %+v, %v; want s-1 waiting and s-2", got, err)
		}
	})

	t.Run("a session forgotten is gone, and forgetting one not kept changes nothing", func(t *testing.T) {
		s := open(t)
		for _, name := range []string{"s-1", "s-2"} {
			if err := s.Journal.Write(ctx, agents.Agent{Session: name, State: agents.Finished}); err != nil {
				t.Fatalf("Write: %v", err)
			}
		}
		for _, name := range []string{"s-1", "s-1", "none"} {
			if err := s.Journal.Forget(ctx, name); err != nil {
				t.Fatalf("Forget %s: %v", name, err)
			}
		}
		if got, err := s.Journal.Read(ctx); err != nil || !slices.Equal(sessions(got), []string{"s-2"}) {
			t.Errorf("Read = %+v, %v; want s-2 alone", got, err)
		}
	})

	t.Run("a session whose name is a path is refused", func(t *testing.T) {
		s := open(t)
		for _, name := range []string{"", "../escape", "a/b", ".", "s 1"} {
			if err := s.Journal.Write(ctx, agents.Agent{Session: name}); err == nil {
				t.Errorf("a session named %q was written", name)
			}
		}
		if got, _ := s.Journal.Read(ctx); len(got) != 0 {
			t.Errorf("Read = %+v; want nothing", got)
		}
	})

	t.Run("a session ended past its keep is not read", func(t *testing.T) {
		s := open(t)
		ended := agents.Agent{Session: "old", State: agents.Finished, Ended: *s.Now}
		if err := s.Journal.Write(ctx, ended); err != nil {
			t.Fatalf("Write: %v", err)
		}
		*s.Now = s.Now.Add(catalog.AgentKeptFor)
		if got, _ := s.Journal.Read(ctx); !slices.Equal(sessions(got), []string{"old"}) {
			t.Fatalf("Read = %+v on its last day; want it kept", got)
		}
		*s.Now = s.Now.Add(time.Hour)
		if err := s.Journal.Write(ctx, agents.Agent{Session: "new", State: agents.Working}); err != nil {
			t.Fatalf("Write: %v", err)
		}
		if got, _ := s.Journal.Read(ctx); !slices.Equal(sessions(got), []string{"new"}) {
			t.Errorf("Read = %+v; want the ended session gone", got)
		}
	})

	t.Run("watch yields the journal now and after each change, and closes when asked", func(t *testing.T) {
		s := open(t)
		if err := s.Journal.Write(ctx, agents.Agent{Session: "s-1", State: agents.Working}); err != nil {
			t.Fatal(err)
		}
		watching, stop := context.WithCancel(ctx)
		frames, err := s.Journal.Watch(watching)
		if err != nil {
			t.Fatalf("Watch: %v", err)
		}
		if first := next(t, frames); !slices.Equal(sessions(first), []string{"s-1"}) {
			t.Fatalf("first frame %+v, want s-1", first)
		}
		if err := s.Journal.Write(ctx, agents.Agent{Session: "s-1", State: agents.Waiting}); err != nil {
			t.Fatal(err)
		}
		if changed := next(t, frames); len(changed) != 1 || changed[0].State != agents.Waiting {
			t.Fatalf("frame %+v, want s-1 waiting", changed)
		}
		stop()
		deadline, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		for {
			select {
			case _, open := <-frames:
				if !open {
					return
				}
			case <-deadline.Done():
				t.Fatal("the watch did not close once its context ended")
			}
		}
	})
}

func next(t *testing.T, frames <-chan []agents.Agent) []agents.Agent {
	t.Helper()
	deadline, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case f, ok := <-frames:
		if !ok {
			t.Fatal("the watch closed")
		}
		return f
	case <-deadline.Done():
		t.Fatal("no frame within 5 s")
	}
	return nil
}
