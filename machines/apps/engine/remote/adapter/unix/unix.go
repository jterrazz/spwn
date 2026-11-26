package unix

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract"
)

const (
	// value: the guest's own API, over the one connection its Mac's engine holds.
	Guest = "/usr/local/bin/station guest serve --stdio"
)

type Engine interface {
	// value: runs line inside vm over the engine's own connection to it,
	// stdin its input, carrying call, the request it holds (empty for a
	// Hello alone); the engine is reached over its unix socket.
	GuestExec(ctx context.Context, vm, call, line, stdin string) (stdout string, code int, err error)
}

type Dialer struct {
	engine Engine
	// value: this Mac's own Hello: its engine is this binary.
	thisMac func() contract.Hello
	// value: the image a VM of this Mac was made from, which holds its
	// station; empty for a VM this Mac keeps no record of.
	image func(vm string) string
	// value: the abilities a call's input asks of the guest that runs it
	// (call.Registry.Asks); nil asks none.
	asks func(name string, in json.RawMessage) []string
}

func New(engine Engine, thisMac func() contract.Hello, image func(vm string) string, asks func(name string, in json.RawMessage) []string) Dialer {
	return Dialer{engine: engine, thisMac: thisMac, image: image, asks: asks}
}

// value: the hello this Mac opens an exchange with: its version, so a
// guest answers in the words it reads; none when the Dialer has no Hello.
func (d Dialer) hello() *api.Hello {
	if d.thisMac == nil {
		return &api.Hello{}
	}
	return &api.Hello{Version: d.thisMac().Version}
}

func (d Dialer) Dial(context.Context, string) (contract.Endpoint, error) {
	return endpoint(d), nil
}

type endpoint Dialer

func (e endpoint) Hello(context.Context) (contract.Hello, error) { return e.thisMac(), nil }

func (e endpoint) Close() error { return nil }

func (e endpoint) Invoke(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (json.RawMessage, error) {
	frames, err := e.exchange(ctx, target, api.Frame{Call: name, Target: api.TargetOf(target), In: in, By: api.ByOf(ctx), Far: call.Far(ctx)})
	if err != nil {
		return nil, err
	}
	for _, f := range frames {
		switch {
		case f.Progress != nil:
			api.Report(p, f.Progress)
		case f.Error != nil:
			return nil, f.Error.Err()
		default:
			return f.Out, nil
		}
	}
	return nil, lost(target, errors.New("the guest answered nothing"))
}

func (e endpoint) Watch(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (<-chan call.Update[json.RawMessage], error) {
	frames, err := e.exchange(ctx, target, api.Frame{Watch: name, Target: api.TargetOf(target), In: in, By: api.ByOf(ctx), Far: call.Far(ctx)})
	if err != nil {
		return nil, err
	}
	out := make(chan call.Update[json.RawMessage], len(frames))
	for _, f := range frames {
		switch {
		case f.Progress != nil:
			api.Report(p, f.Progress)
		case f.Error != nil:
			out <- call.Update[json.RawMessage]{Err: f.Error.Err()}
		case !f.Done:
			out <- call.Update[json.RawMessage]{Value: f.Out}
		}
	}
	close(out)
	return out, nil
}

func (d Dialer) KindOf(ctx context.Context, vm string) (machine.Kind, error) {
	// value: the kind a VM says it is, in the Hello its own station answers.
	far, err := d.far(ctx, machine.Ref{Name: vm})
	if err != nil {
		return "", err
	}
	return far.Kind, nil
}

// value: the Hello a VM's own station answers, alone in its exchange.
func (d Dialer) far(ctx context.Context, target machine.Ref) (*api.Hello, error) {
	hello, _ := json.Marshal(api.Frame{V: api.Version, ID: "1", Hello: d.hello()})
	stdout, code, err := d.engine.GuestExec(ctx, target.Name, "", Guest, string(hello)+"\n")
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(strings.NewReader(stdout))
	s.Buffer(make([]byte, 0, 64<<10), 64<<20)
	for s.Scan() {
		var f api.Frame
		if json.Unmarshal(s.Bytes(), &f) == nil && f.ID == "1" && f.Hello != nil {
			return f.Hello, nil
		}
	}
	return nil, lost(target, errors.New("the guest's station did not answer, exit "+strconv.Itoa(code)))
}

// invariant: two exchanges per call: the guest's server reads its stdin to
// the end, answers and exits, so its Hello is asked alone first and a guest
// on another release than this Mac's, or one that lacks an ability the
// request's input asks, is refused before the request reaches it; the second
// sends the hello and the request, and its Hello is judged again, so a guest
// whose binary changed between the two, to another release or one that
// serves no such call or ability, is refused.
func (e endpoint) exchange(ctx context.Context, target machine.Ref, request api.Frame) ([]api.Frame, error) {
	first, err := Dialer(e).far(ctx, target)
	if err != nil {
		return nil, err
	}
	name := request.Call + request.Watch
	var asks []string
	if e.asks != nil {
		asks = e.asks(name, request.In)
	}
	if refused := e.skew(target, first.Version); refused != nil {
		return nil, refused
	}
	if lacked := first.Lacks(name, asks); lacked != "" && lacked != name {
		return nil, e.tooOld(target, map[string]string{"version": first.Version, "call": lacked})
	}
	request.V, request.ID = api.Version, "2"
	hello, _ := json.Marshal(api.Frame{V: api.Version, ID: "1", Hello: Dialer(e).hello()})
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	stdout, code, err := e.engine.GuestExec(ctx, target.Name, request.Call+request.Watch, Guest, string(hello)+"\n"+string(body)+"\n")
	if err != nil {
		return nil, err
	}
	var answers []api.Frame
	var far *api.Hello
	s := bufio.NewScanner(strings.NewReader(stdout))
	s.Buffer(make([]byte, 0, 64<<10), 64<<20)
	for s.Scan() {
		var f api.Frame
		if json.Unmarshal(s.Bytes(), &f) != nil {
			continue
		}
		switch {
		case f.ID == "1" && f.Hello != nil:
			far = f.Hello
		case f.ID == "2":
			answers = append(answers, f)
		}
	}
	if far == nil {
		return nil, lost(target, errors.New("the guest's station did not answer, exit "+strconv.Itoa(code)))
	}
	if err := e.skew(target, far.Version); err != nil {
		return nil, err
	}
	if lacked := far.Lacks(name, asks); lacked != "" {
		return nil, e.tooOld(target, map[string]string{"version": far.Version, "call": lacked})
	}
	return answers, nil
}

// value: a guest on another release than this Mac's station is refused: an
// older one by its image's repair, a newer one by updating station here.
func (e endpoint) skew(target machine.Ref, version string) error {
	switch api.SkewOf(Dialer(e).hello().Version, version) {
	case api.FarBehind:
		return e.tooOld(target, map[string]string{"version": version})
	case api.HereBehind:
		return fail.New(fail.Refused, call.CallerTooOld, target.Name, map[string]string{"version": version})
	case api.Speaks:
	}
	return nil
}

// value: a VM's station is the one its image was built with, so the refusal
// names that image: rebuilding it, then making the VM again, is the repair.
func (e endpoint) tooOld(target machine.Ref, detail map[string]string) error {
	if e.image != nil {
		if image := e.image(target.Name); image != "" {
			detail["image"] = image
		}
	}
	return fail.New(fail.FarTooOld, call.FarTooOld, target.Name, detail)
}

func lost(target machine.Ref, cause error) error {
	e := fail.New(fail.Unreachable, call.Unreachable, target.Name, nil)
	e.Cause = cause
	return e
}
