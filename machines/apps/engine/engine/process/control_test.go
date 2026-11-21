package process

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

// value: a Mac whose disk holds no file at all, as a Mac whose engine's job
// is its app's own holds no definition in ~/Library/LaunchAgents.
type bareMac struct{ Mac }

func (bareMac) Home() string       { return "/Users/o" }
func (bareMac) Exists(string) bool { return false }

type silentPeer struct{ Peer }

func (silentPeer) Hello(context.Context) (engine.Hello, error) {
	return engine.Hello{}, model.ErrEngineDown
}

// value: the engine's job as its job manager answers: installed or not,
// never loaded.
type installedJob struct {
	Job
	installed bool
}

func (j installedJob) Installed() bool { return j.installed }

func TestAnEngineThatDoesNotAnswerIsStoppedWhenItsJobManagerHoldsItsJobWhateverTheDiskHolds(t *testing.T) {
	for installed, want := range map[bool]model.EngineState{true: model.EngineStopped, false: model.EngineNotInstalled} {
		c := NewControl(bareMac{}, silentPeer{}, nil, nil, installedJob{installed: installed}, ControlOptions{})
		r, err := c.Read(t.Context())
		if err != nil || r.State != want {
			t.Errorf("installed %v: Read = %q, %v; want %q", installed, r.State, err, want)
		}
		if checked := c.Check(); (checked == nil) != installed {
			t.Errorf("installed %v: Check = %v", installed, checked)
		}
	}
}
