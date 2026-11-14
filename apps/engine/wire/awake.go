package wire

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	awakecalls "github.com/jterrazz/studio-station/apps/engine/awake/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	pageawake "github.com/jterrazz/studio-station/apps/engine/pages/awake"
	"github.com/jterrazz/studio-station/apps/engine/remote/each"
	awakecmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/awake"
	awakeoff "github.com/jterrazz/studio-station/apps/engine/surface/cli/awake/off"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

func (m machine) awakeDeps() awakecalls.Deps {
	return awakecalls.Deps{
		Holder: awakeKeeper{m.awakeWaker()}, Mac: m.profile.Ref.Name, Label: m.label(m.profile.Ref.Name), Caps: m.profile.Caps,
		Servers:     func(context.Context) []string { return m.servers() },
		Running:     m.runningWorkspaces,
		NeverSleeps: func() bool { on, _ := m.stores.neverSleeps().Get(); return on },
	}
}

// value: the servers this Mac keeps, by name.
func (m machine) servers() []string {
	records, err := m.machines(false).Records()
	if err != nil {
		return nil
	}
	var out []string
	for _, r := range records {
		if r.Lifetime == model.AlwaysOn {
			out = append(out, r.Name)
		}
	}
	slices.Sort(out)
	return out
}

// value: the work this Mac runs now, as its engine says: what pauses while
// it sleeps.
func (m machine) runningWorkspaces(ctx context.Context) []string {
	records, err := m.machines(false).Records()
	if err != nil {
		return nil
	}
	live, err := m.macEngine().VMs(ctx)
	if err != nil {
		return nil
	}
	vms := make([]model.VM, 0, len(records))
	for _, r := range records {
		v := live[r.Name]
		vm := model.VM{Name: r.Name, Lifetime: r.Lifetime, Agent: v.Agent}
		if v.State == engine.StateRunning {
			vm.State = model.Running
		}
		vms = append(vms, vm)
	}
	return model.Pausing(vms)
}

func (m machine) awakeCmdDeps(env kit.Env) awakecmd.Deps {
	return awakecmd.Deps{ThisMachine: m.where, Now: m.sys.Clock.Now, Zone: time.Local, Label: m.label,
		Every: func(ctx context.Context) (pageawake.AllMachines, error) { return m.everyMacAwake(ctx, env) }}
}

func awakeCommand(env kit.Env, m machine) *cobra.Command {
	return awakecmd.Command(m.awakeCmdDeps(env), env, awakeoff.Command(awakeoff.Deps{ThisMachine: m.where, Label: m.label}, env))
}

// value: station power, power awake and power sleep, the words v2 gave
// awake, a keep-awake and awake off.
func awakeAliases(env kit.Env, m machine) []*cobra.Command {
	return []*cobra.Command{kit.Alias("power", awakecmd.Command(m.awakeCmdDeps(env), env,
		awakecmd.PowerAwakeAlias(m.awakeCmdDeps(env), env),
		kit.Alias("sleep", awakeoff.Command(awakeoff.Deps{ThisMachine: m.where, Label: m.label}, env), "off"),
	), "awake")}
}

// invariant: This Mac first, then every other Mac that answers, each Mac's
// own Awake through its engine; a Mac that does not answer is said so, and
// This Mac's own refusal is the command's.
func (m machine) everyMacAwake(ctx context.Context, env kit.Env) (pageawake.AllMachines, error) {
	f := pageawake.AllMachines{Now: m.sys.Clock.Now(), Zone: time.Local}
	thisMac, err := call.Read(ctx, env.Calls, env.Self, awakecalls.Read, awakecalls.ReadIn{}, nil)
	if err != nil {
		return f, err
	}
	thisMac.ThisMac = true
	f.Reports = append(f.Reports, thisMac)
	known := m.machines(false)
	others := known.OtherMacs()
	slices.Sort(others)
	got := each.All(ctx, others, func(ctx context.Context, mac string) (awake.Report, error) {
		return call.Read(ctx, env.Calls, coremachine.Ref{Name: mac}, awakecalls.Read, awakecalls.ReadIn{}, nil)
	})
	for _, mac := range others {
		if err := got[mac].Err; err != nil {
			f.Silent = append(f.Silent, silentAwake(known.Seen, mac, err))
			continue
		}
		f.Reports = append(f.Reports, got[mac].Value)
	}
	return f, nil
}

// value: a Mac that did not answer, as it last did: asleep or not, when,
// and the servers it then hosted; why it did not answer is screened, as a
// page draws it as text.
func silentAwake(seen func(string) (model.MacRecord, bool), mac string, err error) pageawake.Silent {
	s := pageawake.Silent{Mac: mac, Why: kit.Screened(err, "awake").Message}
	h, ok := seen(mac)
	if !ok {
		return s
	}
	s.Asleep, s.Answered = h.Asleep, h.Answered
	for _, vm := range h.VMs {
		if vm.Lifetime == model.AlwaysOn {
			s.Servers = append(s.Servers, vm.Name)
		}
	}
	return s
}

// value: the servers that pause once Never sleeps is off on the target Mac,
// in the words of the question that turns it off.
func (m machine) pauses(env kit.Env, names []string) []string {
	if !slices.Contains(names, catalog.ProcNeverSleeps.Name()) {
		return nil
	}
	r, err := call.Read(context.Background(), env.Calls, env.Target(), awakecalls.Read, awakecalls.ReadIn{}, nil)
	if err != nil || len(r.Servers) == 0 {
		return nil
	}
	return []string{strings.Join(r.Servers, ", ") + " pauses while " + m.label(r.Mac) + " sleeps"}
}
