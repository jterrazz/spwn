package call

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type Where string

const (
	AtTarget  Where = "at-target"
	InEngine  Where = "in-engine"
	InProcess Where = "in-process"
)

const (
	NotForThisMachine fail.Code = "section.not-for-this-machine"
	// value: a call whose need this Mac lacks the provider of.
	NeedMissing fail.Code = "call.needs"
	BadInput    fail.Code = "call.bad-input"
	BadOutput   fail.Code = "call.bad-output"
	// value: a line a server cannot read as a request frame.
	BadFrame fail.Code = "call.bad-frame"
)

type Progress interface {
	Step(task, detail string)
	Done(task string, err error)
}

const Asks = "asks" // value: the first line of a step's detail, whose lines are why it runs, when the step hands the terminal to a program which asks the owner (sudo's password): a surface says the step once, as a line, and draws nothing over the prompt.

type Call[D, In, Out any] struct {
	Name string
	// invariant: empty for a call of no per-machine section, which its
	// Applies alone gates.
	Section section.ID
	Where   Where
	Applies machine.Applies
	// value: what the call cannot run without, judged by the gate after
	// Applies, from the Profile's one presence read.
	Needs   []machine.Need
	Confirm confirm.Action
	Errors  []fail.Code
	Run     func(ctx context.Context, d D, in In, p Progress) (Out, error)
}

type Update[Out any] struct {
	Value Out
	Err   error
}

type Watch[D, In, Out any] struct {
	// invariant: a read that fills progressively: each Update is the whole
	// value so far, and its producer closes the channel when done.
	Name    string
	Section section.ID
	Where   Where
	Applies machine.Applies
	Needs   []machine.Need
	Errors  []fail.Code
	Run     func(ctx context.Context, d D, in In, p Progress) (<-chan Update[Out], error)
}

type Meta struct {
	Name    string
	Section section.ID
	Where   Where
	Applies machine.Applies
	Needs   []machine.Need
	Confirm confirm.Action
	Errors  []fail.Code
	In      string
	Out     string
	Watch   bool
}

type Handler interface {
	Meta() Meta
	Invoke(ctx context.Context, target machine.Profile, in json.RawMessage, p Progress) (json.RawMessage, error)
	Watch(ctx context.Context, target machine.Profile, in json.RawMessage, p Progress) (<-chan Update[json.RawMessage], error)
}

type Router interface {
	Invoke(ctx context.Context, target machine.Ref, call string, in, out any, p Progress) error
	// invariant: each update's value is the call's Out when it ran in this
	// process, its wire form (json.RawMessage) when it ran elsewhere.
	Watch(ctx context.Context, target machine.Ref, call string, in any, p Progress) (<-chan Update[any], error)
}

func Invoke[D, In, Out any](ctx context.Context, r Router, target machine.Ref, c Call[D, In, Out], in In, p Progress) (Out, error) {
	var out Out
	err := r.Invoke(ctx, target, c.Name, in, &out, p)
	return out, err
}

func Read[D, In, Out any](ctx context.Context, r Router, target machine.Ref, w Watch[D, In, Out], in In, p Progress) (Out, error) {
	var out Out
	err := r.Invoke(ctx, target, w.Name, in, &out, p)
	return out, err
}

