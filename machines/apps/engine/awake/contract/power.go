package contract

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

var ErrUnsupported = errors.New("no power management here: macOS only")

type PowerCaps struct {
	Battery bool
	Lid     bool
	Thermal bool
}

func (p PowerCaps) Onto(c machine.Capabilities) machine.Capabilities {
	// invariant: the three capabilities Power owns, set on a machine's; the
	// others are their own contracts'.
	c.Battery, c.Lid, c.Thermal = p.Battery, p.Lid, p.Thermal
	return c
}

type Power interface {
	// invariant: what this Mac reports it has, never derived from its model;
	// a Mac with no power management reports none.
	Capabilities() PowerCaps
	// invariant: the fields of a missing capability stay zero; ErrUnsupported
	// on a machine with no power management.
	Read() (awake.Reading, error)
	// invariant: one Event per field that moved; closed with ctx, or at once
	// where there is no power management.
	Watch(ctx context.Context) <-chan awake.Event
	// invariant: an idle-sleep assertion only, never a display one; release
	// is safe to call twice.
	Hold(name string) (release func(), err error)
	// invariant: held keeps the Mac awake with its lid closed; false is the
	// safe direction, macOS deciding again.
	Clamshell(held bool) error
	DisplaySleepNow(ctx context.Context) error
	// invariant: SPEC B5.15: macOS told this Mac it is going to sleep
	// (kIOMessageSystemWillSleep) and it has not woken since, as the process
	// that watches heard it; false where there is no power management.
	Sleeping() bool
}
