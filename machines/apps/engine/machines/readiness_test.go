package machines_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

func ready() machines.ReadinessFacts {
	applied := func(name string, root bool) machines.ReadinessSetting {
		return machines.ReadinessSetting{Name: name, State: catalog.StateOK, Root: root, Recorded: true}
	}
	return machines.ReadinessFacts{
		Mac: "mac-mini", MacsServer: true,
		RestartsAfterPowerCut: applied("restart-after-power-cut", true), NeverSleeps: applied("never-sleeps", false), LogsIn: applied("auto-login", true),
		FileVault: catalog.StateAbsent, AuthRestart: catalog.StateOK, EngineAtLogin: true, EngineKnown: true,
	}
}

func TestAMacHasItsReadinessOnlyWhenItMacsAServerOrNeverSleeps(t *testing.T) {
	f := ready()
	if got := machines.Readiness(f); len(got) != 6 {
		t.Fatalf("a Mac hosting a server has %d readiness checks, want the six of SPEC 7.6", len(got))
	}
	f.MacsServer = false
	if got := machines.Readiness(f); len(got) != 6 {
		t.Fatalf("a Mac that never sleeps has %d readiness checks, want six", len(got))
	}
	f.NeverSleeps.State, f.RestartsAfterPowerCut.State, f.LogsIn.State = catalog.StateAbsent, catalog.StateAbsent, catalog.StateAbsent
	if got := machines.Readiness(f); got != nil {
		t.Fatalf("a Mac with no server and no always-on item has readiness %v, want none", got)
	}
	f.RestartsAfterPowerCut.State = catalog.StateOK
	if got := machines.Readiness(f); len(got) != 6 {
		t.Fatalf("a Mac that restarts after a power cut has %d readiness checks, want six", len(got))
	}
	f.RestartsAfterPowerCut.State, f.MacNeverSleeps = catalog.StateAbsent, true
	if got := machines.Readiness(f); len(got) != 6 {
		t.Fatalf("a Mac set never to sleep in System Settings has %d readiness checks, want six", len(got))
	}
}

func TestEachCheckSaysWhoSetItOnTheMac(t *testing.T) {
	f := ready()
	f.RestartsAfterPowerCut.Recorded, f.NeverSleeps.State, f.MacNeverSleeps = false, catalog.StateAbsent, true
	items := machines.Readiness(f)
	if items[0].State != catalog.StateOK || items[0].Value != "set by hand" || items[0].Fix != "" {
		t.Errorf("restarts after a power cut, in place with no record: %+v, want set by hand", items[0])
	}
	if items[1].State != catalog.StateOK || items[1].Value != "set in System Settings" || items[1].Fix != "" {
		t.Errorf("never sleeps, the Mac's own sleep off: %+v, want set in System Settings", items[1])
	}
	if items[2].State != catalog.StateOK || items[2].Value != "set by Machines" {
		t.Errorf("logs in by itself, applied by Machines: %+v", items[2])
	}
}

func TestASettingPartlyThereThatMachinesNeverAppliedIsNotChangedSince(t *testing.T) {
	f := ready()
	f.LogsIn.State, f.LogsIn.Recorded = catalog.StateDegraded, false
	if item := machines.Readiness(f)[2]; item.State != catalog.StateDegraded || item.Value != "partly set" {
		t.Errorf("logs in by itself, partly there with no record: %+v, want partly set", item)
	}
}

func TestADiskThatFileVaultGuardsNeedsYouAfterAPowerCut(t *testing.T) {
	f := ready()
	f.FileVault = catalog.StateOK
	disk := machines.Readiness(f)[3]
	if disk.Name != machines.ItemDiskUnlock || disk.State != catalog.StateDegraded || disk.Fix != "station machines unlock mac-mini (from the same network)" {
		t.Fatalf("FileVault on reads %+v, want needs you with the unlock from the same network", disk)
	}
	f.FileVault = catalog.StateUnknown
	if disk := machines.Readiness(f)[3]; disk.State != catalog.StateUnknown || disk.Fix != "" {
		t.Fatalf("an unread FileVault reads %+v, want unknown, never a fix", disk)
	}
}

func TestASettingNotAppliedNamesItsFixAndRootOnlyWhereItNeedsIt(t *testing.T) {
	f := ready()
	f.RestartsAfterPowerCut.State, f.LogsIn.State = catalog.StateAbsent, catalog.StateDegraded
	items := machines.Readiness(f)
	if items[0].Fix != "sudo ~/.local/bin/station settings apply restart-after-power-cut" || items[0].Value != "not applied" || items[0].State != catalog.StateAbsent {
		t.Errorf("restarts after a power cut, not applied: %+v", items[0])
	}
	if items[2].Fix != "sudo ~/.local/bin/station settings apply auto-login" || items[2].Value != "changed since" || items[2].State != catalog.StateDegraded {
		t.Errorf("logs in by itself, changed since: %+v", items[2])
	}
	if items[1].Fix != "" || items[1].State != catalog.StateOK {
		t.Errorf("never sleeps, applied: %+v, want no fix", items[1])
	}
	f.NeverSleeps.State = catalog.StateDegraded
	f.MacsServer = true
	if items := machines.Readiness(f); items[1].Fix != "station settings apply never-sleeps" || items[1].Value != "changed since" {
		t.Errorf("never sleeps, changed since: %+v, want its fix, not as root", items[1])
	}
}

func TestAnEngineJobNotInstalledIsBrokenWhileOneNotReadIsUnknown(t *testing.T) {
	f := ready()
	f.EngineAtLogin = false
	if item := machines.Readiness(f)[5]; item.State != catalog.StateBroken || item.Fix != "station engine start" {
		t.Errorf("no engine job: %+v, want broken with station engine start", item)
	}
	f.EngineKnown = false
	if item := machines.Readiness(f)[5]; item.State != catalog.StateUnknown {
		t.Errorf("engine job not read: %+v, want unknown", item)
	}
}
