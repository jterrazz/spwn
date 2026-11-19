package macos

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/awake/contract"
	"github.com/jterrazz/studio-station/apps/engine/awake/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type door struct {
	mu       sync.Mutex
	r        darwin.Reading
	none     bool
	tick     chan struct{}
	sleeping bool
}

func (d *door) Sleeping() bool { return d.sleeping }

func (d *door) HoldIdle(string) (func(), error) {
	if d.none {
		return nil, darwin.ErrPowerUnsupported
	}
	return func() {}, nil
}

func (d *door) SetClamshellSleep(bool) error {
	if d.none {
		return darwin.ErrPowerUnsupported
	}
	return nil
}

func (d *door) Read() (darwin.Reading, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.none {
		return darwin.Reading{}, darwin.ErrPowerUnsupported
	}
	return d.r, nil
}

func (d *door) Ticks(context.Context) <-chan struct{} {
	if d.none {
		return nil
	}
	return d.tick
}

type pmset struct{ ran []local.Cmd }

func (p *pmset) Output(_ context.Context, c local.Cmd) (local.Output, error) {
	p.ran = append(p.ran, c)
	return local.Output{}, nil
}

func reading(r awake.Reading) darwin.Reading {
	return darwin.Reading{Lid: r.Lid, Source: r.Source, HasBattery: r.HasBattery, Percent: r.Percent, Charging: r.Charging, LowPower: r.LowPower, Thermal: r.Thermal}
}

func TestTheMacOSPowerKeepsTheContract(t *testing.T) {
	contracttest.Power(t, func(_ *testing.T, r awake.Reading, none bool) contract.Power {
		return New(&door{r: reading(r), none: none}, &pmset{}, "")
	})
}

func TestAMacToldItGoesToSleepSaysSo(t *testing.T) {
	if New(&door{r: reading(contracttest.Laptop())}, &pmset{}, "").Sleeping() {
		t.Fatal("a Mac macOS told nothing reads sleeping")
	}
	if !New(&door{r: reading(contracttest.Laptop()), sleeping: true}, &pmset{}, "").Sleeping() {
		t.Fatal("a Mac told it goes to sleep does not read sleeping")
	}
}

func TestTheDisplaySleepsByThePmsetUnderTheRoot(t *testing.T) {
	run := &pmset{}
	if err := New(&door{}, run, "/sandbox").DisplaySleepNow(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(run.ran) != 1 || run.ran[0].Name != "/sandbox/usr/bin/pmset" || !slices.Equal(run.ran[0].Args, []string{"displaysleepnow"}) {
		t.Fatalf("ran %+v", run.ran)
	}
}

func TestUnpluggingTheLaptopIsOneSourceEvent(t *testing.T) {
	d := &door{r: reading(contracttest.Laptop()), tick: make(chan struct{})}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	events := New(d, &pmset{}, "").Watch(ctx)
	d.tick <- struct{}{}
	d.mu.Lock()
	d.r.Source = "battery"
	d.mu.Unlock()
	d.tick <- struct{}{}
	select {
	case e := <-events:
		if e.Kind != "source" {
			t.Fatalf("event %q, want source", e.Kind)
		}
	case <-ctx.Done():
		t.Fatal("the watch never heard the unplug")
	}
}
