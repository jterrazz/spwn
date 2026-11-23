package calls

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

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
	// value: what keeps this Mac awake, in words; empty when nothing does
	// or the engine cannot say; nil says none.
	Awake func(ctx context.Context) string
	// value: this Mac's address on your LAN; empty when it cannot say.
	LAN func(ctx context.Context) string
	// value: the memory each VM's process holds on this Mac now, in bytes,
	// by VM; nil measures none.
	Uses func(ctx context.Context) map[string]int64
	// value: its servers' uptime records; nil keeps none.
	Uptime contract.UptimeStore
	// value: what sends a Wake-on-LAN packet from this Mac; nil sends none.
	Waker contract.Waker
	// value: installs this Mac's engine job and starts it, as engine start
	// does; nil installs none.
	StartEngine func(ctx context.Context) error
	// value: this Mac's engine job is installed, as its job manager
	// answers; nil says it is not.
	EngineInstalled func() bool
	// value: the Machinefiles workspaces are made from, as this Mac reads
	// them; nil reads none.
	Templates Templates
	// value: grants one of its VMs this Mac's password for a program, as
	// grants.add does, refused by a coded error; nil grants none.
	Grant func(ctx context.Context, program, vm string) error
	// value: the programs a VM of this Mac is signed in to, as accounts.list
	// reads it there.
	SignedIn func(ctx context.Context, vm string) (map[string]bool, error)
	// value: the processes of this Mac, which say whether a create still
	// runs; nil says none does.
	Processes contract.Processes
	// value: the station binary a macOS guest's first login puts in: this
	// Mac's own; nil puts none.
	GuestBinary func() ([]byte, error)
}

type Templates interface {
	// value: the Machinefile of that name, as a workspace is made from it.
	Template(ctx context.Context, name string) (model.Template, error)
	// value: what the image's version was baked from that its Machinefile
	// no longer says, one line a package.
	Drift(ctx context.Context, name, version string) ([]string, error)
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
	// value: how a running workspace stands with its file, as the engine
	// keeps it; nil before its first reading since its boot.
	Drift *model.KeptDrift
	// value: what refused its guest's last connection; empty when nothing did.
	Blocked model.GuestBlock
	// value: why its engine stopped its run, its lockdown having failed;
	// nil otherwise.
	Lockdown *model.LockdownFailure
}

type Engine interface {
	// value: the engine as its hello says it: running, its version and its
	// Mac's Local Network approval.
	Hello(ctx context.Context) (model.EngineView, error)
	VMs(ctx context.Context) (map[string]EngineVM, error)
	Do(ctx context.Context, op, vm string) error
	// value: the ports a VM's engine reserved for the env its run set, each
	// with the package that names it.
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
	// value: the image key's public line, the key made when this Mac has none.
	ImageKey(ctx context.Context) (pub string, err error)
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

func (s *Machinist) engineInstalled() bool {
	return s.o.EngineInstalled != nil && s.o.EngineInstalled()
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

func (s *Machinist) vmUser(cfg prefsView) (string, error) {
	// invariant: the VM user is vm_user, or else this Mac's account, checked
	// by the one rule wherever it is used, never where it is read: a
	// refused name reaches no ssh command, block nor guest.
	if cfg.VMUser != "" {
		return cfg.VMUser, contract.CheckVMUser(cfg.VMUser, false)
	}
	return s.o.User, contract.CheckVMUser(s.o.User, true)
}

// invariant: a guest's user is the OS user of the Mac it runs on, so a VM on
// another Mac is reached as that Mac's recorded user; a record with no user falls
// back to this Mac's VM user, the only one known.
func (s *Machinist) guestUser(cfg prefsView, mac string) (string, error) {
	if mac != cfg.ThisMachine {
		if u := cfg.Macs[mac].User; u != "" {
			return u, nil
		}
	}
	return s.vmUser(cfg)
}

func (s *Machinist) CheckMac() error {
	if s.o.ThisMachine != catalog.Mac {
		return &model.OnlyOnMacError{ThisMachine: s.o.ThisMachine}
	}
	return nil
}

var nameChars = func(r rune) bool { return r == '-' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') }

// invariant: a purpose is one line with no control character, at most
// model.PurposeMax characters: what a list shows on its row.
func validPurpose(purpose string) bool {
	return utf8.ValidString(purpose) && utf8.RuneCountInString(purpose) <= model.PurposeMax &&
		strings.IndexFunc(purpose, func(r rune) bool { return unicode.IsControl(r) || r == '\u2028' || r == '\u2029' }) < 0
}

func validName(name string) bool {
	return len(name) >= 1 && len(name) <= 40 && strings.IndexFunc(name, func(r rune) bool { return !nameChars(r) }) < 0
}
