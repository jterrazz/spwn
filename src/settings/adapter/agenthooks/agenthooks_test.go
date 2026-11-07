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
}

func (h *hook) Install(context.Context) error         { h.on = true; return nil }
func (h *hook) Remove(context.Context) error          { h.on = false; return nil }
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
