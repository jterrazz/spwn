package process

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/remote/serve"

	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
)

func activityOf(d *Engine) engine.ActivityOut {
	h := d.History()
	if h == nil {
		return engine.ActivityOut{Machines: []engine.Kept{}}
	}
	out := engine.ActivityOut{Since: h.Since(), Machines: []engine.Kept{}}
	for _, name := range h.Machines() {
		k, _ := h.Of(name)
		out.Machines = append(out.Machines, engine.Kept{Machine: name, Last: k.Last, Fine: k.Series.Fine(), Coarse: k.Series.Coarse()})
	}
	return out
}

func own[In any](d *Engine, name string, run func(ctx context.Context, d *Engine, in In) (any, error)) call.Handler {
	return call.Bind(call.Call[*Engine, In, any]{
		Name: name, Where: call.InEngine, Applies: machine.Anywhere,
		Run: func(ctx context.Context, d *Engine, in In, _ call.Progress) (any, error) {
			out, err := run(ctx, d, in)
			return out, worded(err)
		},
	}, d)
}

// invariant: an engine refusal leaves as the error it is, typed by whether
// the caller named something unknown, its words the engine's.
func worded(err error) error {
	var op *enginecalls.OpError
	if !errors.As(err, &op) {
		return err
	}
	t := fail.Failed
	if op.Refused() {
		t = fail.Refused
	}
	return &fail.Error{Type: t, Code: fail.Code(op.Code), Message: op.Message, Hint: op.Hint}
}

func calls(d *Engine, served []string) []call.Handler {
	out := make([]call.Handler, 0, 12+len(VMVerbs()))
	out = append(out,
		own(d, engine.CallHello, func(_ context.Context, d *Engine, _ struct{}) (any, error) {
			h := d.Hello()
			h.Calls = served
			return h, nil
		}),
		own(d, engine.CallVMs, func(_ context.Context, d *Engine, _ struct{}) (any, error) { return d.VMs(), nil }),
		own(d, engine.CallPorts, func(ctx context.Context, d *Engine, in engine.VMIn) (any, error) { return d.Ports(ctx, in.VM), nil }),
		own(d, engine.CallReserve, func(ctx context.Context, d *Engine, in engine.ReserveIn) (any, error) {
			return d.Reserve(ctx, in.VM, in.GuestPort)
		}),
		own(d, engine.CallRelease, func(ctx context.Context, d *Engine, in engine.ReserveIn) (any, error) {
			return d.Release(ctx, in.VM, in.GuestPort)
		}),
		own(d, engine.CallGuestExec, func(ctx context.Context, d *Engine, in engine.ExecIn) (any, error) {
			ran, err := d.GuestExec(ctx, in.VM, in.Line, in.Stdin)
			return engine.ExecOut(ran), err
		}),
		own(d, engine.CallAccounts, func(ctx context.Context, d *Engine, _ struct{}) (any, error) { return d.Accounts(ctx), nil }),
		own(d, engine.CallStopAll, func(ctx context.Context, d *Engine, _ struct{}) (any, error) { return d.StopAll(ctx), nil }),
		own(d, engine.CallAwakeView, func(_ context.Context, d *Engine, _ struct{}) (any, error) { return d.AwakeView(), nil }),
		own(d, engine.CallKeepAwake, func(_ context.Context, d *Engine, in engine.AwakeIn) (any, error) { return d.Awake(in) }),
		own(d, engine.CallAwakeOff, func(_ context.Context, d *Engine, _ struct{}) (any, error) { return d.Sleep(), nil }),
		own(d, engine.CallActivity, func(_ context.Context, d *Engine, _ struct{}) (any, error) { return activityOf(d), nil }),
		own(d, engine.CallKeepSettings, func(_ context.Context, d *Engine, in engine.KeepSettingsIn) (any, error) {
			d.KeepSettings(in.VM, in.Read)
			return struct{}{}, nil
		}),
		own(d, engine.CallLastSettings, func(_ context.Context, d *Engine, in engine.VMIn) (any, error) {
			read, ok := d.LastSettings(in.VM)
			return engine.LastSettingsOut{Read: read, Kept: ok}, nil
		}),
		own(d, engine.CallRefresh, func(_ context.Context, d *Engine, in engine.PagesIn) (any, error) {
			return struct{}{}, d.Refresh(in.Pages)
		}),
		own(d, engine.CallAgentsHeld, func(ctx context.Context, d *Engine, _ struct{}) (any, error) { return d.AgentsHeld(ctx) }),
	)
	for _, verb := range VMVerbs() {
		out = append(out, own(d, engine.CallVM(verb), func(ctx context.Context, d *Engine, in engine.VMIn) (any, error) {
			return nil, d.VM(ctx, verb, in.VM)
		}))
	}
	return out
}

type Serving struct {
	// value: the registry the engine serves to its clients (the menu bar's
	// actions), run on this Mac; nil serves the engine's own calls alone.
	Registry    *call.Registry
	ThisMachine machine.Profile
	Word        func(e *fail.Error) (message, hint string)
	Sleep       func(ctx context.Context, d time.Duration) error
	// value: the Router a call for another machine rides.
	Far call.Router
	// value: the VM whose guest socket this server is: its guest's calls
	// (engine.ports, engine.reserve, engine.release) are answered as the
	// engine answers that guest; empty serves this Mac's own.
	Guest string
}

func Server(d *Engine, h Serving) serve.Server {
	// invariant: the engine's server: its own calls first, then the registry's,
	// and the watch of its pages; a call of neither is unknown.
	var served []string
	if h.Registry != nil {
		for _, m := range h.Registry.Metas() {
			served = append(served, m.Name)
		}
	}
	byName := map[string]call.Handler{}
	for _, c := range calls(d, served) {
		byName[c.Meta().Name] = c
	}
	if h.Guest != "" {
		for name, op := range map[string]string{engine.CallPorts: "ports", engine.CallReserve: "reserve", engine.CallRelease: "release"} {
			byName[name] = own(d, name, func(ctx context.Context, d *Engine, in engine.ReserveIn) (any, error) {
				return d.guestRequest(ctx, h.Guest, guestMessage{Op: op, Port: in.GuestPort})
			})
		}
	}
	return serve.Server{
		Calls: func(name string) (call.Handler, bool) {
			if c, ok := byName[name]; ok {
				return c, true
			}
			if h.Registry == nil {
				return nil, false
			}
			return h.Registry.Lookup(name)
		},
		Streams: map[string]serve.Feed{engine.CallPage: func(ctx context.Context, raw json.RawMessage, send func(any) error) error {
			var in engine.PagesIn
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &in); err != nil {
					return err
				}
			}
			return worded(d.Stream(ctx, in.Pages, send))
		}, engine.CallEvents: func(ctx context.Context, _ json.RawMessage, send func(any) error) error {
			return d.Events(ctx, send)
		}, engine.CallWatch: func(ctx context.Context, raw json.RawMessage, send func(any) error) error {
			var in engine.WatchIn
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &in); err != nil {
					return err
				}
			}
			return d.Watch(ctx, in.Machines, send)
		}},
		ThisMachine: h.ThisMachine, Word: h.Word, Hint: "station machines", Sleep: h.Sleep, Far: h.Far,
	}
}
