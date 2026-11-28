package fake

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Shell struct {
	// invariant: the owner's lines run on a machine of the world and end
	// well, saying nothing; a `Shell.Run` fault ends them at their first
	// line, exit 1, the shell's own words on stderr and then the driver's
	// line naming it, as the bash adapter says them. A `sleep <seconds>`
	// line takes that long on the world's clock, and lines that would take
	// longer than they are given end at it, timed out, as the runner ends
	// them.
	w *world.World
}

var _ contract.Shell = (*Shell)(nil)

func ShellOf(w *world.World) *Shell { return &Shell{w: w} }

func (s *Shell) Run(_ context.Context, lines string, within time.Duration) (contract.ShellRun, error) {
	s.w.Lock()
	defer s.w.Unlock()
	if s.w.Fails("Shell.Run") {
		first, _, _ := strings.Cut(strings.TrimLeft(lines, "\n"), "\n")
		return contract.ShellRun{ExitCode: 1, Stderr: "bash: line 1: the line failed\nstation: exit 1 at: " + first + "\n"}, nil
	}
	took := sleeps(lines)
	if within > 0 && took > within {
		s.w.Now = s.w.Now.Add(within)
		return contract.ShellRun{}, errors.New("command timed out")
	}
	s.w.Now = s.w.Now.Add(took)
	return contract.ShellRun{}, nil
}

func sleeps(lines string) time.Duration {
	var took time.Duration
	for _, l := range strings.Split(lines, "\n") {
		if f := strings.Fields(l); len(f) == 2 && f[0] == "sleep" {
			if n, err := strconv.Atoi(f[1]); err == nil {
				took += time.Duration(n) * time.Second
			}
		}
	}
	return took
}
