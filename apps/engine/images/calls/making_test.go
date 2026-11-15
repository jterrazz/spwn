package calls

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/images"
	imagesfake "github.com/jterrazz/studio-station/apps/engine/images/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type running map[int]bool

func (running) Self() (machines.Process, error) { return machines.Process{PID: 1, Start: "1"}, nil }
func (r running) Runs(p machines.Process) bool  { return r[p.PID] }

func TestAMarkWhoseProcessIsGoneIsNoBuild(t *testing.T) {
	marks := imagesfake.NewMarks()
	alive, dead := &machines.Process{PID: 10, Start: "a"}, &machines.Process{PID: 11, Start: "b"}
	for _, m := range []images.Making{
		{Image: "station-linux", Version: "1", Step: images.BuildPackages, Owner: alive},
		{Image: "studio-linux", Version: "1", Step: images.BuildSeal, Owner: dead},
		{Image: "macos", Version: "1", Step: images.BuildSeal},
	} {
		if err := marks.Mark(context.Background(), m); err != nil {
			t.Fatal(err)
		}
	}
	s := &Builder{marks: marks, procs: running{10: true}}
	got := s.making(context.Background())
	if len(got) != 1 || got[0].Image != "station-linux" {
		t.Fatalf("builds making = %+v; want only the one whose process runs", got)
	}
}

func TestABuildMarksItselfAndClearsAtTheEnd(t *testing.T) {
	marks := imagesfake.NewMarks()
	s := &Builder{marks: marks, procs: running{1: true}}
	s.mark(images.Making{Image: "station-linux", Version: "1", Step: images.BuildSettings, Owner: s.owner()})
	if got := s.making(context.Background()); len(got) != 1 || got[0].Step != images.BuildSettings {
		t.Fatalf("a marked build reads %+v", got)
	}
	s.clear("station-linux")
	if got := s.making(context.Background()); len(got) != 0 {
		t.Fatalf("a cleared build is left: %+v", got)
	}
}
