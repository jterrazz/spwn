package agenthooks

import (
	"context"
	"errors"
	"slices"
	"strings"

	agentscontract "github.com/jterrazz/studio-station/src/agents/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/settings"
	"github.com/jterrazz/studio-station/src/settings/contract"
)

type Hook struct {
	// value: the app whose agent the hook reports, by catalogue name, the
	// file its hook lives in, as the owner reads it, and its strategy.
	App   string
	File  string
	Hooks agentscontract.Hooks
}

type Applier struct {
	// invariant: an app not installed here is never hooked, and its hook
	// is not missing.
	hooks []Hook
	has   func(app string) bool
}

var _ contract.Applier = (*Applier)(nil)

func New(has func(app string) bool, hooks ...Hook) *Applier {
	return &Applier{hooks: hooks, has: has}
}

func (*Applier) Kind() catalog.EffectKind { return catalog.EffectAgentHooks }

func (a *Applier) of(it catalog.Setting) ([]Hook, error) {
	e, ok := it.Effect.(catalog.AgentHooks)
	if !ok {
		return nil, errors.New(it.Name + " is not an agent hook station installs")
	}
	var out []Hook
	for _, h := range a.hooks {
		if slices.Contains(e.Apps, h.App) {
			out = append(out, h)
		}
	}
	return out, nil
}

func (a *Applier) installed(it catalog.Setting) ([]Hook, error) {
	hooks, err := a.of(it)
	return slices.DeleteFunc(hooks, func(h Hook) bool { return a.has == nil || !a.has(h.App) }), err
}

func (a *Applier) Observe(ctx context.Context, it catalog.Setting) (settings.Observation, error) {
	hooks, err := a.installed(it)
	if err != nil {
		return settings.Observation{}, err
	}
	var files, hooked, missing []string
	o := settings.Observation{}
	for _, h := range hooks {
		installed, err := h.Hooks.Inspect(ctx)
		if err != nil {
			return settings.Observation{}, err
		}
		files = append(files, h.File)
		if installed {
			hooked = append(hooked, h.App)
			continue
		}
		missing = append(missing, h.App)
		o.Before = append(o.Before, settings.KeyValue{Key: h.App, Value: "no hook"})
		o.After = append(o.After, settings.KeyValue{Key: h.App, Value: "hooked"})
	}
	o.Where = strings.Join(files, ", ")
	if len(hooked) > 0 {
		o.Digest = "hooked:" + strings.Join(hooked, ",")
	}
	switch {
	case len(hooks) == 0 || len(hooked) == 0:
		o.State = catalog.StateAbsent
	case len(missing) > 0:
		o.State, o.Detail = catalog.StateDegraded, "no hook in "+strings.Join(missing, ", ")
	default:
		o.State, o.Detail, o.Before, o.After = catalog.StateOK, "hooked: "+strings.Join(hooked, ", "), nil, nil
	}
	return o, nil
}

func (a *Applier) Apply(ctx context.Context, it catalog.Setting, _ map[string]string) (settings.Change, error) {
	hooks, err := a.installed(it)
	if err != nil {
		return settings.Change{}, err
	}
	var c settings.Change
	for _, h := range hooks {
		installed, err := h.Hooks.Inspect(ctx)
		if err != nil {
			return c, err
		}
		if installed {
			c.Steps = append(c.Steps, settings.ChangeStep{Label: h.App, State: catalog.StateOK, Detail: "hook already there"})
			continue
		}
		if err := h.Hooks.Install(ctx); err != nil {
			return c, err
		}
		c.Steps = append(c.Steps, settings.ChangeStep{Label: h.App, State: catalog.StateOK, Detail: "hook installed in " + h.File})
	}
	return c, nil
}

func (a *Applier) Undo(ctx context.Context, it catalog.Setting) (settings.Change, error) {
	// invariant: off takes station's hook out of every app's file, the app
	// installed or not, what is left of a hook the owner cut down included,
	// and leaves the owner's own hooks.
	hooks, err := a.of(it)
	if err != nil {
		return settings.Change{}, err
	}
	var c settings.Change
	for _, h := range hooks {
		installed, err := h.Hooks.Inspect(ctx)
		if err != nil {
			return c, err
		}
		if err := h.Hooks.Remove(ctx); err != nil {
			return c, err
		}
		if !installed {
			continue
		}
		c.Steps = append(c.Steps, settings.ChangeStep{Label: h.App, State: catalog.StateOK, Detail: "hook removed from " + h.File})
	}
	return c, nil
}
