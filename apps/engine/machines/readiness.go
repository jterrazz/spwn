package machines

import "github.com/jterrazz/studio-station/apps/engine/catalog"

type ReadinessSetting struct {
	// value: the setting that makes the item true, its state on the Mac, and
	// whether it applies only as root.
	Name  string
	State catalog.State
	Root  bool
}

type ReadinessFacts struct {
	// value: the Mac, as the owner names it in a command.
	Mac                                        string
	MacsServer                                 bool
	RestartsAfterPowerCut, NeverSleeps, LogsIn ReadinessSetting
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
	// value: the command that makes the item true, when one does.
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
	// invariant: SPEC 7.6: a Mac that hosts a server, or holds Never sleeps,
	// has its readiness, six items in the one order, each measured; any
	// other Mac has none. A disk that FileVault guards needs you after a
	// power cut, however ready the rest is.
	if !f.MacsServer && f.NeverSleeps.State != catalog.StateOK {
		return nil
	}
	return []ReadinessItem{
		setting(ItemRestartsAfterPowerCut, f.RestartsAfterPowerCut),
		setting(ItemNeverSleeps, f.NeverSleeps),
		setting(ItemLogsIn, f.LogsIn),
		diskUnlock(f.FileVault, f.Mac),
		plannedRestarts(f.AuthRestart),
		engineAtLogin(f.EngineAtLogin, f.EngineKnown),
	}
}

func setting(name string, s ReadinessSetting) ReadinessItem {
	fix, root := "station settings apply "+s.Name, ""
	if s.Root {
		root = ", as root"
	}
	switch s.State {
	case catalog.StateOK:
		return ReadinessItem{Name: name, State: catalog.StateOK, Value: "applied"}
	case catalog.StateDegraded:
		return ReadinessItem{Name: name, State: catalog.StateDegraded, Value: "changed since" + root, Fix: fix}
	case catalog.StateUnknown, catalog.StatePending:
		return ReadinessItem{Name: name, State: catalog.StateUnknown}
	case catalog.StateAbsent, catalog.StateBroken:
	}
	return ReadinessItem{Name: name, State: catalog.StateAbsent, Value: "not applied" + root, Fix: fix}
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
