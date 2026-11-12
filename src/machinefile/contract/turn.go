package contract

import (
	"context"
	"errors"
	"time"
)

type Turn interface {
	// invariant: one apply, upgrade or night of a machine at a time, across
	// every process of it: Take holds the machine's turn until release,
	// waits up to TurnWait for a run that holds it, then refuses with
	// ErrBusy; a process that ends lets go of it.
	Take(ctx context.Context) (release func(), err error)
}

// value: how long a run waits for another to give the turn back before it
// says so: long enough for a run ending, never for one that installs.

const TurnWait = 10 * time.Second

var ErrBusy = errors.New("another station run holds this machine: an apply, an upgrade or the night's")
