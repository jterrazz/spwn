package calls

import (
	"context"
	"strings"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"

	background "github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type Options struct {
	ThisMachine catalog.MachineKind
	// value: the images the catalog knows, each VM's kind read from its own.
	Images []catalog.Image
	// value: the folders of the VMs' known hosts and of their logs.
	KnownHosts, Logs string
	User             string
	Version          string
	// value: asks each other Mac for its own view of itself and hands on
	// each answer as it comes, one at a time; a silent Mac is never handed on.
	Ask       func(ctx context.Context, macs []string, answered func(mac string, view model.MacView))
	GuestPort int
	Jobs      background.Jobs
	Unlocker  contract.Unlocker
	SSHConfig contract.SSHConfig
	// value: whether another Mac's station answers now, on a connection of
	// its own; nil asks its sshd alone.
	Answers func(ctx context.Context, mac string) bool
	// value: what this Mac is, as it reads itself; nil when it cannot say.
	Hardware func(ctx context.Context) coremachine.Hardware
	// value: this Mac has a lid: a laptop.
	Laptop bool
	// value: this Mac runs on battery with its lid closed now: it is going
	// to sleep; nil when it cannot say.
	Asleep func(ctx context.Context) bool
	// value: this Mac's address on your LAN; empty when it cannot say.
	LAN func(ctx context.Context) string
	// value: its servers' uptime records; nil keeps none.
	Uptime contract.UptimeStore
	// value: what sends a Wake-on-LAN packet from this Mac; nil sends none.
	Waker contract.Waker
	// value: this Mac's multiplexer; nil reaches no workspace.
	Terminals Multiplexer
	// value: the name of every agent a running workspace holds, finished
	// ones too; none when it does not answer, nil reads none.
	AgentsOf func(ctx context.Context, vm string) []string
	// value: installs this Mac's engine job and starts it, as engine start
	// does; nil installs none.
	StartEngine func(ctx context.Context) error
	// value: the Machinefiles workspaces are made from, as this Mac reads
	// them; nil reads none.
	Templates Templates
	// value: shares an account this Mac holds with one of its VMs for a
	// term, as accounts share does; nil shares none.
	Share func(ctx context.Context, account, vm string, term time.Duration) error
}

type Templates interface {
	// value: the Machinefile of that name, as a workspace is made from it.
	Template(ctx context.Context, name string) (model.Template, error)
	// value: what the image's version was baked from that its Machinefile
	// no longer says, one line an item.
	Drift(ctx context.Context, name, version string) ([]string, error)
}

type Multiplexer interface {
	// value: makes this Mac's multiplexer reach a workspace by its name,
	// and forget it.
	Add(ctx context.Context, machine string) error
	Remove(ctx context.Context, machine string) error
}

type Nodes interface {
	Peers(ctx context.Context) (map[string]model.Node, error)
}

type EngineVM struct {
	State, Guest, IP, Detail string
	// value: its ssh connection opened; a server's restarts and check.
	SSH      bool
	Restarts model.Restarts
	Check    model.CheckStatus
	BootAt   time.Time
	// value: a server's last check runs, and the step its restart runs.
	Latency []model.LatencyPoint
	Restart *model.RestartWalk
	// value: why a server is down, and its console's last lines then.
	Cause     string
	LastLines []string
	// value: when the create it was made by made it ready.
	ReadyAt time.Time
	Agent   *model.AgentView
}

type Engine interface {
	Ping(ctx context.Context) (version string, err error)
	VMs(ctx context.Context) (map[string]EngineVM, error)
	Do(ctx context.Context, op, vm string) error
	// value: the ports a VM's engine reserved for the env its run set, each
	// with the item that names it.
	EnvPorts(ctx context.Context, vm string) ([]model.Reserved, error)
	// invariant: a command inside a VM of this Mac rides the engine's one
	// connection to it; station reaches a VM no other way.
	GuestExec(ctx context.Context, vm, line, stdin string) (GuestRan, error)
}

type Releaser interface {
	ReleaseVM(ctx context.Context, vm string) error
}

type Images interface {
	Latest(ctx context.Context, image string) (disk, version string, err error)
	KeyPath() string
}

type Machinist struct {
	h         ThisMac
	engine    contract.Hypervisor
	p         Prefs
	recs      contract.MachineStore
	nodes     Nodes
	img       Images
	engineAPI Engine
	rel       Releaser
	o         Options
}

type Prefs struct {
	Name   setupcontract.NamePrefs
	Macs   contract.MacsPrefs
	VMUser contract.VMUserPrefs
}

type prefsView struct {
	ThisMachine string
	Macs        map[string]model.MacRecord
	VMUser      string
}

func (s *Machinist) prefs() (prefsView, error) {
	var v prefsView
	var err error
	if v.ThisMachine, err = s.p.Name.Get(); err != nil {
		return prefsView{}, err
	}
	if v.Macs, err = s.p.Macs.Get(); err != nil {
		return prefsView{}, err
	}
	if v.VMUser, err = s.p.VMUser.Get(); err != nil {
		return prefsView{}, err
	}
	return v, nil
}

func New(h ThisMac, e contract.Hypervisor, p Prefs, vms contract.MachineStore, nodes Nodes, img Images, d Engine, rel Releaser, o Options) *Machinist {
	return &Machinist{h: h, engine: e, p: p, recs: vms, nodes: nodes, img: img, engineAPI: d, rel: rel, o: o}
}

type Machine struct {
	Name        string
	Kind        catalog.MachineKind
	ThisMachine bool
	Mac         string
	User, Key   string
	// value: a Mac's last address on your LAN, as it answered with it.
	LAN      string
	Lifetime model.Lifetime
}

func (s *Machinist) vmUser(cfg prefsView) string {
	if cfg.VMUser != "" {
		return cfg.VMUser
	}
	return s.o.User
}

func (s *Machinist) CheckMac() error {
	if s.o.ThisMachine != catalog.Mac {
		return &model.OnlyOnMacError{ThisMachine: s.o.ThisMachine}
	}
	return nil
}

var nameChars = func(r rune) bool { return r == '-' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') }

func validName(name string) bool {
	return len(name) >= 1 && len(name) <= 40 && strings.IndexFunc(name, func(r rune) bool { return !nameChars(r) }) < 0
}
