package process

import (
	"context"
	"errors"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

// invariant: the key agent opens at the engine's start, before the VMs are
// adopted. A socket another process holds is waited on, never shared: the
// engine reads it held, by whom, and binds the socket the moment its holder
// lets it go. A Mac that cannot keep a device key serves none and reads none.
func (d *Engine) openAgent(ctx context.Context, note func(line string)) {
	if d.o.Agent == nil {
		return
	}
	serve, held, ok := d.tryAgent(ctx, note)
	if ok {
		go d.keepAgent(ctx, note, serve, held)
	}
}

func (d *Engine) tryAgent(ctx context.Context, note func(line string)) (func(context.Context) error, *engine.AgentHeldError, bool) {
	serve, err := d.o.Agent(ctx)
	var held *engine.AgentHeldError
	var none *contract.NoDeviceKeyError
	switch {
	case err == nil:
		d.agentIs(model.Agent{State: model.AgentServed})
		return serve, nil, true
	case errors.As(err, &held):
		held.Holder.State = model.AgentHeld
		if d.agentIs(held.Holder) {
			d.noteAgent(note, err)
		}
		return nil, held, true
	case errors.As(err, &none):
		d.noteAgent(note, err)
		return nil, nil, false
	}
	d.agentIs(model.Agent{State: model.AgentFailed, Detail: err.Error()})
	d.noteAgent(note, err)
	return nil, nil, false
}

func (d *Engine) keepAgent(ctx context.Context, note func(line string), serve func(context.Context) error, held *engine.AgentHeldError) {
	for held != nil {
		if held.Freed(ctx) != nil {
			return
		}
		var ok bool
		if serve, held, ok = d.tryAgent(ctx, note); !ok {
			return
		}
	}
	if err := serve(ctx); err != nil && ctx.Err() == nil {
		d.agentIs(model.Agent{State: model.AgentFailed, Detail: err.Error()})
		d.noteAgent(note, err)
	}
}

func (d *Engine) agentIs(a model.Agent) (changed bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	changed = d.agent != a
	d.agent = a
	return changed
}

func (d *Engine) noteAgent(note func(line string), err error) {
	if note != nil {
		note(d.h.Now().UTC().Format(time.RFC3339) + " agent: " + err.Error())
	}
}
