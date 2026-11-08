package wire

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/images"
	"github.com/jterrazz/studio-station/src/platform/local"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	imagescalls "github.com/jterrazz/studio-station/src/images/calls"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/machines/adapter/macos"
	"github.com/jterrazz/studio-station/src/machines/adapter/openssh"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/machines/contract"
	settingsmacos "github.com/jterrazz/studio-station/src/settings/adapter/macos"
	settingscontract "github.com/jterrazz/studio-station/src/settings/contract"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
	machinescmd "github.com/jterrazz/studio-station/src/surface/cli/machines"
	machinesadd "github.com/jterrazz/studio-station/src/surface/cli/machines/add"
	machinescleanup "github.com/jterrazz/studio-station/src/surface/cli/machines/cleanup"
	machinescreate "github.com/jterrazz/studio-station/src/surface/cli/machines/create"
	machinesdelete "github.com/jterrazz/studio-station/src/surface/cli/machines/delete"
	machinesedit "github.com/jterrazz/studio-station/src/surface/cli/machines/edit"
	machinesforget "github.com/jterrazz/studio-station/src/surface/cli/machines/forget"
	machineslog "github.com/jterrazz/studio-station/src/surface/cli/machines/log"
	machinespipe "github.com/jterrazz/studio-station/src/surface/cli/machines/pipe"
	machinesprobe "github.com/jterrazz/studio-station/src/surface/cli/machines/probe"
	machinesrestart "github.com/jterrazz/studio-station/src/surface/cli/machines/restart"
	machinesstart "github.com/jterrazz/studio-station/src/surface/cli/machines/start"
	machinesstop "github.com/jterrazz/studio-station/src/surface/cli/machines/stop"
	machinessupervise "github.com/jterrazz/studio-station/src/surface/cli/machines/supervise"
	machinesunlock "github.com/jterrazz/studio-station/src/surface/cli/machines/unlock"
	overviewcmd "github.com/jterrazz/studio-station/src/surface/cli/overview"
)

const (
	lanBudget = 2 * time.Second
)

func (m machine) machines(_ bool) *machinescalls.Machinist {
	knownHosts := filepath.Join(m.paths.State, "known_hosts")
	return machinescalls.New(macSystem{m.sys}, m.hypervisor(), machinescalls.Prefs{Name: m.stores.name(), Macs: m.stores.macs(), VMUser: m.stores.vmUser()}, m.stores.machines, macNodes{tailnet: m.tailnet()}, allMachinesImages{images: m.images()}, machinesEngine{m.macEngine()}, reservationReleaser{m: m}, machinescalls.Options{
		ThisMachine: m.kind,
		KnownHosts:  knownHosts, Logs: filepath.Join(m.paths.State, "logs"),
		User:      m.sys.Env.User(),
		Version:   m.stationVersion(),
		Ask:       m.askAllMachines,
		GuestPort: m.guestPort(),
		Jobs:      m.jobs(0),
		Unlocker:  m.unlocker(knownHosts),
		Answers:   m.answers,
		SSHConfig: m.sshConfig(knownHosts),
		Hardware:  m.hardware,
		Laptop:    m.profile.Caps.Lid,
		Asleep:    m.asleep,
		LAN:       m.lanAddress,
		Uptime:    m.stores.uptime,
	})
}

// value: this Mac's address on your LAN, as it reads it (its world's in
// the spec build).
func (m machine) lanAddress(ctx context.Context) string {
	var facts settingscontract.Inspector = settingsmacos.New(m.sys.Run, m.sys.FS, m.sys.Clock, settingsmacos.Options{Root: m.systemRoot()})
	if m.world != nil {
		facts = m.world.inspector()
	}
	ctx, cancel := context.WithTimeout(ctx, lanBudget)
	defer cancel()
	got, err := facts.Read(ctx, catalog.FactLocalIP)
	if err != nil || got.State != catalog.StateOK {
		return ""
	}
	return got.Value
}

// value: this Mac runs on battery with its lid closed, as its power reads
// (its world's in the spec build), or its engine heard macOS say it is
// going to sleep.
func (m machine) asleep(ctx context.Context) bool {
	if m.powerPort == nil {
		return false
	}
	if r, err := m.powerPort.Read(); err == nil && m.profile.Caps.Lid && r.LidClosedOnBattery() {
		return true
	}
	ctx, cancel := context.WithTimeout(ctx, lanBudget)
	defer cancel()
	var v awake.View
	return m.engineAPI().do(ctx, engine.CallAwakeView, nil, &v) == nil && v.Sleeping
}

// value: what this Mac is, as it reads itself (its world's in the spec
// build).
func (m machine) hardware(ctx context.Context) coremachine.Hardware {
	self := selfOf(m.sys)
	if m.world != nil {
		self = m.world.self()
	}
	_, _, _, hw, _ := self.Read(ctx)
	return hw
}

