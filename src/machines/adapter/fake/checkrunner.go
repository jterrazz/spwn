package fake

import (
	"context"
	"errors"
	"time"

	"github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/machines/contract"
	"github.com/jterrazz/studio-station/src/world"
)

var ErrNoAnswer = errors.New("the check did not answer")

type CheckRunner struct{ w *world.World }

var _ contract.CheckRunner = (*CheckRunner)(nil)

func CheckRunnerOf(w *world.World) *CheckRunner { return &CheckRunner{w: w} }

// value: how long a check of the world takes to answer, unless its VM says.
const answersIn = 41 * time.Millisecond

func (p *CheckRunner) Run(_ context.Context, t machines.CheckTarget) (machines.CheckAnswer, error) {
	// invariant: a check of the world answers while the VM its name holds
	// runs, until a scripted fault of CheckRunner.Run stands for what
	// inside stopped answering it; a check on any other name answers. A URL
	// answers 200, in the time its VM's check takes (41 ms when unsaid).
	p.w.Lock()
	defer p.w.Unlock()
	if p.w.Fails("CheckRunner.Run") {
		return machines.CheckAnswer{}, ErrNoAnswer
	}
	took := answersIn
	if mac := p.w.Holder(t.Name); mac != "" {
		v := p.w.Macs[mac].VMs[t.Name]
		if !v.Running() {
			return machines.CheckAnswer{}, ErrNoAnswer
		}
		if v.CheckTakes > 0 {
			took = v.CheckTakes
		}
	}
	if t.URL == "" {
		return machines.CheckAnswer{Took: took}, nil
	}
	return machines.CheckAnswer{Status: "200", Took: took}, nil
}
