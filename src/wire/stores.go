package wire

import (
	"context"
	"path/filepath"

	awakecontract "github.com/jterrazz/studio-station/src/awake/contract"

	accountsjsonfile "github.com/jterrazz/studio-station/src/accounts/adapter/jsonfile"
	accountscontract "github.com/jterrazz/studio-station/src/accounts/contract"
	corejsonfile "github.com/jterrazz/studio-station/src/core/adapter/jsonfile"
	corecontract "github.com/jterrazz/studio-station/src/core/contract"
	"github.com/jterrazz/studio-station/src/core/store"
	imagesjsonfile "github.com/jterrazz/studio-station/src/images/adapter/jsonfile"
	imagescontract "github.com/jterrazz/studio-station/src/images/contract"
	machinesjsonfile "github.com/jterrazz/studio-station/src/machines/adapter/jsonfile"
	machinescontract "github.com/jterrazz/studio-station/src/machines/contract"
	"github.com/jterrazz/studio-station/src/platform/local"
	portsjsonfile "github.com/jterrazz/studio-station/src/ports/adapter/jsonfile"
	portscontract "github.com/jterrazz/studio-station/src/ports/contract"
	settingsjsonfile "github.com/jterrazz/studio-station/src/settings/adapter/jsonfile"
	settingscontract "github.com/jterrazz/studio-station/src/settings/contract"
	"github.com/jterrazz/studio-station/src/setup"
	setupcontract "github.com/jterrazz/studio-station/src/setup/contract"
)

// invariant: the preference groups in the order settings.json writes them,
// each group named by its one owner's contract.
func preferenceGroups() []string {
	return []string{
		setupcontract.GroupName, machinescontract.GroupMacs, machinescontract.GroupVMUser,
		accountscontract.GroupShareList, accountscontract.GroupV2ShareList, setupcontract.GroupUserEmail, setupcontract.GroupUserName,
		portscontract.GroupPorts, settingscontract.GroupOffHere, awakecontract.GroupNeverSleeps,
	}
}

type preferences interface {
	corecontract.Preferences
	store.Migrator
	File() string
	// value: moves the file station v2 kept them in, once; moved is whether
	// this call moved it.
	MoveRetired(ctx context.Context) (moved bool, err error)
}

type machineStore interface {
	machinescontract.MachineStore
	store.Migrator
}

type imageStore interface {
	imagescontract.ImageStore
	store.Migrator
}

type shareStore interface {
	accountscontract.ShareStore
	store.Migrator
}

type reservationStore interface {
	portscontract.ReservationStore
	store.Migrator
}

type declarationStore interface {
	portscontract.DeclarationStore
	store.Migrator
}

type uptimeStore interface {
	machinescontract.UptimeStore
	store.Migrator
}

type appliedStore interface {
	settingscontract.AppliedStore
	store.Migrator
}

// value: every store station keeps on this machine, each built once here
// over its file; nothing else constructs one.
type stores struct {
	prefs        preferences
	machines     machineStore
	images       imageStore
	shares       shareStore
	reservations reservationStore
	declarations declarationStore
	applied      appliedStore
	uptime       uptimeStore
}

func newStores(files local.FS, p setup.Paths) *stores {
	return &stores{
		prefs: corejsonfile.NewPreferences(files, p.Prefs, preferenceGroups()).Renaming(machinescontract.RetiredGroupMacs, machinescontract.GroupMacs).
			Renaming(setupcontract.RetiredGroupName, setupcontract.GroupName).Renaming(portscontract.RetiredGroupPorts, portscontract.GroupPorts),
		machines:     machinesjsonfile.New(files, filepath.Join(p.State, "vms")),
		images:       imagesjsonfile.New(files, filepath.Join(p.State, "images")),
		shares:       accountsjsonfile.NewShares(files, filepath.Join(p.State, "shares.json"), filepath.Join(p.State, "vms")),
		reservations: portsjsonfile.NewReservations(files, filepath.Join(p.State, "leases.json")),
		declarations: portsjsonfile.NewDeclarations(files, filepath.Join(p.Prefs, "public.json")),
		applied:      settingsjsonfile.NewApplied(files, filepath.Join(p.State, "applied")),
		uptime:       machinesjsonfile.NewUptime(files, filepath.Join(p.State, "uptime.json")),
	}
}

func (s *stores) migrators() []store.Migrator {
	return []store.Migrator{s.prefs, s.machines, s.images, s.shares, s.reservations, s.declarations, s.applied, s.uptime}
}

func (s *stores) name() setupcontract.NamePrefs         { return setupcontract.Name(s.prefs) }
func (s *stores) identity() setupcontract.IdentityPrefs { return setupcontract.Identity(s.prefs) }
func (s *stores) macs() machinescontract.MacsPrefs      { return machinescontract.Macs(s.prefs) }
func (s *stores) vmUser() machinescontract.VMUserPrefs  { return machinescontract.VMUser(s.prefs) }
func (s *stores) shareList() accountscontract.ShareListPrefs {
	return accountscontract.ShareList(s.prefs)
}
func (s *stores) ports() portscontract.PortsPrefs { return portscontract.Ports(s.prefs) }
func (s *stores) neverSleeps() awakecontract.NeverSleepsPrefs {
	return awakecontract.NeverSleeps(s.prefs)
}
