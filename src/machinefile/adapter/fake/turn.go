package fake

import (
	"context"

	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/world"
)

type Turn struct {
	// invariant: one Mac's turn, held within the process; a turn the world
	// scripts held (`Turn.Take`) is refused as another run's.
	w   *world.World
	mac string
}

var _ contract.Turn = Turn{}

func TurnOf(w *world.World, mac string) Turn { return Turn{w: w, mac: mac} }

func (t Turn) Take(ctx context.Context) (func(), error) {
	t.w.Lock()
	defer t.w.Unlock()
	if t.w.Fails("Turn.Take") || !t.w.Mac(t.mac).TakeTurn() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return nil, contract.ErrBusy
	}
	return func() {
		t.w.Lock()
		defer t.w.Unlock()
		t.w.Mac(t.mac).GiveTurn()
	}, nil
}
