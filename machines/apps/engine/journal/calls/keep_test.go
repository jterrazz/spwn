package calls_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/journal/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/journal/calls"
)

func TestAGestureIsKeptOnItsMachineByItsAuthorOnThisMacsClock(t *testing.T) {
	now := time.Date(2026, 11, 4, 3, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		name string
		ran  calls.Ran
		line journal.Line
		want journal.Line
	}{
		{"the owner's, done", calls.Ran{Call: "machines.delete", Machine: "macbook-pro"}, journal.Line{Machine: "gate-2", Text: "gate-2 deleted"},
			journal.Line{At: now, Machine: "gate-2", Call: "machines.delete", Text: "gate-2 deleted", By: journal.ByYou, Outcome: journal.Done}},
		{"a session's, failed", calls.Ran{Call: "disk.free", Machine: "linux-1", By: "review-bot", Failed: "the disk did not answer"}, journal.Line{Text: "caches freed"},
			journal.Line{At: now, Machine: "linux-1", Call: "disk.free", Text: "caches freed", By: "review-bot", Outcome: journal.Failed, Detail: "the disk did not answer"}},
		{"the night's, its own outcome and retry", calls.Ran{Call: "machinefile.upgrade", Machine: "mac-mini", By: journal.ByNight, Failed: "one app failed"}, journal.Line{Text: "ripgrep", Outcome: journal.Failed, Retry: &finding.Fix{Call: "apps.update"}},
			journal.Line{At: now, Machine: "mac-mini", Call: "machinefile.upgrade", Text: "ripgrep", By: journal.ByNight, Outcome: journal.Failed, Retry: &finding.Fix{Call: "apps.update", On: "mac-mini"}, Night: now}},
		{"a create by the session that asked it", calls.Ran{Call: "machines.create-workspace", Machine: "macbook-pro"}, journal.Line{Machine: "api", Text: "api created", By: "review-bot"},
			journal.Line{At: now, Machine: "api", Call: "machines.create-workspace", Text: "api created", By: "review-bot", Outcome: journal.Done}},
	} {
		t.Run(c.name, func(t *testing.T) {
			store := fake.New()
			if err := calls.Keep(t.Context(), calls.Deps{Journal: store, Now: func() time.Time { return now }}, c.ran, c.line); err != nil {
				t.Fatal(err)
			}
			lines, err := store.Lines(t.Context(), time.Time{}, time.Time{})
			if err != nil || len(lines) != 1 {
				t.Fatalf("Lines = %v, %v; want the one line kept", lines, err)
			}
			got := lines[0]
			if got.At != c.want.At || got.Machine != c.want.Machine || got.Call != c.want.Call || got.Text != c.want.Text || got.By != c.want.By || got.Outcome != c.want.Outcome || got.Detail != c.want.Detail || !got.Night.Equal(c.want.Night) || (got.Retry == nil) != (c.want.Retry == nil) || got.Retry != nil && got.Retry.On != c.want.Retry.On {
				t.Errorf("kept %+v; want %+v", got, c.want)
			}
		})
	}
}

func TestAnActKeepsTheStepsItMarkedWithTheirStartAndDuration(t *testing.T) {
	// invariant: a line carries each step its act marked, in order, each
	// starting where it was marked and lasting to the next mark, the last to
	// the instant the line is kept; a step marked twice in a row is one.
	start := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	ctx := journal.Recording(t.Context())
	journal.Mark(ctx, start, "clone")
	journal.Mark(ctx, start.Add(2*time.Second), "boot")
	journal.Mark(ctx, start.Add(3*time.Second), "boot")
	journal.Mark(ctx, start.Add(8*time.Second), "address")
	end := start.Add(9500 * time.Millisecond)
	store := fake.New()

	if err := calls.Keep(ctx, calls.Deps{Journal: store, Now: func() time.Time { return end }}, calls.Ran{Call: "machines.create-workspace", Machine: "mac"}, journal.Line{Text: "api created"}); err != nil {
		t.Fatal(err)
	}
	lines, _ := store.Lines(t.Context(), time.Time{}, time.Time{})
	want := []journal.Phase{
		{Name: "clone", At: start, Took: 2 * time.Second},
		{Name: "boot", At: start.Add(2 * time.Second), Took: 6 * time.Second},
		{Name: "address", At: start.Add(8 * time.Second), Took: 1500 * time.Millisecond},
	}
	if len(lines) != 1 || !slices.Equal(lines[0].Phases, want) {
		t.Errorf("kept %+v; want the phases %+v", lines, want)
	}
}

func TestAnActThatMarkedNothingKeepsNoSteps(t *testing.T) {
	store := fake.New()
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)

	if err := calls.Keep(journal.Recording(t.Context()), calls.Deps{Journal: store, Now: func() time.Time { return now }}, calls.Ran{Call: "disk.free", Machine: "mac"}, journal.Line{Text: "caches freed"}); err != nil {
		t.Fatal(err)
	}
	lines, _ := store.Lines(t.Context(), time.Time{}, time.Time{})
	if len(lines) != 1 || lines[0].Phases != nil {
		t.Errorf("kept %+v; want a line with no steps", lines)
	}
}
