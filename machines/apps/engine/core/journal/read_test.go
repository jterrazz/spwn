package journal

import (
	"slices"
	"testing"
	"time"
)

func at(day, hour int) time.Time { return time.Date(2026, 11, day, hour, 0, 0, 0, time.UTC) }

func TestANightMergesWhatReadsAlikeAndKeepsAFailurePerMachine(t *testing.T) {
	now, night := at(14, 10), at(14, 3)
	node := &Subject{Kind: App, Name: "node"}
	up := &Upgrade{From: "24.11.0", To: "24.11.1"}
	lines := []Line{
		{At: night, Machine: "mac-mini", Text: "node", Subject: node, Upgrade: up, Outcome: Done, Night: night},
		{At: night, Machine: "macbook-pro", Text: "node", Subject: node, Upgrade: up, Outcome: Done, Night: night},
		{At: night.Add(time.Minute), Machine: "macbook-pro", Text: "ripgrep", Outcome: Failed, Night: night},
		{At: night.Add(time.Minute), Machine: "mac-mini", Text: "ripgrep", Outcome: Failed, Night: night},
		{At: night, Machine: "macbook-pro", Text: "orbstack", Outcome: Held, Night: night},
		{At: at(14, 9), Machine: "linux-1", Text: "linux-1 restarted", Outcome: Done, By: "you"},
	}
	groups, next := Read(lines, now, Query{})
	if len(groups) != 2 || groups[0].Title != "Today" || groups[1].Title != "Last night" || next != "" {
		t.Fatalf("groups = %+v, next %q", groups, next)
	}
	n := groups[1]
	if n.Sum != "1 moved, 1 held, 2 failed" || len(n.Lines) != 4 || n.Lines[0].Outcome != Failed || n.Lines[3].Outcome != Held {
		t.Fatalf("night = %+v", n)
	}
	if !slices.Equal(n.Lines[2].On, []string{"mac-mini", "macbook-pro"}) {
		t.Errorf("node's machines = %v; want both Macs", n.Lines[2].On)
	}
	if narrowed, _ := Read(lines, now, Query{Narrowed: true}); narrowed[1].Sum != "" {
		t.Errorf("a narrowed read sums %q; want none", narrowed[1].Sum)
	}
}

func TestBeforePagesToTheNextOlderDayThatHoldsALine(t *testing.T) {
	now := at(20, 10)
	lines := []Line{
		{At: at(20, 9), Machine: "m", Text: "today", Outcome: Done},
		{At: at(12, 9), Machine: "m", Text: "older", Outcome: Done},
		{At: at(2, 9), Machine: "m", Text: "older still", Outcome: Done},
		{At: at(19, 23).Add(-30 * 24 * time.Hour), Machine: "m", Text: "dropped", Outcome: Done},
	}
	groups, next := Read(lines, now, Query{})
	if len(groups) != 1 || groups[0].Day != "2026-11-20" || next != "2026-11-13" {
		t.Fatalf("first page = %+v, next %q", groups, next)
	}
	groups, next = Read(lines, now, Query{Before: next, Days: 1})
	if len(groups) != 1 || groups[0].Lines[0].Text != "older" || groups[0].Title != "Thursday 12 November" || next != "2026-11-03" {
		t.Fatalf("the 12th = %+v, next %q", groups, next)
	}
}
