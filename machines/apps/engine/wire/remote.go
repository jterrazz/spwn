package wire

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/ports"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	overviewcalls "github.com/jterrazz/studio-station/apps/engine/overview/calls"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
	"github.com/jterrazz/studio-station/apps/engine/remote"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/sshstdio"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/unix"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract"
	"github.com/jterrazz/studio-station/apps/engine/remote/each"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
	engineattach "github.com/jterrazz/studio-station/apps/engine/surface/cli/engine/attach"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

// value: the Router of this run, set once it is built, for what reaches
// another machine from inside a call (the fleet's fan-out, a release there).
type late struct {
	router *remote.Router
	// value: this run's engine, when it is one: a sign-in attaches inside a VM through it.
	engine *process.Engine
}

func (m machine) router(env kit.Env) *remote.Router {
	var macs contract.EndpointDialer
	if spawn, ok := m.sys.Run.(local.Spawner); ok {
		macs = sshstdio.New(spawn, Version())
	}
	if m.world != nil {
		macs = m.world.macs()
	}
	r := remote.NewRouter(m.registry(env), m.profile, remote.Reach{
		Macs:    macs,
		Engine:  unix.New(guestRelay{m.macEngine()}, func() contract.Hello { return m.far.router.Served() }, m.imageOf, func(name string, in json.RawMessage) []string { return m.far.router.Asks(name, in) }),
		Locate:  m.locate,
		Holder:  m.holder,
		Version: Version(),
		Label:   m.label,
		Unborn:  m.unborn,
		Stopped: m.stopped,
		Kept:    m.kept(),
	})
	m.far.router = r
	return r
}

