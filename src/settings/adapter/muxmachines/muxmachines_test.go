package muxmachines_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/settings/adapter/muxmachines"
	"github.com/jterrazz/studio-station/src/settings/contract/contracttest"
)

type mux struct{ reached []string }

func (m *mux) Machines(context.Context) ([]string, error) { return slices.Clone(m.reached), nil }

func (m *mux) Add(_ context.Context, name string) error {
	if !slices.Contains(m.reached, name) {
		m.reached = append(m.reached, name)
	}
	return nil
}

func (m *mux) Remove(_ context.Context, name string) error {
	m.reached = slices.DeleteFunc(m.reached, func(n string) bool { return n == name })
	return nil
}

func (*mux) Attach(context.Context, string, string) error { return nil }

func setting() catalog.Setting {
	return catalog.Setting{Name: catalog.MuxMachinesSetting, Effect: catalog.MuxMachines{}}
}

func workspaces(names ...string) func(context.Context) ([]string, error) {
	return func(context.Context) ([]string, error) { return names, nil }
}

func TestApplier(t *testing.T) {
	contracttest.Applier(t, func(*testing.T) contracttest.Opened {
		return contracttest.Opened{Applier: muxmachines.New(&mux{}, workspaces("linux-1", "linux-2")), Item: setting()}
	})
}

func TestOneWorkspaceTheHostDoesNotReachIsAGapAndOffLeavesTheOwnersMachines(t *testing.T) {
	ctx := context.Background()
	multiplexer := &mux{}
	_ = multiplexer.Add(ctx, "linux-1")
	_ = multiplexer.Add(ctx, "vps")
	a := muxmachines.New(multiplexer, workspaces("linux-1", "linux-2"))
	o, err := a.Observe(ctx, setting())
	if err != nil || o.State != catalog.StateDegraded {
		t.Fatalf("Observe = %+v (%v); want degraded, linux-2 not reached", o, err)
	}
	if _, err := a.Apply(ctx, setting(), nil); err != nil {
		t.Fatal(err)
	}
	if got := multiplexer.reached; !slices.Equal(got, []string{"linux-1", "vps", "linux-2"}) {
		t.Errorf("reached %q, want linux-2 added", got)
	}
	if _, err := a.Undo(ctx, setting()); err != nil {
		t.Fatal(err)
	}
	if got := multiplexer.reached; !slices.Equal(got, []string{"vps"}) {
		t.Errorf("reached %q once off, want the owner's vps alone", got)
	}
}