// invariant: a Mac answers when its station does, on a connection dialed
// for the question: one kept from before it went down proves nothing.
func (m machine) answers(ctx context.Context, mac string) bool {
	r := m.far.router
	if r == nil {
		return false
	}
	r.Drop(mac)
	_, err := r.Hello(ctx, mac)
	return err == nil
}

func (m machine) unlocker(knownHosts string) contract.Unlocker {
	if m.world != nil {
		return m.world.unlocker()
	}
	return macos.New(m.sys.Run, m.sys.FS, knownHosts)
}

func (m machine) sshConfig(knownHosts string) contract.SSHConfig {
	if m.world != nil {
		return m.world.sshConfig()
	}
	return openssh.New(m.sys.FS, openssh.Options{
		Home: m.sys.Env.Home(), KnownHosts: knownHosts, Executable: m.sys.Env.Executable(),
	})
}

func (m machine) port(fallback int, names ...string) int {
	for _, name := range names {
		if n, err := strconv.Atoi(m.vars[name]); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func machinesCommand(env kit.Env, m machine) *cobra.Command {
	vms := m.machines(env.Interactive)
	overview := func(ctx context.Context, io kit.IO) error {
		return overviewcmd.Run(ctx, m.overviewDeps(false), env, false, io)
	}
	return machinescmd.Command(machinescmd.Deps{ThisMachine: m.where, Names: vms.Names, Overview: overview}, env,
		machinescreate.Command(machinescreate.Deps{
			Resources: model.ParseResources, Check: createCheck(vms),
			NotBuilt: func(image string) error { return &model.NoImageError{Image: image} },
			Admit: func() error {
				return call.Gate(call.Meta{Name: machinescalls.CreateWorkspace.Name, Applies: machinescalls.CreateWorkspace.Applies}, m.profile)
			},
			AllMachines: func(ctx context.Context) (model.AllMachines, error) { return vms.AllMachines(ctx, false) },
			ThisMachine: m.where, Ask: createAsker(env),
		}, env),
		machinesstart.Command(machinesstart.Deps{Names: vms.Names, ThisMachine: m.where}, env),
		machinesstop.Command(machinesstop.Deps{Names: vms.Names, ThisMachine: m.where}, env),
		machinesrestart.Command(machinesrestart.Deps{Names: vms.Names, IsMac: func(name string) bool {
			r, err := vms.Remote(name)
			return err == nil && r.Kind == catalog.Mac && !env.Far()
		}, Pausing: m.pausing(env), ThisMachine: m.where}, env),
		machinesdelete.Command(machinesdelete.Deps{Names: vms.Names, ThisMachine: m.where}, env),
		machinesedit.Command(machinesedit.Deps{Names: vms.Names, Resources: model.ParseResources, ThisMachine: m.where}, env),
		machinesprobe.Alias(machinesprobe.Deps{Names: vms.Names, Overview: overview}, env),
		machinescmd.SSHCommand(machinescmd.SSHDeps{Names: vms.Names, ThisMachine: m.where}, env),
		machineslog.Command(machineslog.Deps{Names: vms.Names}, env),
		machinescleanup.Command(machinescleanup.Deps{Names: vms.Names, ThisMachine: m.where}, env),
		machinesadd.Command(machinesadd.Deps{ThisMachine: m.where}, env),
		machinesforget.Command(machinesforget.Deps{Names: vms.Names, ThisMachine: m.where}, env),
		machinesunlock.Command(machinesunlock.Deps{Names: vms.Names, ThisMachine: m.where}, env),
		machinessupervise.Command(env),
		machinespipe.Command(machinespipe.Deps{Machines: vmPipe{vms, m.sys.Sockets}}, env),
	)
}

func (m machine) pausing(env kit.Env) func(mac string) []string {
	return func(mac string) []string {
		vms, _ := machineCalls{Machinist: m.machines(env.Interactive), m: m}.VMsOf(context.Background(), mac)
		var names []string
		for _, vm := range model.Ordered(vms) {
			if vm.State == model.Running {
				names = append(names, vm.Name)
			}
		}
		return names
	}
}

func createCheck(s *machinescalls.Machinist) func(ctx context.Context, r model.CreateRequest) (bool, error) {
	return func(ctx context.Context, r model.CreateRequest) (bool, error) {
		_, err := s.CheckCreate(ctx, r)
		var unbuilt *model.NoImageError
		if errors.As(err, &unbuilt) {
			return true, err
		}
		return false, err
	}
}

type vmPipe struct {
	*machinescalls.Machinist
	sockets local.Sockets
}

func (p vmPipe) Pipe(ctx context.Context, vm string, in io.Reader, out io.Writer) error {
	return p.Machinist.Pipe(ctx, vm, func(ctx context.Context, ip string, port int) error {
		return p.sockets.Pipe(ctx, ip, port, in, out)
	})
}

func (p vmPipe) Check(ctx context.Context, vm string) error {
	_, err := p.CheckPipe(ctx, vm)
	return err
}

// invariant: the machines calls' Deps over today's service: each check the
// leaves ran before, worded by the service's own errors.
type machineCalls struct {
	*machinescalls.Machinist
	m machine
}

// invariant: a Mac is reached as every later call reaches it, through this
// run's Router over its ssh block; its version is judged against this one's
// by the fleet's own rule.
func (c machineCalls) Reach(ctx context.Context, name string) (model.Added, error) {
	hello, err := c.m.far.router.Hello(ctx, name)
	if err != nil {
		return model.Added{}, err
	}
	update, _ := model.Behind(model.MacView{Name: name, StationVersion: hello.Version}, c.m.stationVersion())
	return model.Added{Name: name, Label: c.m.label(name), Version: hello.Version, Update: update}, nil
}

func (c machineCalls) CheckVM(name, verb string) error {
	_, err := c.VM(name, verb)
	return err
}

func (c machineCalls) CheckRestart(name string) (bool, error) {
	r, err := c.Remote(name)
	if err != nil {
		return false, err
	}
	if r.Kind == catalog.Mac {
		return false, nil
	}
	return true, nil
}

func (c machineCalls) Running(ctx context.Context, name string) (bool, error) {
	allMachines, err := c.AllMachines(ctx, true)
	if err != nil {
		return false, err
	}
	for _, h := range allMachines.Macs {
		for _, vm := range h.VMs {
			if vm.Name == name {
				return vm.State == model.Running, nil
			}
		}
	}
	return false, &model.UnknownMachineError{Name: name}
}

func (c machineCalls) Restart(ctx context.Context, name string) (model.RestartReport, error) {
	return c.Machinist.Restart(ctx, name, nil)
}

func (c machineCalls) CheckRemoteMac(name, verb string) (string, error) {
	m, err := c.RemoteMac(name, verb)
	return m.Name, err
}

func (c machineCalls) Unlock(ctx context.Context, name string, interactive bool) error {
	if !interactive {
		return &model.NeedsTerminalError{Verb: "machines unlock"}
	}
	return c.Machinist.Unlock(ctx, name)
}

func (c machineCalls) VMsOf(ctx context.Context, name string) ([]model.VM, error) {
	out, err := call.Read(ctx, c.m.far.router, coremachine.Ref{Name: name}, machinescalls.List, machinescalls.ListIn{ThisMac: true}, nil)
	if err != nil || len(out.All.Macs) == 0 {
		return nil, err
	}
	return out.All.Macs[0].VMs, nil
}

func (c machineCalls) StartOn(ctx context.Context, mac, vm string) error {
	_, err := call.Invoke(ctx, c.m.far.router, coremachine.Ref{Name: mac}, machinescalls.Start, machinescalls.VMIn{VM: vm}, nil)
	return err
}

func (c machineCalls) CheckRemote(name string) error {
	_, err := c.Remote(name)
	return err
}

type allMachinesImages struct{ images *imagescalls.Builder }

func (i allMachinesImages) KeyPath() string { return i.images.KeyPath() }

func (i allMachinesImages) Latest(ctx context.Context, image string) (disk, version string, err error) {
	r, err := i.images.Latest(ctx, image)
	var unbuilt *images.NotBuiltError
	if errors.As(err, &unbuilt) {
		return "", "", &model.NoImageError{Image: image}
	}
	if err != nil {
		return "", "", err
	}
	return imagescalls.DiskName(r.Name, r.Version), r.Version, nil
}

func createAsker(env kit.Env) kit.Asker {
	if !env.Interactive {
		return nil
	}
	return env.Prompt
}

// value: what a VM of this Mac holds beyond its disk, as this Mac's stores
// keep it: its reserved ports and the accounts shared with it, by their label.
type holdings struct{ m machine }

func (h holdings) Held(ctx context.Context, vm string) (ports []int, shared []string, err error) {
	reserved, err := h.m.stores.reservations.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range reserved {
		if r.VM == vm {
			ports = append(ports, r.GuestPort)
		}
	}
	shares, err := h.m.stores.shares.Get(ctx, vm)
	if err != nil {
		return nil, nil, err
	}
	for _, s := range shares {
		if account := accounts.Label(accounts.AccountOf(s.Name)); !slices.Contains(shared, account) {
			shared = append(shared, account)
		}
	}
	return ports, shared, nil
}