func Follow[D, In, Out any](ctx context.Context, r Router, target machine.Ref, w Watch[D, In, Out], in In, p Progress) (<-chan Update[Out], error) {
	updates, err := r.Watch(ctx, target, w.Name, in, p)
	if err != nil {
		return nil, err
	}
	out := make(chan Update[Out])
	go func() {
		defer close(out)
		for u := range updates {
			next := Update[Out]{Err: u.Err}
			if u.Err == nil {
				next.Value, next.Err = valueOf[Out](w.Name, u.Value)
			}
			select {
			case out <- next:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

func valueOf[Out any](name string, v any) (Out, error) {
	switch v := v.(type) {
	case json.RawMessage:
		var out Out
		if err := Decode(v, &out); err != nil {
			e := fail.New(fail.Failed, BadOutput, name, map[string]string{"out": reflect.TypeFor[Out]().Name()})
			e.Cause = err
			return out, e
		}
		return out, nil
	case Out:
		return v, nil
	}
	var zero Out
	return zero, fail.New(fail.Failed, BadOutput, name, map[string]string{"out": reflect.TypeFor[Out]().Name()})
}

func Bind[D, In, Out any](c Call[D, In, Out], d D) Handler {
	return bound[D, In, Out]{meta: Meta{
		Name: c.Name, Section: c.Section, Where: c.Where, Applies: c.Applies, Needs: c.Needs, Confirm: c.Confirm, Errors: c.Errors,
		In: reflect.TypeFor[In]().Name(), Out: reflect.TypeFor[Out]().Name(),
	}, runFn: c.Run, deps: d}
}

func BindWatch[D, In, Out any](w Watch[D, In, Out], d D) Handler {
	return watched[D, In, Out]{meta: Meta{
		Name: w.Name, Section: w.Section, Where: w.Where, Applies: w.Applies, Needs: w.Needs, Errors: w.Errors,
		In: reflect.TypeFor[In]().Name(), Out: reflect.TypeFor[Out]().Name(), Watch: true,
	}, runFn: w.Run, deps: d}
}

type bound[D, In, Out any] struct {
	meta  Meta
	runFn func(context.Context, D, In, Progress) (Out, error)
	deps  D
}

func (b bound[D, In, Out]) Meta() Meta { return b.meta }

func (b bound[D, In, Out]) Invoke(ctx context.Context, target machine.Profile, raw json.RawMessage, p Progress) (json.RawMessage, error) {
	in, err := admit[In](b.meta, target, raw)
	if err != nil {
		return nil, err
	}
	out, err := b.runFn(ctx, b.deps, in, orSilent(p))
	if err != nil {
		return nil, err
	}
	return encode(b.meta, out)
}

func (b bound[D, In, Out]) Watch(ctx context.Context, target machine.Profile, raw json.RawMessage, p Progress) (<-chan Update[json.RawMessage], error) {
	out, err := b.Invoke(ctx, target, raw, p)
	if err != nil {
		return nil, err
	}
	once := make(chan Update[json.RawMessage], 1)
	once <- Update[json.RawMessage]{Value: out}
	close(once)
	return once, nil
}

type watched[D, In, Out any] struct {
	meta  Meta
	runFn func(context.Context, D, In, Progress) (<-chan Update[Out], error)
	deps  D
}

func (w watched[D, In, Out]) Meta() Meta { return w.meta }

func (w watched[D, In, Out]) Watch(ctx context.Context, target machine.Profile, raw json.RawMessage, p Progress) (<-chan Update[json.RawMessage], error) {
	in, err := admit[In](w.meta, target, raw)
	if err != nil {
		return nil, err
	}
	updates, err := w.runFn(ctx, w.deps, in, orSilent(p))
	if err != nil {
		return nil, err
	}
	out := make(chan Update[json.RawMessage])
	go func() {
		defer close(out)
		for u := range updates {
			next := Update[json.RawMessage]{Err: u.Err}
			if u.Err == nil {
				next.Value, next.Err = encode(w.meta, u.Value)
			}
			select {
			case out <- next:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

func (w watched[D, In, Out]) Invoke(ctx context.Context, target machine.Profile, raw json.RawMessage, p Progress) (json.RawMessage, error) {
	updates, err := w.Watch(ctx, target, raw, p)
	if err != nil {
		return nil, err
	}
	var last json.RawMessage
	for u := range updates {
		if u.Err != nil {
			return nil, u.Err
		}
		last = u.Value
	}
	return last, ctx.Err()
}

// invariant: the gate every call passes before it runs, in this order: the
// section table of the target's kind, the call's own rule, its needs, its
// input.
func admit[In any](m Meta, target machine.Profile, raw json.RawMessage) (In, error) {
	var in In
	if err := gate(m, target); err != nil {
		return in, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return in, nil
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		e := fail.New(fail.Refused, BadInput, m.Name, map[string]string{"in": m.In})
		e.Cause = err
		return in, e
	}
	return in, nil
}

func Gate(m Meta, target machine.Profile) error { return gate(m, target) }

func gate(m Meta, target machine.Profile) error {
	if m.Section != "" {
		if v := section.Shown(m.Section)(target); !v.OK {
			return fail.New(fail.NotForThisMachine, NotForThisMachine, target.Ref.Name, map[string]string{
				"call": m.Name, "section": string(m.Section), "kind": string(target.Kind), "os": string(target.System.OS), "why": v.Why, "fix": v.Fix,
			})
		}
	}
	if m.Applies != nil {
		if v := m.Applies(target); !v.OK {
			return fail.New(fail.NotForThisMachine, NotForThisMachine, target.Ref.Name, map[string]string{
				"call": m.Name, "why": v.Why, "fix": v.Fix,
			})
		}
	}
	for _, n := range m.Needs {
		if p, lacks := target.Caps.Lacks(n); lacks {
			return fail.New(fail.Refused, NeedMissing, p.Name, map[string]string{"call": m.Name, "need": string(n), "item": p.Item})
		}
	}
	return nil
}

func encode[Out any](m Meta, out Out) (json.RawMessage, error) {
	raw, err := json.Marshal(out)
	if err != nil {
		e := fail.New(fail.Failed, BadOutput, m.Name, map[string]string{"out": m.Out})
		e.Cause = err
		return nil, e
	}
	return raw, nil
}

type silent struct{}

func (silent) Step(string, string) {}
func (silent) Done(string, error)  {}
func orSilent(p Progress) Progress {
	if p == nil {
		return silent{}
	}
	return p
}

type Registry struct {
	// invariant: built once in wire from every family's All(d); a name is
	// claimed by one call only.
	byName map[string]Handler
}

func NewRegistry(families ...[]Handler) (*Registry, error) {
	r := &Registry{byName: map[string]Handler{}}
	for _, family := range families {
		for _, h := range family {
			name := h.Meta().Name
			if _, taken := r.byName[name]; taken {
				return nil, fmt.Errorf("call: two calls are named %q", name)
			}
			r.byName[name] = h
		}
	}
	return r, nil
}

func (r *Registry) Lookup(name string) (Handler, bool) {
	h, ok := r.byName[name]
	return h, ok
}

func Updates[Out any](ctx context.Context, values <-chan Out) <-chan Update[Out] {
	// value: a stream of whole values as a watch's updates.
	out := make(chan Update[Out])
	go func() {
		defer close(out)
		for v := range values {
			select {
			case out <- Update[Out]{Value: v}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func (r *Registry) Metas() []Meta {
	out := make([]Meta, 0, len(r.byName))
	for _, h := range r.byName {
		out = append(out, h.Meta())
	}
	slices.SortFunc(out, func(a, b Meta) int { return strings.Compare(a.Name, b.Name) })
	return out
}
