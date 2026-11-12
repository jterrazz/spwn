package fake

import (
	"context"

	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/world"
)

type Shell struct {
	// invariant: the owner's lines run on a machine of the world and end
	// well, saying nothing; a `Shell.Run` fault ends them at their first
	// line, exit 1, the shell's own words on stderr.
	w *world.World
}

var _ contract.Shell = (*Shell)(nil)

func ShellOf(w *world.World) *Shell { return &Shell{w: w} }

func (s *Shell) Run(context.Context, string) (contract.ShellRun, error) {
	s.w.Lock()
	defer s.w.Unlock()
	if s.w.Fails("Shell.Run") {
		return contract.ShellRun{ExitCode: 1, Stderr: "bash: line 1: the line failed\n"}, nil
	}
	return contract.ShellRun{}, nil
}
