package macos

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/awake/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const pmsetWait = 5 * time.Second

type Commands interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Power struct {
	door  darwin.Door
	run   Commands
	pmset string
}

func New(door darwin.Door, run Commands, root string) *Power {
	// value: root is where /usr/bin/pmset is read under: "" on a Mac, the
	// sandbox in a spec, so the real one never runs there; launchd's PATH is no
	// guide, so it is absolute.
	return &Power{door: door, run: run, pmset: root + "/usr/bin/pmset"}
}

func unsupported(err error) error {
	if errors.Is(err, darwin.ErrPowerUnsupported) {
		return contract.ErrUnsupported
	}
	return err
}

func (p *Power) Capabilities() contract.PowerCaps {
	// invariant: a lid is what reports a lid state and a battery is what
	// IOPowerSources lists as internal, so a desktop Mac reports neither,
	// whatever its model.
	r, err := p.door.Read()
	if err != nil {
		return contract.PowerCaps{}
	}
	return contract.PowerCaps{Battery: r.HasBattery, Lid: r.Lid != "", Thermal: true}
}

func (p *Power) Read() (awake.Reading, error) {
	r, err := p.door.Read()
	if err != nil {
		return awake.Reading{}, unsupported(err)
	}
	return awake.Reading{
		Lid: r.Lid, Source: r.Source, HasBattery: r.HasBattery, Percent: r.Percent,
		Charging: r.Charging, LowPower: r.LowPower, Thermal: r.Thermal, Health: r.Health, Cycles: r.Cycles,
	}, nil
}

func (p *Power) Hold(name string) (func(), error) {
	release, err := p.door.HoldIdle(name)
	return release, unsupported(err)
}

func (p *Power) Sleeping() bool { return p.door.Sleeping() }

func (p *Power) Clamshell(held bool) error { return unsupported(p.door.SetClamshellSleep(held)) }

func (p *Power) DisplaySleepNow(ctx context.Context) error {
	// invariant: the display sleeps by pmset displaysleepnow alone, never a
	// display assertion, so no other app's assertion keeps it lit lid closed.
	out, err := p.run.Output(ctx, local.Cmd{Name: p.pmset, Args: []string{"displaysleepnow"}, Timeout: pmsetWait})
	switch {
	case err != nil:
		return err
	case out.ExitCode != 0:
		return errors.New("pmset exited " + strconv.Itoa(out.ExitCode))
	}
	return nil
}

func (p *Power) Watch(ctx context.Context) <-chan awake.Event {
	// invariant: a door with no ticks has no power management, and its watch
	// is closed at once; else each tick compares a reading with the last one and
	// sends one Event per field that moved, so a falling percent ends a session
	// within the second it is posted; Low Power Mode is read by the guard loop's
	// own poll. A watch also starts hearing macOS's will-sleep, so the engine
	// that watches knows from its start when its Mac goes to sleep.
	out := make(chan awake.Event, 8)
	p.door.Sleeping()
	ticks := p.door.Ticks(ctx)
	if ticks == nil {
		close(out)
		return out
	}
	go func() {
		defer close(out)
		last, _ := p.Read()
		for range ticks {
			now, err := p.Read()
			if err != nil {
				continue
			}
			for _, kind := range moved(last, now) {
				select {
				case out <- awake.Event{Kind: kind}:
				case <-ctx.Done():
					return
				}
			}
			last = now
		}
	}()
	return out
}

func moved(a, b awake.Reading) []string {
	var kinds []string
	if a.Thermal != b.Thermal {
		kinds = append(kinds, "thermal")
	}
	if a.Source != b.Source {
		kinds = append(kinds, "source")
	}
	if a.Lid != b.Lid {
		kinds = append(kinds, "lid")
	}
	if a.HasBattery != b.HasBattery || a.Percent != b.Percent {
		kinds = append(kinds, "battery")
	}
	return kinds
}
