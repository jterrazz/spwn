package call

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

const (
	Unknown fail.Code = "call.unknown"
	// value: a machine that does not answer, or whose station API does not.
	Unreachable fail.Code = "remote.unreachable"
	// value: a VM whose Mac does not answer, so nothing reaches it.
	VMMacUnreachable fail.Code = "remote.vm-mac-unreachable"
	// value: a far binary that serves no such call, or on an older release
	// than this station's.
	FarTooOld fail.Code = "remote.far-too-old"
	// value: a caller behind the release of the station it reaches: refused
	// by the far station, or here before any call is sent when the far
	// station is on a newer release.
	CallerTooOld fail.Code = "remote.caller-too-old"
	// value: a Mac whose ssh answers and that has no station to answer it.
	NoStation fail.Code = "remote.no-station"
	// value: a Mac name that is no ssh word, or that ssh reads as an option,
	// refused before any ssh runs.
	BadMacName fail.Code = "remote.bad-mac-name"
	// value: a bare name that VMs of more than one other Mac bear, refused
	// before anything is dialed, never guessed.
	AmbiguousName fail.Code = "remote.ambiguous-name"
)

func Ambiguous(vm string, macs, silent []string) *fail.Error {
	// value: the refusal of a bare name, each Mac that holds it in name
	// order, so its words name every full address, and apart the ones that
	// did not answer and held it at their last answer.
	detail := map[string]string{"on": strings.Join(slices.Sorted(slices.Values(macs)), " ")}
	if len(silent) > 0 {
		detail["silent"] = strings.Join(slices.Sorted(slices.Values(silent)), " ")
	}
	return fail.New(fail.Refused, AmbiguousName, vm, detail)
}

type local interface {
	run(ctx context.Context, target machine.Profile, in any, p Progress) (any, error)
	stream(ctx context.Context, target machine.Profile, in any, p Progress) (<-chan Update[any], error)
	input(raw json.RawMessage) (any, error)
}

func (r *Registry) Input(name string, raw json.RawMessage) (any, error) {
	// value: a frame's input as the call's own In, for Run or Stream.
	h, err := r.local(name)
	if err != nil {
		return nil, err
	}
	return h.input(raw)
}

func (r *Registry) Asks(name string, raw json.RawMessage) []string {
	// value: the abilities a frame's input asks of the binary that runs it;
	// none for a call this registry does not hold or an input it cannot
	// read, which the far side refuses itself.
	v, err := r.Input(name, raw)
	if err != nil {
		return nil
	}
	if a, ok := v.(Asker); ok {
		return a.Asks()
	}
	return nil
}

func input[In any](m Meta, raw json.RawMessage) (any, error) {
	var in In
	if len(raw) == 0 || string(raw) == "null" {
		return in, nil
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		e := fail.New(fail.Refused, BadInput, m.Name, map[string]string{"in": m.In})
		e.Cause = err
		return nil, e
	}
	return in, nil
}

func (b bound[D, In, Out]) input(raw json.RawMessage) (any, error)   { return input[In](b.meta, raw) }
func (w watched[D, In, Out]) input(raw json.RawMessage) (any, error) { return input[In](w.meta, raw) }

func (r *Registry) Run(ctx context.Context, target machine.Profile, name string, in any, p Progress) (any, error) {
	// invariant: a call run in this process passes the same gate as a frame
	// (section, rule) and keeps its values as they are: nothing of In or Out
	// goes through the wire form, so an error inside an Out stays an error,
	// and a call that fails part-way still hands back what it did.
	h, err := r.local(name)
	if err != nil {
		return nil, err
	}
	return h.run(ctx, target, in, orSilent(p))
}

func (r *Registry) Stream(ctx context.Context, target machine.Profile, name string, in any, p Progress) (<-chan Update[any], error) {
	h, err := r.local(name)
	if err != nil {
		return nil, err
	}
	return h.stream(ctx, target, in, orSilent(p))
}

func (r *Registry) local(name string) (local, error) {
	h, ok := r.byName[name]
	if !ok {
		return nil, fail.New(fail.NotFound, Unknown, name, nil)
	}
	l, ok := h.(local)
	if !ok {
		return nil, fail.New(fail.NotFound, Unknown, name, nil)
	}
	return l, nil
}

func Set(out, v any) {
	// invariant: out is the pointer a Router was handed; a nil v leaves it as it is.
	if v == nil {
		return
	}
	reflect.ValueOf(out).Elem().Set(reflect.ValueOf(v))
}

func typed[In any](m Meta, target machine.Profile, in any) (In, error) {
	var zero In
	if err := gate(m, target); err != nil {
		return zero, err
	}
	switch v := in.(type) {
	case nil:
		return zero, nil
	case *In:
		return *v, nil
	case In:
		return v, nil
	}
	return zero, fail.New(fail.Refused, BadInput, m.Name, map[string]string{"in": m.In})
}

func (b bound[D, In, Out]) run(ctx context.Context, target machine.Profile, in any, p Progress) (any, error) {
	v, err := typed[In](b.meta, target, in)
	if err != nil {
		return nil, err
	}
	return b.runFn(inside(ctx, b.meta), b.deps, v, p)
}

func (b bound[D, In, Out]) stream(ctx context.Context, target machine.Profile, in any, p Progress) (<-chan Update[any], error) {
	out, err := b.run(ctx, target, in, p)
	if err != nil {
		return nil, err
	}
	once := make(chan Update[any], 1)
	once <- Update[any]{Value: out}
	close(once)
	return once, nil
}

func (w watched[D, In, Out]) stream(ctx context.Context, target machine.Profile, in any, p Progress) (<-chan Update[any], error) {
	v, err := typed[In](w.meta, target, in)
	if err != nil {
		return nil, err
	}
	updates, err := w.runFn(ctx, w.deps, v, p)
	if err != nil {
		return nil, err
	}
	out := make(chan Update[any])
	go func() {
		defer close(out)
		for u := range updates {
			select {
			case out <- Update[any]{Value: u.Value, Err: u.Err}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

func (w watched[D, In, Out]) run(ctx context.Context, target machine.Profile, in any, p Progress) (any, error) {
	updates, err := w.stream(ctx, target, in, p)
	if err != nil {
		return nil, err
	}
	var last any
	for u := range updates {
		if u.Err != nil {
			return nil, u.Err
		}
		last = u.Value
	}
	return last, ctx.Err()
}
