package fake

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract"
)

type Mac struct {
	Registry *call.Registry
	Profile  machine.Profile
	Version  string
	// value: the calls of the registry, and the abilities of their inputs,
	// that Mac's older binary does not serve: its Hello leaves them out.
	Lacks []string
	// value: how that Mac's binary words its errors; nil words only what
	// is already worded.
	Word call.Wording
	// value: that Mac's own routing, for a call addressed to one of its VMs;
	// nil runs every call on the Mac itself.
	Relay Relay
	// value: where that Mac keeps a gesture it runs on itself, as its
	// serving keeps it; nil keeps none.
	Kept call.Keeper
	// value: what that Mac's binary sends in place of each answer it
	// encoded, as a broken or hostile binary sends it; nil sends it as
	// encoded.
	Answers func(json.RawMessage) json.RawMessage
}

type Relay interface {
	Invoke(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (any, error)
	Watch(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (<-chan call.Update[any], error)
}

type Dialer struct {
	// invariant: a second Mac in this process: its calls run on its own
	// registry, every value through the wire form, as over ssh. A connection
	// keeps the Mac it was dialed to, as a far process keeps its binary, and
	// breaks when that Mac goes down or when it is closed.
	mu     sync.Mutex
	macs   map[string]Mac
	down   map[string]bool
	dialed []string
}

func New() *Dialer { return &Dialer{macs: map[string]Mac{}, down: map[string]bool{}} }

func (d *Dialer) Serve(name string, m Mac) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.macs[name] = m
	delete(d.down, name)
}

func (d *Dialer) Down(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.down[name] = true
}

func (d *Dialer) Dial(_ context.Context, mac string) (contract.Endpoint, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dialed = append(d.dialed, mac)
	m, ok := d.macs[mac]
	if !ok || d.down[mac] {
		return nil, fail.New(fail.Unreachable, call.Unreachable, mac, nil)
	}
	return endpoint{d: d, name: mac, m: m, closed: new(atomic.Bool)}, nil
}

func (d *Dialer) Dialed() []string {
	// value: every name Dial was asked for, in order.
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.dialed)
}

type endpoint struct {
	d      *Dialer
	name   string
	m      Mac
	closed *atomic.Bool
}

func (e endpoint) broken() error {
	e.d.mu.Lock()
	defer e.d.mu.Unlock()
	if e.d.down[e.name] || e.closed.Load() {
		return fail.New(fail.Unreachable, call.Unreachable, e.name, nil)
	}
	return nil
}

func (e endpoint) Hello(context.Context) (contract.Hello, error) {
	if err := e.broken(); err != nil {
		return contract.Hello{}, err
	}
	h := contract.Hello{Version: e.m.Version, Calls: []string{}}
	for _, meta := range e.m.Registry.Metas() {
		if !slices.Contains(e.m.Lacks, meta.Name) {
			h.Calls = append(h.Calls, meta.Name)
		}
		for _, a := range meta.Abilities {
			if !slices.Contains(e.m.Lacks, a) {
				h.Abilities = append(h.Abilities, a)
			}
		}
	}
	return h, nil
}

func (e endpoint) Invoke(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (json.RawMessage, error) {
	// invariant: that Mac reads every request as engine attach reads it at
	// the end of ssh: marked far.
	ctx = call.WithFar(ctx)
	if err := e.broken(); err != nil {
		return nil, err
	}
	if target.On != "" && e.m.Relay != nil {
		out, err := e.m.Relay.Invoke(ctx, target, name, in, p)
		if err != nil {
			return nil, e.over(err)
		}
		return e.encoded(out)
	}
	v, err := e.m.Registry.Input(name, in)
	if err != nil {
		return nil, e.over(err)
	}
	ctx = call.Recording(ctx)
	out, err := e.m.Registry.Run(ctx, e.m.Profile, name, v, p)
	if h, ok := e.m.Registry.Lookup(name); ok {
		e.m.Kept.Keep(ctx, e.m.Profile, e.m.Profile.Ref, h.Meta(), v, out, err)
	}
	if err != nil {
		return nil, e.over(err)
	}
	return e.encoded(out)
}

func (e endpoint) Watch(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (<-chan call.Update[json.RawMessage], error) {
	ctx = call.WithFar(ctx)
	if err := e.broken(); err != nil {
		return nil, err
	}
	updates, err := e.stream(ctx, target, name, in, p)
	if err != nil {
		return nil, e.over(err)
	}
	out := make(chan call.Update[json.RawMessage])
	go func() {
		defer close(out)
		for u := range updates {
			if err := e.broken(); err != nil {
				out <- call.Update[json.RawMessage]{Err: err}
				return
			}
			next := call.Update[json.RawMessage]{Err: e.over(u.Err)}
			if u.Err == nil {
				next.Value, next.Err = e.encoded(u.Value)
			}
			out <- next
		}
	}()
	return out, nil
}

func (e endpoint) stream(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (<-chan call.Update[any], error) {
	if target.On != "" && e.m.Relay != nil {
		return e.m.Relay.Watch(ctx, target, name, in, p)
	}
	v, err := e.m.Registry.Input(name, in)
	if err != nil {
		return nil, err
	}
	return e.m.Registry.Stream(ctx, e.m.Profile, name, v, p)
}

func (e endpoint) encoded(out any) (json.RawMessage, error) {
	// invariant: what crosses from one Mac to the other is an answer: the
	// world build's tripwire reads it as the serve loop reads a frame.
	data, err := call.Encode(out, e.word())
	if err != nil {
		return nil, err
	}
	secret.Tripwire(data)
	if e.m.Answers != nil {
		data = e.m.Answers(data)
	}
	return data, nil
}

func (e endpoint) Close() error {
	e.closed.Store(true)
	return nil
}

func worded(err error) *fail.Error {
	var e *fail.Error
	if errors.As(err, &e) {
		return e
	}
	return &fail.Error{Message: err.Error(), Exit: fail.ExitFailure}
}

func (e endpoint) word() call.Wording {
	if e.m.Word != nil {
		return e.m.Word
	}
	return worded
}

func (e endpoint) over(err error) error {
	if err == nil {
		return nil
	}
	w := call.ToWire(err, e.word())
	secret.TripwireJSON(w)
	return w.Err()
}
