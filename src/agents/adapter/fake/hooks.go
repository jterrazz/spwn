package fake

import (
	"context"
	"encoding/json"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/agents/contract"
	"github.com/jterrazz/studio-station/src/world"
)

type Hooks struct {
	// invariant: the hook of one app in one VM of the world; its payload is
	// the event itself, as the world's agent events write it.
	inVM
	app agents.App
}

var _ contract.Hooks = (*Hooks)(nil)

func NewHooks(app agents.App) *Hooks { return &Hooks{inVM: alone(), app: app} }

func HooksOf(w *world.World, vm string, app agents.App) *Hooks {
	return &Hooks{inVM: inVM{w: w, vm: vm}, app: app}
}

func (h *Hooks) App() agents.App { return h.app }

func (h *Hooks) Launch() []string { return []string{string(h.app)} }

func (h *Hooks) set(on bool) {
	h.w.Lock()
	defer h.w.Unlock()
	held := h.agents()
	if held.Hooks == nil {
		held.Hooks = map[agents.App]bool{}
	}
	if on {
		held.Hooks[h.app] = true
	} else {
		delete(held.Hooks, h.app)
	}
}

func (h *Hooks) Install(context.Context) error {
	h.set(true)
	return nil
}

func (h *Hooks) Remove(context.Context) error {
	h.set(false)
	return nil
}

func (h *Hooks) Inspect(context.Context) (bool, error) {
	h.w.Lock()
	defer h.w.Unlock()
	return h.agents().Hooks[h.app], nil
}

func (h *Hooks) Event(_ context.Context, payload []byte) (agents.Event, bool) {
	var e agents.Event
	if json.Unmarshal(payload, &e) != nil || e.Session == "" {
		return agents.Event{}, false
	}
	switch e.Kind {
	case agents.Began, agents.Prompted, agents.UseStarting, agents.UseDone, agents.Notified, agents.Stopped, agents.Ended, agents.TurnDone:
	default:
		return agents.Event{}, false
	}
	e.App = h.app
	if e.At.IsZero() {
		h.w.Lock()
		e.At = h.w.Now
		h.w.Unlock()
	}
	return e, true
}
