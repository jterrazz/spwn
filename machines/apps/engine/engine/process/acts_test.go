package process_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/guide"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type memFile struct{ data []byte }

func (f *memFile) Read() ([]byte, error)   { return f.data, nil }
func (f *memFile) Write(data []byte) error { f.data = data; return nil }

type owners struct{ running bool }

func projected(a guide.Act) (guide.Guide, bool) {
	return machines.CreateGuide(a, nil), a.Family == machines.GuideCreate
}

func TestThePhaseThatPutsAnActAnswersItsFirstGuide(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC) }
	board := process.OpenActs(&memFile{}, owners{running: true}, now, func(guide.Acts) error { return nil }, projected)
	rail := []guide.Rung{{ID: "cloning", Label: "Disk"}, {ID: "booting", Label: "Boot"}}
	id, first := board.Phase(machines.GuideCreate, guide.Phase{Subject: "linux-2", Rail: rail, Step: "cloning", Line: "cloning"})
	_, later := board.Phase(machines.GuideCreate, guide.Phase{Subject: "linux-2", Step: "booting"})
	if first == nil || first.ID != id || first.View.Headline.Now != "Disk now" || !first.Live {
		t.Fatalf("the act's first phase answers its live Guide at its first step: %+v", first)
	}
	if later != nil {
		t.Fatalf("a later phase of the same act answers no Guide: %+v", later)
	}
}

func (o owners) Self() (machines.Process, error) { return machines.Process{PID: 42}, nil }
func (o owners) Runs(machines.Process) bool      { return o.running }

func TestEveryWriteOfTheBoardReachesTheStoreAtOnce(t *testing.T) {
	file := &memFile{}
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	var seen []guide.Acts
	board := process.OpenActs(file, owners{running: true}, func() time.Time { return now }, func(a guide.Acts) error {
		seen = append(seen, a)
		return nil
	}, projected)
	if len(seen) != 0 {
		t.Fatalf("an empty board put %v", seen)
	}
	rail := []guide.Rung{{ID: "cloning", Label: "Disk"}, {ID: "booting", Label: "Boot"}}
	id, _ := board.Phase(machines.GuideCreate, guide.Phase{Subject: "linux-2", Rail: rail, Step: "cloning", Line: "cloning"})
	board.Phase(machines.GuideCreate, guide.Phase{Subject: "linux-2", Step: "booting", Line: "Booting linux-2"})
	if len(seen) != 2 || seen[1].Acts[0].Step != "booting" || len(seen[1].Acts[0].Rail) != 2 {
		t.Fatalf("each step puts the act at once, its rail kept: %+v", seen)
	}
	board.End(id, guide.EndedFailed, &call.WireError{Code: "machines.did-not-boot", Message: "linux-2 did not boot"})
	if got := seen[2].Acts[0]; got.Ended != guide.EndedFailed || got.Code != "machines.did-not-boot" || got.Line != "" {
		t.Fatalf("its end puts the failure: %+v", got)
	}
	now = now.Add(process.ActLinger)
	board.Phase(machines.GuideCreate, guide.Phase{Subject: "linux-3", Rail: rail, Step: "cloning"})
	if acts := seen[3].Acts; len(acts) != 1 || acts[0].Subject != "linux-3" {
		t.Fatalf("an act that lingered its 10 minutes leaves the board: %+v", acts)
	}
}

func TestAnActWhoseProcessNoLongerRunsReadsInterrupted(t *testing.T) {
	file := &memFile{}
	now := func() time.Time { return time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC) }
	put := func(guide.Acts) error { return nil }
	process.OpenActs(file, owners{running: true}, now, put, projected).Phase(machines.GuideCreate, guide.Phase{Subject: "linux-2", Step: "booting"})
	var seen guide.Acts
	process.OpenActs(file, owners{running: false}, now, func(a guide.Acts) error { seen = a; return nil }, projected)
	if len(seen.Acts) != 1 || seen.Acts[0].Ended != guide.EndedInterrupted || seen.Acts[0].Step != "booting" {
		t.Fatalf("the next engine reads the cut act interrupted at its step: %+v", seen)
	}
}

func TestARefusedRepairEndsOnlyAnEndedActFailedOnItsWords(t *testing.T) {
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	var seen guide.Acts
	board := process.OpenActs(&memFile{}, owners{running: true}, func() time.Time { return now }, func(a guide.Acts) error {
		seen = a
		return nil
	}, projected)
	rail := []guide.Rung{{ID: "cloning", Label: "Disk"}, {ID: "booting", Label: "Boot"}}
	id, _ := board.Phase(machines.GuideCreate, guide.Phase{Subject: "linux-2", Rail: rail, Step: "booting"})
	refused := &call.WireError{Code: "machines.not-enough-memory", Message: "linux-2 needs 4 GB, and 2 GB are free"}
	board.Refused(id, refused)
	if got := seen.Acts[0]; got.Ended != "" || got.Cause != "" {
		t.Fatalf("an act that still runs takes no refusal: %+v", got)
	}
	board.End(id, guide.EndedInterrupted, nil)
	now = now.Add(time.Minute)
	board.Refused(id, refused)
	if got := seen.Acts[0]; got.Ended != guide.EndedFailed || got.Code != string(refused.Code) || got.Cause != refused.Message || !got.At.Equal(now) || got.Step != "booting" {
		t.Fatalf("an ended act whose repair is refused ends failed at its step on the refusal, from then: %+v", got)
	}
}
