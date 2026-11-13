package darwin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const watchEvery = time.Second

var ErrPowerUnsupported = errors.New("no power management here: macOS only")

type Door interface {
	// value: the door to this Mac's power management an adapter is given: an
	// idle assertion, the clamshell flag, a reading, and how a watch learns that
	// the reading moved.
	HoldIdle(name string) (release func(), err error)
	SetClamshellSleep(disabled bool) error
	Read() (Reading, error)
	// invariant: one tick at every period and at every post that the
	// reading may have moved, until ctx ends; nil where there is no power
	// management.
	Ticks(ctx context.Context) <-chan struct{}
	// value: macOS told this Mac it is going to sleep, and it has not woken
	// since.
	Sleeping() bool
}

func ticks(ctx context.Context, every time.Duration, wake <-chan struct{}) <-chan struct{} {
	out := make(chan struct{}, 1)
	go func() {
		defer close(out)
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			case <-wake:
			}
			select {
			case out <- struct{}{}:
			default:
			}
		}
	}()
	return out
}

func NewDoor(env []string) Door {
	vars := map[string]string{}
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok {
			vars[k] = v
		}
	}
	if dir := vars["STATION_POWER_STUB"]; filepath.IsAbs(dir) {
		return &stubPower{dir: dir, calls: filepath.Join(home(vars), ".calls")}
	}
	if !Supported {
		return noPower{}
	}
	return &macPower{}
}

type macPower struct {
	iokit Power
}

func (p *macPower) HoldIdle(name string) (func(), error) { return HoldIdle(name) }

func (p *macPower) SetClamshellSleep(disabled bool) error {
	return SetClamshellSleep(disabled)
}

func (p *macPower) Read() (Reading, error) { return p.iokit.Read(), nil }

func (p *macPower) Sleeping() bool { return SleepAnnounced() }

func (p *macPower) Ticks(ctx context.Context) <-chan struct{} {
	return ticks(ctx, watchEvery, PowerSourcesChanged(ctx))
}

type noPower struct{}

func (noPower) HoldIdle(string) (func(), error) { return nil, ErrPowerUnsupported }

func (noPower) SetClamshellSleep(bool) error { return ErrPowerUnsupported }

func (noPower) Read() (Reading, error) { return Reading{}, ErrPowerUnsupported }

func (noPower) Ticks(context.Context) <-chan struct{} { return nil }

func (noPower) Sleeping() bool { return false }

func home(vars map[string]string) string {
	if h := vars["HOME"]; h != "" {
		return h
	}
	h, _ := os.UserHomeDir()
	return h
}
