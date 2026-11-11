package remote

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/remote/api"
	"github.com/jterrazz/studio-station/src/remote/contract"
	"github.com/jterrazz/studio-station/src/remote/serve"
)

type Reach struct {
	// value: another Mac, reached through its engine (sshstdio).
	Macs contract.EndpointDialer
	// value: this Mac's engine, which relays a call to one of its VMs (unix).
	Engine contract.EndpointDialer
	// value: where a bare name runs, from this Mac's own records: its On set
	// when it is a VM of this Mac; known false when no record names it.
	Locate func(name string) (ref machine.Ref, known bool)
	// value: the other Mac that holds a VM this Mac has no record of.
	Holder func(ctx context.Context, vm string) (mac string, err error)
	// value: the version Hello answers with.
	Version string
	// value: a Mac's name as the owner reads it (mac-mini); nil keeps it whole.
	Label func(mac string) string
	// value: a call's In in the words a station older than api.WordsSince
	// reads beyond its kinds and lifetimes, its family's own (an update's
	// managers), and a word of it that station has none of; nil leaves
	// the In as api.OlderWords writes it.
	Older func(call string, in json.RawMessage) (out json.RawMessage, lacks string)
}

type Router struct {
	registry *call.Registry
	thisMac  machine.Profile
	reach    Reach
	// value: this Mac's engine, where a call of this machine that needs its
	// state runs; nil runs every call of this machine in this process.
	engine call.Router

	mu   sync.Mutex
	open map[string]*link
}

type link struct {
	// invariant: one connection to a Mac, shared by every call while it
	// lives: the callers that wait on one dial share its answer, a failed dial
	// is never kept, and a connection that broke or lacks a call is retired,
	// closed once the calls still riding it end.
	done    chan struct{}
	ep      contract.Endpoint
	err     error
	users   int
	retired bool
}

func NewRouter(registry *call.Registry, thisMac machine.Profile, reach Reach) *Router {
	return &Router{registry: registry, thisMac: thisMac, reach: reach, open: map[string]*link{}}
}

func (r *Router) WithEngine(engine call.Router) *Router {
	// value: the router with this Mac's engine as its third row.
	r.engine = engine
	return r
}

func (r *Router) Invoke(ctx context.Context, target machine.Ref, name string, in, out any, p call.Progress) error {
	t, err := r.resolve(ctx, target)
	if err != nil {
		return err
	}
	if r.local(t) && r.inEngine(name) {
		return r.engine.Invoke(ctx, target, name, in, out, p)
	}
	if r.local(t) {
		v, ran := r.registry.Run(ctx, r.thisMac, name, in, p)
		call.Set(out, v)
		return ran
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return err
	}
	l, raw, err := r.endpoint(ctx, t, name, raw)
	if err != nil {
		return r.named(t, err)
	}
	defer r.release(l)
	answer, err := l.ep.Invoke(ctx, t, name, raw, p)
	if err != nil {
		r.broke(t.Mac(), l, err)
		return r.named(t, r.labeled(t.Mac(), err))
	}
	if decoded := call.Decode(answer, out); decoded != nil {
		e := fail.New(fail.Failed, call.BadOutput, name, map[string]string{"out": name})
		e.Cause = decoded
		return e
	}
	return nil
}