// invariant: a VM whose create still runs answers its Overview from its
// Mac's record, one update for each step of its create, the birth
// machines.list gives; any other call waits for its guest.
func (m machine) unborn(ctx context.Context, vm, name string) (<-chan any, bool) {
	if name != overviewcalls.Read.Name {
		return nil, false
	}
	births, ok := m.machines(false).Births(m.world.watchContext(ctx), vm)
	if !ok {
		return nil, false
	}
	out := make(chan any)
	go func() {
		defer close(out)
		for birth := range births {
			select {
			case out <- overviewcalls.ReadOut{Creating: &birth}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, true
}

// invariant: a VM of this Mac that its engine holds stopped answers its
// Overview from its Mac's record, its head saying it is stopped; any
// other call goes to its guest, which refuses it.
func (m machine) stopped(ctx context.Context, vm, name string, in json.RawMessage) (any, bool) {
	var read overviewcalls.ReadIn
	if name != overviewcalls.Read.Name || len(in) > 0 && json.Unmarshal(in, &read) != nil {
		return nil, false
	}
	live, err := m.macEngine().VMs(ctx)
	if err != nil || live[vm].State != engine.StateStopped {
		return nil, false
	}
	found, err := m.machines(false).Lookup(ctx, vm)
	if err != nil {
		return nil, false
	}
	return overviewcalls.ReadOut{Summary: overview.Stopped(vm, found.Lifetime.Kind())}, true
}

func (m machine) label(mac string) string {
	if short, ok := strings.CutPrefix(mac, m.sys.Env.User()+"-"); ok && short != "" {
		return short
	}
	return mac
}

// invariant: a bare name is a Mac when this Mac holds its host record, a VM
// of this Mac when it holds its VM record; any other name is unknown here.
func (m machine) locate(name string) (coremachine.Ref, bool) {
	found, err := m.machines(false).Lookup(context.Background(), name)
	switch {
	case err != nil:
		return coremachine.Ref{Name: name}, false
	case found.Kind != catalog.Mac:
		return coremachine.Ref{Name: found.Name, On: m.profile.Ref.Name}, true
	}
	return coremachine.Ref{Name: found.Name}, true
}

// invariant: a VM of another Mac is found in that Mac's own list, asked of
// every other Mac at once; a name none of them holds is unknown, and while a
// Mac is silent the VM may be its own, so that silence is the answer: the
// silent Mac that held it at its last answer first.
func (m machine) holder(ctx context.Context, vm string) (string, error) {
	r := m.far.router
	others := m.machines(false).OtherMacs()
	got := each.All(ctx, others, func(ctx context.Context, mac string) (bool, error) {
		out, err := call.Read(ctx, r, coremachine.Ref{Name: mac}, machinescalls.List, machinescalls.ListIn{ThisMac: true}, nil)
		if err != nil {
			return false, err
		}
		for _, mac := range out.All.Macs {
			if slices.ContainsFunc(mac.VMs, func(v model.VM) bool { return v.Name == vm }) {
				return true, nil
			}
		}
		return false, nil
	})
	for _, mac := range others {
		if got[mac].Err == nil && got[mac].Value {
			return mac, nil
		}
	}
	for _, mac := range others {
		if got[mac].Err != nil && m.machines(false).Remembers(mac, vm) {
			return mac, nil
		}
	}
	for _, mac := range others {
		if err := got[mac].Err; err != nil {
			return "", err
		}
	}
	return "", &model.UnknownMachineError{Name: vm}
}

// value: the Mac that runs a VM: this Mac for one it holds the record of,
// else the other Mac whose list names it; ok false for a Mac's name or a
// name no Mac holds.
func (m machine) macOf(ctx context.Context, name string) (string, bool) {
	if ref, known := m.locate(name); known {
		return ref.On, ref.On != ""
	}
	if m.far.router == nil {
		return "", false
	}
	mac, err := m.holder(ctx, name)
	return mac, err == nil && mac != ""
}

// value: the image a VM of this Mac was made from, as its record keeps it;
// empty for a name this Mac keeps no VM record of.
func (m machine) imageOf(vm string) string {
	records, err := m.machines(false).Records()
	if err != nil {
		return ""
	}
	for _, r := range records {
		if r.Name == vm {
			return r.Image
		}
	}
	return ""
}

type guestRelay struct{ d macEngine }

func (g guestRelay) GuestExec(ctx context.Context, vm, name, line, stdin string) (stdout string, code int, err error) {
	out, err := g.d.GuestCall(ctx, vm, name, line, stdin)
	return out.Stdout, out.Code, err
}

func (m machine) askAllMachines(ctx context.Context, macs []string, answered func(string, model.MacView)) {
	r := m.far.router
	for got := range each.Do(ctx, macs, func(ctx context.Context, mac string) (model.MacView, error) {
		hello, err := r.Hello(ctx, mac)
		if err != nil {
			return model.MacView{}, err
		}
		out, err := call.Read(ctx, r, coremachine.Ref{Name: mac}, machinescalls.List, machinescalls.ListIn{ThisMac: true}, nil)
		if otherRelease(err) {
			return model.MacView{Name: mac, StationVersion: hello.Version, Reachable: true, OtherRelease: true}, nil
		}
		if err != nil || len(out.All.Macs) == 0 {
			return model.MacView{}, orNone(err)
		}
		view := out.All.Macs[0]
		view.StationVersion = hello.Version
		return view, nil
	}) {
		if got.Err == nil {
			answered(got.Mac, got.Value)
		}
	}
}

// value: a Mac on another release answers its version alone: its row says
// which side to update, and its VMs are the ones remembered.
func otherRelease(err error) bool {
	var e *fail.Error
	return errors.As(err, &e) && (e.Code == call.CallerTooOld || e.Code == call.FarTooOld && e.Detail["call"] == "")
}

func (m machine) askPorts(ctx context.Context, macs []string) map[string]ports.MacPorts {
	r := m.far.router
	answers := map[string]ports.MacPorts{}
	for mac, got := range each.All(ctx, macs, func(ctx context.Context, mac string) (ports.MacPorts, error) {
		out, err := call.Invoke(ctx, r, coremachine.Ref{Name: mac}, portscalls.List, portscalls.ListIn{}, nil)
		if err != nil || len(out.AllMachines.Macs) == 0 {
			return ports.MacPorts{}, orNone(err)
		}
		return out.AllMachines.Macs[0], nil
	}) {
		if got.Err == nil {
			answers[mac] = got.Value
		}
	}
	return answers
}

func (m machine) releaseOn(ctx context.Context, mac string, t ports.Target) (ports.Reservation, error) {
	in := portscalls.StopSharingIn{VM: t.VM, Port: strconv.Itoa(t.Port)}
	out, err := call.Invoke(ctx, m.far.router, coremachine.Ref{Name: mac}, portscalls.StopSharing, in, nil)
	return out.Reservation, err
}

// value: a declaration kept by another Mac, asked of that Mac.
type farPorts struct{ m machine }

func (f farPorts) Declare(ctx context.Context, mac string, in portscalls.DeclareIn) (portscalls.DeclareOut, error) {
	return call.Invoke(ctx, f.m.far.router, coremachine.Ref{Name: mac}, portscalls.Declare, in, nil)
}

func (f farPorts) Undeclare(ctx context.Context, mac string, in portscalls.UndeclareIn) (portscalls.UndeclareOut, error) {
	return call.Invoke(ctx, f.m.far.router, coremachine.Ref{Name: mac}, portscalls.Undeclare, in, nil)
}

func orNone(err error) error {
	if err != nil {
		return err
	}
	return fail.New(fail.Unreachable, call.Unreachable, "", nil)
}

// value: the Router a call reaches another machine through: this run's,
// once it is built.
type farRouter struct{ m machine }

func (f farRouter) Invoke(ctx context.Context, target coremachine.Ref, name string, in, out any, p call.Progress) error {
	if f.m.far == nil || f.m.far.router == nil {
		return orNone(nil)
	}
	return f.m.far.router.Invoke(ctx, target, name, in, out, p)
}

func (f farRouter) Watch(ctx context.Context, target coremachine.Ref, name string, in any, p call.Progress) (<-chan call.Update[any], error) {
	if f.m.far == nil || f.m.far.router == nil {
		return nil, orNone(nil)
	}
	return f.m.far.router.Watch(ctx, target, name, in, p)
}

// invariant: a server runs each frame through this run's Router: a call
// for this machine here, one for its VM over its engine's connection. A
// door onto the engine socket also sends a frame for another Mac, or for
// a VM of another Mac, through that Router, as the socket's Far does.
type served struct {
	r    *remote.Router
	door door
}

// value: a door onto the engine socket: the frames that name another Mac
// or its VM, which ride the Router, and whether This Mac's engine answers
// them; a zero door runs every frame by the Router's own serving rule.
type door struct {
	far func(coremachine.Ref) bool
	up  func() bool
}

// invariant: a far frame is answered by the engine's socket, so with This
// Mac's engine not running it is refused as an absent socket refuses it.
func (d door) rides(target coremachine.Ref) (bool, error) {
	if d.far == nil || !d.far(target) {
		return false, nil
	}
	if !d.up() {
		return true, &enginecalls.DownError{}
	}
	return true, nil
}

func (s served) Hello() contract.Hello { return s.r.Served() }

func (s served) Invoke(ctx context.Context, target coremachine.Ref, name string, in json.RawMessage, p call.Progress) (any, error) {
	far, err := s.door.rides(target)
	if err != nil {
		return nil, err
	}
	if far {
		var out json.RawMessage
		if err := s.r.Invoke(ctx, target, name, in, &out, p); err != nil {
			return nil, err
		}
		return out, nil
	}
	return s.r.Serve(ctx, target, name, in, p)
}

func (s served) Watch(ctx context.Context, target coremachine.Ref, name string, in json.RawMessage, p call.Progress) (<-chan call.Update[any], error) {
	far, err := s.door.rides(target)
	if err != nil {
		return nil, err
	}
	if far {
		return s.r.Watch(ctx, target, name, in, p)
	}
	return s.r.ServeWatch(ctx, target, name, in, p)
}

func serveFrames(env kit.Env, d door) func(ctx context.Context, in io.Reader, out io.Writer) error {
	return func(ctx context.Context, in io.Reader, out io.Writer) error {
		r, _ := env.Calls.(*remote.Router)
		return serve.Stream(ctx, in, out, served{r: r, door: d}, func(err error) *fail.Error { return kit.Translate(err, "") })
	}
}

// invariant: a world run on a Mac is the window's door onto its engine
// socket, so a frame there for another Mac, or for a VM of another Mac, is
// answered by that machine as the socket answers it, while the world says
// This Mac's engine runs; any other run's stdio door answers for its own
// machine and its VMs alone.
func (m machine) socketDoor() door {
	if m.world == nil || m.kind != catalog.Mac {
		return door{}
	}
	return door{far: func(t coremachine.Ref) bool {
		if t.Name == "" || t.Name == m.profile.Ref.Name {
			return false
		}
		ref, _ := m.locate(t.Name)
		return ref.On != m.profile.Ref.Name
	}, up: m.world.runs}
}

func engineCommand(env kit.Env, m machine) *cobra.Command {
	return engineNoun(env, m, engineRunCommand(env, m), engineattach.Command(engineattach.Deps{Serve: serveFrames(env, door{})}, env))
}

// value: a run that serves frames on its stdout: nothing else of it may write there.
func serving(args []string) bool {
	return len(args) >= 2 && (args[0] == "engine" && args[1] == "attach" || args[0] == "guest" && args[1] == "serve")
}
