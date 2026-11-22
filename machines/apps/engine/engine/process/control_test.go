package process

import (
	"context"
	"testing"
	"time"

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

type sleepingMac struct {
	Mac
	slept time.Duration
}

func (m *sleepingMac) Sleep(_ context.Context, d time.Duration) error {
	m.slept += d
	return nil
}

// value: an engine that answers, its Local Network read checking for its
// first hellos, then the answer it settles on.
type checkingPeer struct {
	Peer
	checking int
	settles  model.LocalNetwork
}

func (p *checkingPeer) Hello(context.Context) (engine.Hello, error) {
	if p.checking > 0 {
		p.checking--
		return engine.Hello{Version: "v3", LocalNetwork: model.LocalNetworkChecking}, nil
	}
	return engine.Hello{Version: "v3", LocalNetwork: p.settles}, nil
}

func TestAStartWaitsForTheEnginesLocalNetworkReadToSettleWithinItsBound(t *testing.T) {
	for _, c := range []struct {
		name      string
		checking  int
		want      model.LocalNetwork
		mostSlept time.Duration
	}{
		{"settled approved", 10, model.LocalNetworkApproved, 2 * time.Second},
		{"settled not approved", 3, model.LocalNetworkNotApproved, time.Second},
		{"still checking", 1000, model.LocalNetworkChecking, localNetworkSettleWait},
	} {
		h := &sleepingMac{}
		r, err := NewControl(h, &checkingPeer{checking: c.checking, settles: c.want}, nil, nil, installedJob{installed: true}, ControlOptions{}).Start(t.Context())
		if err != nil || r.LocalNetwork != c.want || h.slept > c.mostSlept {
			t.Errorf("%s: Start = %q after %s, %v; want %q within %s", c.name, r.LocalNetwork, h.slept, err, c.want, c.mostSlept)
		}
	}
}
