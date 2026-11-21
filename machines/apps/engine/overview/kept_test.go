package overview_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/overview"
)

func TestAKeptOverviewSaysItsUptimeAsOfTheInstantItIsDrawn(t *testing.T) {
	read := time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	s := overview.Summary{Head: overview.Head{Name: "macbook", Kind: machine.Computer, ThisMac: true, Up: 2 * time.Hour}, Read: read, VMs: []overview.VM{}}
	s.Head.Verdict = overview.HeadOf(s)
	kept := overview.KeptOf(s)
	if kept.Summary.Head.Up != 0 || !kept.Summary.Read.IsZero() || kept.Summary.Head.Verdict.Line != "" {
		t.Fatalf("kept holds a value of the clock: %+v", kept.Summary.Head)
	}
	if again := overview.KeptOf(overview.Summary{Head: overview.Head{Name: "macbook", Kind: machine.Computer, ThisMac: true, Up: 3 * time.Hour}, Read: read.Add(time.Hour), VMs: []overview.VM{}}); !reflect.DeepEqual(again, kept) {
		t.Fatalf("two readings of one boot keep two values: %+v and %+v", again, kept)
	}
	if got := kept.At(read, read); got.Head.Up != 2*time.Hour || got.Head.Verdict.Line != s.Head.Verdict.Line || !got.Read.Equal(read) {
		t.Fatalf("at the reading's instant: up %s, line %q, read %s; want %s, %q", got.Head.Up, got.Head.Verdict.Line, got.Read, 2*time.Hour, s.Head.Verdict.Line)
	}
	later := kept.At(read.Add(24*time.Hour), read)
	if later.Head.Up != 26*time.Hour || later.Head.Verdict.Line == s.Head.Verdict.Line {
		t.Fatalf("a day later: up %s, line %q", later.Head.Up, later.Head.Verdict.Line)
	}
	if unknown := (overview.Kept{Summary: kept.Summary}).At(read, read); unknown.Head.Up != 0 || unknown.Head.Verdict.Line != "" {
		t.Fatalf("no boot instant says an uptime: %+v", unknown.Head)
	}
}
