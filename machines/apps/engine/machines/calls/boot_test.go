package calls

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

// value: an engine whose guest answers every line at once, the deadline of
// each line kept.
type boundEngine struct {
	Engine
	until []time.Time
}

func (e *boundEngine) GuestExec(ctx context.Context, _, _, _ string) (GuestRan, error) {
	until, _ := ctx.Deadline()
	e.until = append(e.until, until)
	return GuestRan{}, nil
}

func TestABootPhaseOutlastsItsPackagesAndItsLines(t *testing.T) {
	// invariant: the boot phase's apply is given the packages' share and the
	// lines' bound together, so lines that run long end as a broken
	// run.boot step inside it, never as the apply cut short.
	e := &boundEngine{}
	s := &Machinist{engineAPI: e}
	want := catalog.PackagesBound + catalog.RunBound
	ref, cancel := context.WithTimeout(context.Background(), want)
	defer cancel()
	due, _ := ref.Deadline()
	if err := s.applyBoot(context.Background(), "web", model.Template{Name: "studio-linux"}); err != nil {
		t.Fatal(err)
	}
	if off := e.until[0].Sub(due); off < 0 || off > time.Minute {
		left := e.until[0].Sub(due) + want
		t.Fatalf("the boot phase had %v; want %v, the packages' share and the lines' bound", left, want)
	}
}

// value: an engine whose guest's apply ends as the given run.
type ranEngine struct {
	Engine
	ran GuestRan
}

func (e ranEngine) GuestExec(context.Context, string, string, string) (GuestRan, error) {
	return e.ran, nil
}

func TestAFirstBootWhoseLinesRanOutOfTimeFails(t *testing.T) {
	// invariant: the boot phase's apply exits non-zero when a file's lines
	// were left for want of time, and the first boot fails with its words,
	// never taken as done.
	s := &Machinist{engineAPI: ranEngine{ran: GuestRan{ExitCode: 1, Stderr: "Error: 1 of 2 steps failed\nHint: station machinefile apply\n"}}}
	err := s.applyBoot(context.Background(), "web", model.Template{Name: "studio-linux"})
	if err == nil || err.Error() != "the boot phase exited 1: Error: 1 of 2 steps failed" {
		t.Fatalf("a boot phase whose lines ran out of time: %v; want it failed with the apply's words", err)
	}
}

// value: a Mac whose clock stands still and whose kept lines are collected by file.
type keepingMac struct {
	ThisMac
	kept map[string]string
}

func (keepingMac) Now() time.Time { return time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC) }

func (m keepingMac) Append(file, line string) error {
	m.kept[file] += line
	return nil
}

func TestARefusedGrantWithoutACodeKeepsItsCauseInTheEngineLog(t *testing.T) {
	mac := keepingMac{kept: map[string]string{}}
	s := &Machinist{h: mac, o: Options{Logs: "/state/logs"}}
	cause := errors.New("dial tcp 192.0.2.7:22: connection refused")
	code, subject := s.refusal(cause, "web")
	if code != accounts.GrantCannotDeliver || subject != "web" {
		t.Fatalf("got %q on %q; want %q on web", code, subject, accounts.GrantCannotDeliver)
	}
	want := "2026-11-16T09:00:00Z grant to web: dial tcp 192.0.2.7:22: connection refused\n"
	if got := mac.kept["/state/logs/engine.log"]; got != want {
		t.Fatalf("the engine log holds %q; want %q", got, want)
	}
}

func TestARefusedGrantWithACodeKeepsItsCodeAndLogsNothing(t *testing.T) {
	mac := keepingMac{kept: map[string]string{}}
	s := &Machinist{h: mac, o: Options{Logs: "/state/logs"}}
	code, subject := s.refusal(accounts.Refusal(fail.Refused, accounts.GrantNoPassword, "tool-a"), "web")
	if code != accounts.GrantNoPassword || subject != "tool-a" || len(mac.kept) != 0 {
		t.Fatalf("got %q on %q with %d files logged; want %q on tool-a and none", code, subject, len(mac.kept), accounts.GrantNoPassword)
	}
}
