package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
)

type Deps struct {
	Holder Holder
	// value: this Mac: its name, the short label its words use, what it
	// reports it can do.
	Mac   string
	Label string
	Caps  machine.Capabilities
	// value: the servers this Mac hosts, by name.
	Servers func(ctx context.Context) []string
	// value: the workspaces running on this Mac now, by name.
	Running func(ctx context.Context) []string
	// value: the never_sleeps preference of this Mac.
	NeverSleeps func() bool
}

type Holder interface {
	// value: refuses a machine that cannot keep awake, in its own words.
	Check() error
	Read(ctx context.Context) <-chan awake.Report
	Awake(ctx context.Context, r awake.Request) (awake.Awoken, error)
	Sleep(ctx context.Context) (awake.Slept, error)
}

const (
	OutOfBounds fail.Code = "awake.out-of-bounds"
	NoLid       fail.Code = "awake.no-lid"
	NoBattery   fail.Code = "awake.no-battery"
)

func (d Deps) refused(err error, in awake.Request) error {
	detail := map[string]string{"label": d.Label, "for": in.Words}
	switch {
	case errors.Is(err, awake.ErrOutOfBounds):
		return fail.New(fail.Refused, OutOfBounds, in.Words, detail)
	case errors.Is(err, awake.ErrNoLid):
		return fail.New(fail.NotForThisMachine, NoLid, d.Mac, detail)
	case errors.Is(err, awake.ErrNoBattery):
		return fail.New(fail.NotForThisMachine, NoBattery, d.Mac, detail)
	}
	return err
}

func (d Deps) never() bool { return d.NeverSleeps != nil && d.NeverSleeps() }

func (d Deps) servers(ctx context.Context) []string {
	if d.Servers == nil {
		return nil
	}
	return d.Servers(ctx)
}

func (d Deps) running(ctx context.Context) []string {
	if d.Running == nil {
		return nil
	}
	return d.Running(ctx)
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.BindWatch(Read, d), call.Bind(Keep, d), call.Bind(Off, d)}
}
