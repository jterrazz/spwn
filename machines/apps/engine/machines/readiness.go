package machines

import (
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type ReadinessSetting struct {
	// value: the setting that makes the check true, its state as measured on
	// the Mac, whether it applies only as root, and whether Machines keeps a
	// record of applying it there.
	Name     string
	State    catalog.State
	Root     bool
	Recorded bool
}

type ReadinessFacts struct {
	// value: the Mac, as the owner names it in a command.
	Mac                                        string
	MacsServer                                 bool
	RestartsAfterPowerCut, NeverSleeps, LogsIn ReadinessSetting
	// value: the Mac's own sleep timer is off (pmset sleep 0, System
	// Settings' "Prevent automatic sleeping"), whatever the engine holds.
	MacNeverSleeps bool
	// value: FileVault on, and FileVault's planned restart supported, each
	// ok, absent or unknown as the Mac reads it.
	FileVault, AuthRestart catalog.State
	// value: the engine's job is installed, so the engine starts at login;
	// false with Known false is a reading that did not answer.
	EngineAtLogin, EngineKnown bool
}

type ReadinessItem struct {
	Name  string
	State catalog.State
	Value string
	// value: the command that makes the check true, when one does.
	Fix string
}

const (
	ItemRestartsAfterPowerCut = "restarts-after-a-power-cut"
	ItemNeverSleeps           = "never-sleeps"
	ItemLogsIn                = "logs-in-by-itself"
	ItemDiskUnlock            = "disk-unlock-after-a-power-cut"
	ItemPlannedRestarts       = "planned-restarts-unlock-themselves"
	ItemEngineAtLogin         = "engine-starts-at-login"
)

func Readiness(f ReadinessFacts) []ReadinessItem {
	// invariant: SPEC 7.6: a Mac that hosts a server, or has any always-on
	// check in force whoever set it, has its readiness, six checks in the one
	// order, each measured; any other Mac has none. A disk that FileVault
	// guards needs you after a power cut, however ready the rest is.
	if !f.MacsServer && !f.MacNeverSleeps && !slices.ContainsFunc([]ReadinessSetting{f.RestartsAfterPowerCut, f.NeverSleeps, f.LogsIn}, func(s ReadinessSetting) bool { return s.State == catalog.StateOK }) {
		return nil
	}
	never := setting(ItemNeverSleeps, f.NeverSleeps)
	if never.State != catalog.StateOK && f.MacNeverSleeps {
		never = ReadinessItem{Name: ItemNeverSleeps, State: catalog.StateOK, Value: "set in System Settings"}
	}
	return []ReadinessItem{
		setting(ItemRestartsAfterPowerCut, f.RestartsAfterPowerCut),
		never,
		setting(ItemLogsIn, f.LogsIn),
		diskUnlock(f.FileVault, f.Mac),
		plannedRestarts(f.AuthRestart),
		engineAtLogin(f.EngineAtLogin, f.EngineKnown),
	}
}

func setting(name string, s ReadinessSetting) ReadinessItem {
	fix := "station settings apply " + s.Name
	if s.Root {
		fix = catalog.StationAsRoot("settings apply " + s.Name)
	}
	switch {
	case s.State == catalog.StateOK && s.Recorded:
		return ReadinessItem{Name: name, State: catalog.StateOK, Value: "set by Machines"}
	case s.State == catalog.StateOK:
		return ReadinessItem{Name: name, State: catalog.StateOK, Value: "set by hand"}
	case s.State == catalog.StateDegraded && s.Recorded:
		return ReadinessItem{Name: name, State: catalog.StateDegraded, Value: "changed since", Fix: fix}
	case s.State == catalog.StateDegraded:
		return ReadinessItem{Name: name, State: catalog.StateDegraded, Value: "partly set", Fix: fix}
	case s.State == catalog.StateUnknown, s.State == catalog.StatePending:
		return ReadinessItem{Name: name, State: catalog.StateUnknown}
	}
	return ReadinessItem{Name: name, State: catalog.StateAbsent, Value: "not applied", Fix: fix}
}

func diskUnlock(fileVault catalog.State, mac string) ReadinessItem {
	switch fileVault {
	case catalog.StateOK:
		return ReadinessItem{Name: ItemDiskUnlock, State: catalog.StateDegraded, Value: "needs you", Fix: "station machines unlock " + mac + " (from the same network)"}
	case catalog.StateAbsent:
		return ReadinessItem{Name: ItemDiskUnlock, State: catalog.StateOK, Value: "no password at boot"}
	case catalog.StateDegraded, catalog.StateBroken, catalog.StateUnknown, catalog.StatePending:
	}
	return ReadinessItem{Name: ItemDiskUnlock, State: catalog.StateUnknown}
}

func plannedRestarts(authRestart catalog.State) ReadinessItem {
	switch authRestart {
	case catalog.StateOK:
		return ReadinessItem{Name: ItemPlannedRestarts, State: catalog.StateOK, Value: "supported"}
	case catalog.StateAbsent:
		return ReadinessItem{Name: ItemPlannedRestarts, State: catalog.StateAbsent, Value: "not supported"}
	case catalog.StateDegraded, catalog.StateBroken, catalog.StateUnknown, catalog.StatePending:
	}
	return ReadinessItem{Name: ItemPlannedRestarts, State: catalog.StateUnknown}
}

func engineAtLogin(installed, known bool) ReadinessItem {
	switch {
	case installed:
		return ReadinessItem{Name: ItemEngineAtLogin, State: catalog.StateOK, Value: "installed"}
	case known:
		return ReadinessItem{Name: ItemEngineAtLogin, State: catalog.StateBroken, Value: "not installed", Fix: "station engine start"}
	}
	return ReadinessItem{Name: ItemEngineAtLogin, State: catalog.StateUnknown}
}
