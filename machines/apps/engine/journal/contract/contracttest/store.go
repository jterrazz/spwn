package contracttest

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/journal/contract"
)

func line(at time.Time, text string) journal.Line {
	return journal.Line{At: at, Machine: "macbook-pro", Text: text, By: journal.ByYou, Outcome: journal.Done, Subject: &journal.Subject{Kind: journal.Machine, Name: "gate-2"}}
}

func Store(t *testing.T, open func(t *testing.T) contract.Journal) {
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 14, 0, 0, 0, time.UTC)

	t.Run("a store never written keeps no line", func(t *testing.T) {
		got, err := open(t).Lines(ctx, time.Time{}, time.Time{})
		if err != nil || len(got) != 0 {
			t.Fatalf("Lines = %+v, %v; want none", got, err)
		}
	})

	t.Run("lines written are read back newest first, between the bounds", func(t *testing.T) {
		s := open(t)
		if err := s.Write(ctx, now, line(now.Add(-2*time.Hour), "older")); err != nil {
			t.Fatalf("Write: %v", err)
		}
		if err := s.Write(ctx, now, line(now.Add(-time.Hour), "newer"), line(now.Add(-3*time.Hour), "oldest")); err != nil {
			t.Fatalf("Write: %v", err)
		}
		got, err := s.Lines(ctx, time.Time{}, time.Time{})
		if err != nil || len(got) != 3 || got[0].Text != "newer" || got[1].Text != "older" || got[2].Text != "oldest" {
			t.Fatalf("Lines = %+v, %v; want newer, older, oldest", got, err)
		}
		if got[0].Subject == nil || got[0].Subject.Name != "gate-2" || !got[0].At.Equal(now.Add(-time.Hour)) {
			t.Errorf("Lines[0] = %+v; want the line as written", got[0])
		}
		got, _ = s.Lines(ctx, now.Add(-150*time.Minute), now.Add(-time.Hour))
		if len(got) != 1 || got[0].Text != "older" {
			t.Errorf("Lines between = %+v; want older alone (since kept, before excluded)", got)
		}
	})

	t.Run("a write drops what is past the horizon", func(t *testing.T) {
		s := open(t)
		if err := s.Write(ctx, now, line(now.Add(-journal.Horizon+time.Hour), "kept")); err != nil {
			t.Fatalf("Write: %v", err)
		}
		if err := s.Write(ctx, now.Add(2*time.Hour), line(now, "new")); err != nil {
			t.Fatalf("Write: %v", err)
		}
		got, _ := s.Lines(ctx, time.Time{}, time.Time{})
		if len(got) != 1 || got[0].Text != "new" {
			t.Errorf("Lines = %+v; want the old line dropped", got)
		}
	})
}
