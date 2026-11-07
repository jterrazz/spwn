package agenthooks_test

import (
	"context"
	"testing"

	agentscontract "github.com/jterrazz/studio-station/src/agents/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/settings/adapter/agenthooks"
	"github.com/jterrazz/studio-station/src/settings/contract/contracttest"
)

type hook struct {
	agentscontract.Hooks
	on bool
	// value: what is left of station's hook once the owner took it off
	// some of the app's events: no longer on, still in the file.
	left bool
}

func (h *hook) Install(context.Context) error         { h.on = true; return nil }
func (h *hook) Remove(context.Context) error          { h.on, h.left = false, false; return nil }
func (h *hook) Inspect(context.Context) (bool, error) { return h.on, nil }

func setting() catalog.Setting {
	return catalog.Setting{Name: catalog.AgentHooksSetting, Effect: catalog.AgentHooks{Apps: []string{"writer", "reviewer"}}}
}

func TestApplier(t *testing.T) {
	contracttest.Applier(t, func(*testing.T) contracttest.Opened {
		a := agenthooks.New(func(string) bool { return true },
			agenthooks.Hook{App: "writer", File: "~/.writer/settings.json", Hooks: &hook{}},
			agenthooks.Hook{App: "reviewer", File: "~/.reviewer/config.toml", Hooks: &hook{}})
		return contracttest.Opened{Applier: a, Item: setting()}
	})
}

func TestAnAppNotInstalledIsNeitherHookedNorMissing(t *testing.T) {
	ctx := context.Background()
	writer, reviewer := &hook{}, &hook{}
	a := agenthooks.New(func(app string) bool { return app == "writer" },
		agenthooks.Hook{App: "writer", File: "~/.writer/settings.json", Hooks: writer},
		agenthooks.Hook{App: "reviewer", File: "~/.reviewer/config.toml", Hooks: reviewer})
	if _, err := a.Apply(ctx, setting(), nil); err != nil {
		t.Fatal(err)
	}
	if !writer.on || reviewer.on {
		t.Fatalf("hooked writer %v, reviewer %v; want writer alone", writer.on, reviewer.on)
	}
	if o, _ := a.Observe(ctx, setting()); !o.Present() {
		t.Errorf("Observe = %+v; want present with reviewer not installed", o)
	}
}

func TestOffTakesOutWhatIsLeftOfAHookTheOwnerCutDown(t *testing.T) {
	writer := &hook{left: true}
	a := agenthooks.New(func(string) bool { return true },
		agenthooks.Hook{App: "writer", File: "~/.writer/settings.json", Hooks: writer})
	if _, err := a.Undo(context.Background(), setting()); err != nil {
		t.Fatal(err)
	}
	if writer.left {
		t.Error("off left station's hook on the events the owner did not take it off")
	}
}

func TestAWorkspaceWhereNoAppIsHookedReadsNotAppliedRatherThanBroken(t *testing.T) {
	a := agenthooks.New(func(string) bool { return true },
		agenthooks.Hook{App: "writer", File: "~/.writer/settings.json", Hooks: &hook{}},
		agenthooks.Hook{App: "reviewer", File: "~/.reviewer/config.toml", Hooks: &hook{}})
	o, err := a.Observe(context.Background(), setting())
	if err != nil {
		t.Fatal(err)
	}
	if o.State != catalog.StateAbsent {
		t.Errorf("no app hooked reads %s %q; want absent, the setting never applied", o.State, o.Detail)
	}
}
