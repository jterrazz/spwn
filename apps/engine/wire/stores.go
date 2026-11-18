package wire

import (
	"context"
	"path/filepath"

	awakecontract "github.com/jterrazz/studio-station/apps/engine/awake/contract"

	accountsjsonfile "github.com/jterrazz/studio-station/apps/engine/accounts/adapter/jsonfile"
	accountscontract "github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	corejsonfile "github.com/jterrazz/studio-station/apps/engine/core/adapter/jsonfile"
	corecontract "github.com/jterrazz/studio-station/apps/engine/core/contract"
	imagesjsonfile "github.com/jterrazz/studio-station/apps/engine/images/adapter/jsonfile"
	imagescontract "github.com/jterrazz/studio-station/apps/engine/images/contract"
	machinefilejsonfile "github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/jsonfile"
	machinefilecontract "github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	machinesjsonfile "github.com/jterrazz/studio-station/apps/engine/machines/adapter/jsonfile"
	machinescontract "github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	portsjsonfile "github.com/jterrazz/studio-station/apps/engine/ports/adapter/jsonfile"
	portscontract "github.com/jterrazz/studio-station/apps/engine/ports/contract"
	settingsjsonfile "github.com/jterrazz/studio-station/apps/engine/settings/adapter/jsonfile"
	settingscontract "github.com/jterrazz/studio-station/apps/engine/settings/contract"
	"github.com/jterrazz/studio-station/apps/engine/setup"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"
)

// invariant: the preference groups in the order preferences.json writes them,
// each group named by its one owner's contract.
func preferenceGroups() []string {
	return []string{
		setupcontract.GroupName, machinescontract.GroupMacs, machinescontract.GroupVMUser,
		setupcontract.GroupUserEmail, setupcontract.GroupUserName,
		portscontract.GroupPorts, settingscontract.GroupOffHere, awakecontract.GroupNeverSleeps, machinefilecontract.GroupSource,
	}
}

type preferences interface {
	corecontract.Preferences
	File() string
}

// value: every store station keeps on this machine, each built once here
// over its file; nothing else constructs one.
type stores struct {
	prefs        preferences
	machines     machinescontract.MachineStore
	images       imagescontract.ImageStore
	marks        imagescontract.BuildMarks
	book         accountscontract.BookStore
	reservations portscontract.ReservationStore
	declarations portscontract.DeclarationStore
	applied      settingscontract.AppliedStore
	uptime       machinescontract.UptimeStore
	ledger       machinefilecontract.Ledger
}

func newStores(files local.FS, p setup.Paths) *stores {
	return &stores{
		prefs:        corejsonfile.NewPreferences(files, p.Prefs, preferenceGroups()),
		machines:     machinesjsonfile.New(lockable{files}, filepath.Join(p.State, "vms")),
		images:       imagesjsonfile.New(files, filepath.Join(p.State, "images")),
		marks:        imagesjsonfile.NewMarks(files, filepath.Join(p.State, "builds")),
		book:         accountsjsonfile.NewBook(lockable{files}, filepath.Join(p.State, "vault.json")),
		reservations: portsjsonfile.NewReservations(files, filepath.Join(p.State, "leases.json")),
		declarations: portsjsonfile.NewDeclarations(files, filepath.Join(p.Prefs, "public.json")),
		applied:      settingsjsonfile.NewApplied(files, filepath.Join(p.State, "applied")),
		uptime:       machinesjsonfile.NewUptime(files, filepath.Join(p.State, "uptime.json")),
		ledger:       machinefilejsonfile.NewLedger(lockable{files}, filepath.Join(p.State, "ledger.json")),
	}
}

// value: the machine's files with the lock a store takes around a change.
type lockable struct{ local.FS }

func (lockable) Lock(path string) (func(), error) { return local.Exclusive(context.Background(), path) }

func (lockable) Hold(ctx context.Context, path string) (func(), error) {
	return local.Exclusive(ctx, path)
}

func (s *stores) name() setupcontract.NamePrefs         { return setupcontract.Name(s.prefs) }
func (s *stores) identity() setupcontract.IdentityPrefs { return setupcontract.Identity(s.prefs) }
func (s *stores) macs() machinescontract.MacsPrefs      { return machinescontract.Macs(s.prefs) }
func (s *stores) vmUser() machinescontract.VMUserPrefs  { return machinescontract.VMUser(s.prefs) }

func (s *stores) ports() portscontract.PortsPrefs { return portscontract.Ports(s.prefs) }
func (s *stores) neverSleeps() awakecontract.NeverSleepsPrefs {
	return awakecontract.NeverSleeps(s.prefs)
}
