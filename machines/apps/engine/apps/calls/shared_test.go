package calls_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
)

// value: the engine's clock as a story moves it; a budget never runs out.
type moved struct {
	mu sync.Mutex
	at time.Duration
}

func (c *moved) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Unix(0, 0).Add(c.at)
}

func (*moved) Sleep(ctx context.Context, _ time.Duration) error {
	<-ctx.Done()
	return ctx.Err()
}

func (c *moved) pass(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at += d
}

// invariant: the engine's pool lives as long as the engine; an app the
// owner installs with its manager, outside the engine, is listed by the
// next apps.list once the shared reading outlived the pool's keep.
func TestAnAppInstalledOutsideTheEngineIsListedOnceTheSharedReadingOutlivedItsKeep(t *testing.T) {
	tool := catalog.App{Name: "tool", Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.CaskInstall{Token: "tool"}}}}
	casks := fake.New(catalog.MethodCask, &fake.State{Installed: map[string]string{}})
	clock := &moved{}
	pool := lanes.NewPool(context.Background(), clock, 16, 1)
	pool.Keep(10 * time.Minute)
	shelf := calls.NewShelf(machine{}, catalog.Catalog{Apps: []catalog.App{tool}}, nil, calls.ShelfOptions{ThisMachine: catalog.Mac, Root: "/", Installers: contract.Installers{catalog.MethodCask: casks}, Pool: pool})
	state := func() catalog.State {
		reports, err := shelf.Inventory(context.Background())
		if err != nil || len(reports) != 1 {
			t.Fatalf("inventory: %v %v", reports, err)
		}
		return reports[0].State
	}
	if got := state(); got != catalog.StateAbsent {
		t.Fatalf("tool reads %s before its install, want absent", got)
	}
	if err := casks.Install(context.Background(), []catalog.Install{tool.Installs[0].Install}, nil); err != nil {
		t.Fatal(err)
	}
	clock.pass(time.Minute)
	if got := state(); got != catalog.StateAbsent {
		t.Errorf("a minute after the first read tool reads %s, want the shared reading", got)
	}
	clock.pass(9 * time.Minute)
	if got := state(); got != catalog.StateOK {
		t.Errorf("once the reading outlived the keep, tool reads %s, want installed", got)
	}
}
