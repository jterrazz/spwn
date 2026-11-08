package wire

import (
	"context"
	"encoding/json"
	"github.com/jterrazz/studio-station/src/agents"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/ports"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"

	model "github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/platform/local"
	portscalls "github.com/jterrazz/studio-station/src/ports/calls"
	"github.com/jterrazz/studio-station/src/remote"
	"github.com/jterrazz/studio-station/src/remote/adapter/sshstdio"
	"github.com/jterrazz/studio-station/src/remote/adapter/unix"
	"github.com/jterrazz/studio-station/src/remote/contract"
	"github.com/jterrazz/studio-station/src/remote/each"
	"github.com/jterrazz/studio-station/src/remote/serve"
	engineattach "github.com/jterrazz/studio-station/src/surface/cli/engine/attach"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

// value: the Router of this run, set once it is built, for what reaches
// another machine from inside a call (the fleet's fan-out, a release there).
type late struct{ router *remote.Router }

func (m machine) router(env kit.Env) *remote.Router {
	var macs contract.EndpointDialer
	if spawn, ok := m.sys.Run.(local.Spawner); ok {
		macs = sshstdio.New(spawn)
	}
	if m.world != nil {
		macs = m.world.macs()
	}
	r := remote.NewRouter(m.registry(env), m.profile, remote.Reach{
		Macs:    macs,
		Engine:  unix.New(guestRelay{m.macEngine()}, func() contract.Hello { return m.far.router.Served() }),
		Locate:  m.locate,
		Holder:  m.holder,
		Version: Version(),
		Label:   m.label,
	})
	m.far.router = r
	return r
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

type guestRelay struct{ d macEngine }

func (g guestRelay) GuestExec(ctx context.Context, vm, line, stdin string) (stdout string, code int, err error) {
	out, err := g.d.GuestExec(ctx, vm, line, stdin)
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

func (m machine) releaseOn(ctx context.Context, mac string, t ports.Target, yes bool) (ports.Reservation, error) {
	in := portscalls.StopSharingIn{VM: t.VM, Port: strconv.Itoa(t.Port)}
	if yes {
		in.Confirm = confirm.Yes
	}
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

// invariant: a workspace's agents are reached inside it, at its own
// station, as any call for a VM of this Mac.
type farAgents struct{ m machine }

// value: how long a create waits for one workspace to say its agents' names;
// one silent past it holds none.
const heldBudget = 2 * time.Second

func (f farAgents) Held(ctx context.Context, vm string) []string {
	if f.m.far == nil || f.m.far.router == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, heldBudget)
	defer cancel()
	kept, err := call.Invoke(ctx, f.m.far.router, coremachine.Ref{Name: vm}, agentscalls.List, agentscalls.ListIn{AsGiven: true}, nil)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(kept))
	for _, a := range kept {
		if a.Name != "" {
			names = append(names, a.Name)
		}
	}
	return names
}

func (f farAgents) Live(ctx context.Context, vm string) []string {
	kept, err := call.Invoke(ctx, f.m.far.router, coremachine.Ref{Name: vm}, agentscalls.List, agentscalls.ListIn{}, nil)
	if err != nil {
		return nil
	}
	var names []string
	for _, a := range kept {
		if agents.Live(a) {
			names = append(names, a.Name)
		}
	}
	return names
}

func (f farAgents) Stop(ctx context.Context, vm, reason string) (stopped, warnings []string) {
	for _, name := range f.Live(ctx, vm) {
		if _, err := call.Invoke(ctx, f.m.far.router, coremachine.Ref{Name: vm}, agentscalls.Stop, agentscalls.StopIn{Agent: name, Reason: reason, Confirm: confirm.Yes}, nil); err != nil {
			said, _ := word(fail.New(fail.Failed, agentscalls.NotStopped, name, map[string]string{"cause": kit.Translate(err, "").Message}))
			warnings = append(warnings, said)
			continue
		}
		stopped = append(stopped, name)
	}
	return stopped, warnings
}

func orNone(err error) error {
	if err != nil {
		return err
	}
	return fail.New(fail.Unreachable, call.Unreachable, "", nil)
}

// invariant: a server runs each frame through this run's Router: a call
// for this machine here, one for its VM over its engine's connection.
type served struct{ r *remote.Router }

func (s served) Hello() contract.Hello { return s.r.Served() }

func (s served) Invoke(ctx context.Context, target coremachine.Ref, name string, in json.RawMessage, p call.Progress) (any, error) {
	return s.r.Serve(ctx, target, name, in, p)
}

func (s served) Watch(ctx context.Context, target coremachine.Ref, name string, in json.RawMessage, p call.Progress) (<-chan call.Update[any], error) {
	return s.r.ServeWatch(ctx, target, name, in, p)
}

func serveFrames(env kit.Env) func(ctx context.Context, in io.Reader, out io.Writer) error {
	return func(ctx context.Context, in io.Reader, out io.Writer) error {
		r, _ := env.Calls.(*remote.Router)
		return serve.Stream(ctx, in, out, served{r}, func(err error) *fail.Error { return kit.Translate(err, "") })
	}
}

func engineCommand(env kit.Env, m machine) *cobra.Command {
	return engineNoun(env, m, engineRunCommand(env, m), engineattach.Command(engineattach.Deps{Serve: serveFrames(env)}, env))
}

// value: a run that serves frames on its stdout: nothing else of it may write there.
func serving(args []string) bool {
	return len(args) >= 2 && (args[0] == "engine" && args[1] == "attach" || args[0] == "guest" && args[1] == "serve")
}
