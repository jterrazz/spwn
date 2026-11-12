package files

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/src/machinefile/contract"
)

// value: the machine's turn, a flock(2) on one file of station's state,
// `~/.local/state/station/runs.lock`, tried until it is free or the wait
// ends.

type Turn struct {
	path string
	take Exclusive
	wait time.Duration
}

type Exclusive func(ctx context.Context, path string, within time.Duration) (release func(), held bool, err error)

var _ contract.Turn = Turn{}

func NewTurn(path string, take Exclusive) Turn {
	return Turn{path: path, take: take, wait: contract.TurnWait}
}

func (t Turn) Take(ctx context.Context) (func(), error) {
	release, held, err := t.take(ctx, t.path, t.wait)
	if err != nil {
		return nil, err
	}
	if held {
		return nil, contract.ErrBusy
	}
	return release, nil
}