func (r *Router) Watch(ctx context.Context, target machine.Ref, name string, in any, p call.Progress) (<-chan call.Update[any], error) {
	t, err := r.resolve(ctx, target)
	if err != nil {
		return nil, err
	}
	if r.local(t) && r.inEngine(name) {
		return r.engine.Watch(ctx, target, name, in, p)
	}
	if r.local(t) {
		return r.registry.Stream(ctx, r.thisMac, name, in, p)
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	l, raw, err := r.endpoint(ctx, t, name, raw)
	if err != nil {
		return nil, r.named(t, err)
	}
	updates, err := l.ep.Watch(ctx, t, name, raw, p)
	if err != nil {
		r.broke(t.Mac(), l, err)
		r.release(l)
		return nil, err
	}
	return wire(ctx, updates, func() { r.release(l) }), nil
}

func wire(ctx context.Context, updates <-chan call.Update[json.RawMessage], ended func()) <-chan call.Update[any] {
	out := make(chan call.Update[any])
	go func() {
		defer close(out)
		defer ended()
		for u := range updates {
			next := call.Update[any]{Err: u.Err}
			if u.Err == nil {
				next.Value = u.Value
			}
			select {
			case out <- next:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// invariant: the Router's rows, in order: no target or this machine runs in
// this process, a call that needs the engine's state in this Mac's engine
// when it has one; a VM of this Mac rides this Mac's engine; any other machine
// rides the engine of the Mac it is on, which resolves it the same way; a
// name no record here holds is asked of the other Macs, never dialed.
func (r *Router) resolve(ctx context.Context, target machine.Ref) (machine.Ref, error) {
	if target == (machine.Ref{}) || target == r.thisMac.Ref || target.Name == r.thisMac.Ref.Name {
		return machine.Ref{}, nil
	}
	if target.On != "" || r.reach.Locate == nil {
		return target, nil
	}
	t, known := r.reach.Locate(target.Name)
	if known || r.reach.Holder == nil {
		return t, nil
	}
	mac, err := r.reach.Holder(ctx, target.Name)
	if err != nil {
		return machine.Ref{}, err
	}
	return machine.Ref{Name: target.Name, On: mac}, nil
}

func (r *Router) local(t machine.Ref) bool { return t == (machine.Ref{}) }

func (r *Router) inEngine(name string) bool {
	if r.engine == nil {
		return false
	}
	h, ok := r.registry.Lookup(name)
	return ok && h.Meta().Where == call.InEngine
}

func (r *Router) endpoint(ctx context.Context, t machine.Ref, name string, in json.RawMessage) (*link, json.RawMessage, error) {
	// invariant: a call the far Hello lacks is asked again of a fresh
	// connection, so a Mac upgraded since its connection was opened serves it;
	// the In it returns is in the words that far binary reads (ADR-032), and
	// a manager it has no word for is refused as a call it lacks, since it
	// would update nothing.
	l, hello, fresh, err := r.connect(ctx, t)
	if err == nil && !fresh && !slices.Contains(hello.Calls, name) {
		r.drop(t.Mac(), l)
		r.release(l)
		l, hello, _, err = r.connect(ctx, t)
	}
	if err != nil {
		return nil, nil, err
	}
	if !slices.Contains(hello.Calls, name) {
		r.release(l)
		detail := map[string]string{"version": hello.Version, "call": name}
		if r.reach.Label != nil {
			detail["label"] = r.reach.Label(t.Mac())
		}
		return nil, nil, fail.New(fail.FarTooOld, call.FarTooOld, t.Mac(), detail)
	}
	if api.Older(hello.Version, r.reach.Version) {
		var lacks string
		if in = api.OlderWords(in); r.reach.Older != nil {
			in, lacks = r.reach.Older(name, in)
		}
		if lacks != "" {
			r.release(l)
			detail := map[string]string{"version": hello.Version, "call": name + " --manager " + lacks}
			if r.reach.Label != nil {
				detail["label"] = r.reach.Label(t.Mac())
			}
			return nil, nil, fail.New(fail.FarTooOld, call.FarTooOld, t.Mac(), detail)
		}
	}
	return l, in, nil
}

func (r *Router) connect(ctx context.Context, t machine.Ref) (l *link, hello contract.Hello, fresh bool, err error) {
	// invariant: a link it returns is held by the caller until release.
	dialer, mac := r.reach.Macs, t.Mac()
	if t.On != "" && t.On == r.thisMac.Ref.Name {
		dialer = r.reach.Engine
	}
	if dialer == nil {
		return nil, contract.Hello{}, false, fail.New(fail.Unreachable, call.Unreachable, mac, nil)
	}
	r.mu.Lock()
	l, kept := r.open[mac]
	if !kept {
		l = &link{done: make(chan struct{})}
		r.open[mac] = l
	}
	l.users++
	r.mu.Unlock()
	if !kept {
		l.ep, l.err = dialer.Dial(ctx, mac)
		if l.err != nil {
			r.drop(mac, l)
		}
		close(l.done)
	}
	select {
	case <-l.done:
	case <-ctx.Done():
		r.release(l)
		return nil, contract.Hello{}, false, ctx.Err()
	}
	if l.err != nil {
		r.release(l)
		return nil, contract.Hello{}, false, r.labeled(mac, l.err)
	}
	hello, err = l.ep.Hello(ctx)
	if err != nil {
		r.broke(mac, l, err)
		r.release(l)
		return nil, contract.Hello{}, false, r.labeled(mac, err)
	}
	return l, hello, !kept, nil
}

func (r *Router) broke(mac string, l *link, err error) {
	// invariant: only the connection's own end retires it: a far side that
	// answers that one of its VMs is unreachable keeps its connection.
	var e *fail.Error
	if errors.As(err, &e) && (e.Type == fail.Unreachable || e.Code == call.NoStation) && e.Subject == mac {
		r.drop(mac, l)
	}
}

func (r *Router) drop(mac string, l *link) {
	r.mu.Lock()
	if r.open[mac] == l {
		delete(r.open, mac)
	}
	l.retired = true
	r.mu.Unlock()
}

func (r *Router) release(l *link) {
	r.mu.Lock()
	l.users--
	unused := l.retired && l.users == 0 && l.ep != nil
	r.mu.Unlock()
	if unused {
		_ = l.ep.Close()
	}
}

func (r *Router) Hello(ctx context.Context, mac string) (contract.Hello, error) {
	// value: what the far binary says on the connection open now: its version and its calls.
	l, hello, _, err := r.connect(ctx, machine.Ref{Name: mac})
	if err == nil {
		r.release(l)
	}
	return hello, err
}

func (r *Router) Drop(mac string) {
	// value: the connection to mac is not kept: the next call dials it again.
	r.mu.Lock()
	l, ok := r.open[mac]
	r.mu.Unlock()
	if ok {
		r.drop(mac, l)
		r.mu.Lock()
		unused := l.users == 0 && l.ep != nil
		r.mu.Unlock()
		if unused {
			_ = l.ep.Close()
		}
	}
}

func (r *Router) Close() {
	r.mu.Lock()
	open := r.open
	r.open = map[string]*link{}
	r.mu.Unlock()
	for _, l := range open {
		select {
		case <-l.done:
			if l.ep != nil {
				_ = l.ep.Close()
			}
		default:
		}
	}
}

func (r *Router) Served() contract.Hello {
	return serve.Local(r.registry, r.thisMac, r.reach.Version).Hello()
}

func (r *Router) Serve(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (any, error) {
	// invariant: a server answers for its own machine and for its VMs alone: a
	// frame for this machine runs here, one for one of its VMs rides its
	// engine's connection to that VM, and nothing hops to a third machine.
	vm, ok := r.vm(target)
	if !ok {
		return serve.Local(r.registry, r.thisMac, r.reach.Version).Invoke(ctx, target, name, in, p)
	}
	l, in, err := r.endpoint(ctx, vm, name, in)
	if err != nil {
		return nil, err
	}
	defer r.release(l)
	out, err := l.ep.Invoke(ctx, vm, name, in, p)
	if err != nil {
		r.broke(vm.Mac(), l, err)
	}
	return out, err
}

func (r *Router) ServeWatch(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (<-chan call.Update[any], error) {
	vm, ok := r.vm(target)
	if !ok {
		return serve.Local(r.registry, r.thisMac, r.reach.Version).Watch(ctx, target, name, in, p)
	}
	l, in, err := r.endpoint(ctx, vm, name, in)
	if err != nil {
		return nil, err
	}
	updates, err := l.ep.Watch(ctx, vm, name, in, p)
	if err != nil {
		r.broke(vm.Mac(), l, err)
		r.release(l)
		return nil, err
	}
	return wire(ctx, updates, func() { r.release(l) }), nil
}

func (r *Router) vm(target machine.Ref) (machine.Ref, bool) {
	if target.Name == "" || target.Name == r.thisMac.Ref.Name {
		return machine.Ref{}, false
	}
	if r.reach.Locate == nil {
		return machine.Ref{}, false
	}
	if t, _ := r.reach.Locate(target.Name); t.On != r.thisMac.Ref.Name {
		return machine.Ref{}, false
	}
	return machine.Ref{Name: target.Name, On: r.thisMac.Ref.Name}, true
}

func Endpoint(c api.Client) call.Router {
	// value: a station.api/v2 server reached as a Router: each call a frame,
	// each watch its own connection, its updates in their wire form.
	return engineRow{c}
}

type engineRow struct{ c api.Client }

func (e engineRow) Invoke(ctx context.Context, target machine.Ref, name string, in, out any, p call.Progress) error {
	return e.c.Call(ctx, name, target, in, out, p)
}

func (e engineRow) Watch(ctx context.Context, _ machine.Ref, name string, in any, _ call.Progress) (<-chan call.Update[any], error) {
	raw, err := e.c.Watch(ctx, name, in)
	if err != nil {
		return nil, err
	}
	out := make(chan call.Update[any])
	go func() {
		defer close(out)
		for v := range raw {
			select {
			case out <- call.Update[any]{Value: v}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

// value: a VM of another Mac that does not answer is named as the owner
// asked for it: the VM, on that Mac (SPEC §8.1).
func (r *Router) named(t machine.Ref, err error) error {
	var e *fail.Error
	mac := t.Mac()
	if t.Name == mac || mac == r.thisMac.Ref.Name || !errors.As(err, &e) || e.Code != call.Unreachable || e.Subject != mac {
		return err
	}
	detail := map[string]string{"mac": mac}
	for k, v := range e.Detail {
		detail[k] = v
	}
	return fail.New(e.Type, call.VMMacUnreachable, t.Name, detail)
}

// value: a Mac that does not answer, or that names itself in its refusal,
// named as the owner reads it in its words (SPEC §2.5), its full name kept.
func (r *Router) labeled(mac string, err error) error {
	var e *fail.Error
	if r.reach.Label == nil || !errors.As(err, &e) {
		return err
	}
	if (e.Code != call.Unreachable && e.Code != call.NoStation || e.Subject != mac) && e.Detail["mac"] != mac {
		return err
	}
	detail := map[string]string{"label": r.reach.Label(mac)}
	for k, v := range e.Detail {
		detail[k] = v
	}
	return fail.New(e.Type, e.Code, e.Subject, detail)
}
