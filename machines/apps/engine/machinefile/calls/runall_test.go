package calls

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
)

type quiet struct{}

func (quiet) Step(string, string) {}
func (quiet) Done(string, error)  {}

// value: a shell whose lines take the time their file is given on a clock
// of the test's, each run noted with the time it was given; cut, when set,
// is called as the named file's lines run.
type timed struct {
	now   time.Time
	takes map[string]time.Duration
	given []time.Duration
	cut   func()
}

func (s *timed) Run(_ context.Context, lines string, within time.Duration) (contract.ShellRun, error) {
	s.given = append(s.given, within)
	s.now = s.now.Add(min(s.takes[lines], within))
	if s.cut != nil {
		s.cut()
		return contract.ShellRun{ExitCode: 143}, nil
	}
	if s.takes[lines] > within {
		return contract.ShellRun{ExitCode: 137}, nil
	}
	return contract.ShellRun{}, nil
}

func scriptOf(file, lines string) machinefile.Script {
	return machinefile.Script{Phase: machinefile.RunBoot, Lines: lines, Origin: machinefile.Origin{Name: file}}
}

func states(ran []machinefile.Ran) []string {
	out := make([]string, 0, len(ran))
	for _, r := range ran {
		out = append(out, r.Script.Origin.Name+":"+string(r.State)+":"+r.Detail)
	}
	return out
}

func TestAPhasesLinesShareOneBoundAcrossItsFiles(t *testing.T) {
	sh := &timed{now: epoch(), takes: map[string]time.Duration{"a": 12 * time.Minute, "b": 12 * time.Minute, "c": time.Minute}}
	d := Deps{Shell: sh, Now: func() time.Time { return sh.now }}
	got := runAll(context.Background(), d, []machinefile.Script{scriptOf("a-kit", "a"), scriptOf("b-kit", "b"), scriptOf("top", "c")}, quiet{})
	want := []string{"a-kit:ok:", "b-kit:broken:exited 137", "top:absent:after b-kit's lines, which failed"}
	if !slices.Equal(states(got), want) || !slices.Equal(sh.given, []time.Duration{catalog.RunBound, catalog.RunBound - 12*time.Minute}) {
		t.Errorf("ran %q given %v; want %q, the second kit given what the first left of the bound", states(got), sh.given, want)
	}
	sh = &timed{now: epoch(), takes: map[string]time.Duration{"a": catalog.RunBound, "c": time.Minute}}
	d.Now = func() time.Time { return sh.now }
	d.Shell = sh
	got = runAll(context.Background(), d, []machinefile.Script{scriptOf("a-kit", "a"), scriptOf("top", "c")}, quiet{})
	if want := []string{"a-kit:ok:", "top:absent:the phase's lines had run their 20 minutes already"}; !slices.Equal(states(got), want) {
		t.Errorf("ran %q; want %q, the lines with no time left not run", states(got), want)
	}
}

func TestAnInterruptDuringAFilesLinesIsNoFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sh := &timed{now: epoch(), cut: cancel}
	got := runAll(ctx, Deps{Shell: sh, Now: func() time.Time { return sh.now }}, []machinefile.Script{scriptOf("a-kit", "a"), scriptOf("top", "c")}, quiet{})
	if len(got) != 1 || got[0].Script.Origin.Name != "a-kit" || len(sh.given) != 1 {
		t.Errorf("ran %q; want the interrupted kit's alone, nothing said of the rest", states(got))
	}
}
