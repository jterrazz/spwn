package settings_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

func checked(t *testing.T, name string) catalog.CheckedSetting {
	t.Helper()
	it, ok := settings.CheckedNamed(name)
	if !ok {
		t.Fatalf("no checked setting %q", name)
	}
	return it
}

func on(value string) settings.Fact {
	if value == "off" {
		return settings.Fact{State: catalog.StateAbsent, Value: value}
	}
	return settings.Fact{State: catalog.StateOK, Value: value}
}

func TestJudgeCheckedHoldsEachMacToWhatItShouldHave(t *testing.T) {
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	laptop := settings.Expect(machine.Capabilities{Battery: true, Lid: true}, false, now)
	desktop := settings.Expect(machine.Capabilities{}, false, now)
	alwaysOn := settings.Expect(machine.Capabilities{}, true, now)
	for _, c := range []struct {
		name    string
		setting string
		facts   map[catalog.Fact]settings.Fact
		expect  settings.Expectation
		want    settings.CheckedState
		value   string
	}{
		{"FileVault on", "filevault", map[catalog.Fact]settings.Fact{catalog.FactFileVault: on("on")}, laptop, settings.AsExpected, "on"},
		{"FileVault off", "filevault", map[catalog.Fact]settings.Fact{catalog.FactFileVault: on("off")}, laptop, settings.NeedsFix, "off"},
		{"SIP disabled", "sip", map[catalog.Fact]settings.Fact{catalog.FactSIP: on("off")}, desktop, settings.NeedsFix, "off"},
		{"Gatekeeper off", "gatekeeper", map[catalog.Fact]settings.Fact{catalog.FactGatekeeper: on("off")}, desktop, settings.NeedsFix, "off"},
		{"automatic updates off", "automatic-updates", map[catalog.Fact]settings.Fact{catalog.FactAutoUpdates: on("off")}, laptop, settings.NeedsFix, "off"},
		{"a macOS update waiting", "macos-updates", map[catalog.Fact]settings.Fact{catalog.FactSystemUpdates: {State: catalog.StateDegraded, Value: "1 pending"}}, laptop, settings.NeedsFix, "1 update waiting"},
		{"macOS up to date", "macos-updates", map[catalog.Fact]settings.Fact{catalog.FactSystemUpdates: on("up to date")}, laptop, settings.AsExpected, "up to date"},
		{"Remote login off on a laptop", "remote-login", map[catalog.Fact]settings.Fact{catalog.FactRemoteLogin: on("off")}, laptop, settings.AsExpected, "off"},
		{"Remote login on on a laptop", "remote-login", map[catalog.Fact]settings.Fact{catalog.FactRemoteLogin: on("on")}, laptop, settings.NeedsFix, "on"},
		{"Remote login on on an always-on Mac", "remote-login", map[catalog.Fact]settings.Fact{catalog.FactRemoteLogin: on("on")}, alwaysOn, settings.AsExpected, "on"},
		{"Remote login off on an always-on Mac", "remote-login", map[catalog.Fact]settings.Fact{catalog.FactRemoteLogin: on("off")}, alwaysOn, settings.NeedsFix, "off"},
		{"a laptop that locks after 10 min", "screen-lock", map[catalog.Fact]settings.Fact{catalog.FactScreenLock: on("0"), catalog.FactDisplaySleep: on("10")}, laptop, settings.NeedsFix, "locks after 10 min"},
		{"a laptop that locks after 2 min", "screen-lock", map[catalog.Fact]settings.Fact{catalog.FactScreenLock: on("0"), catalog.FactDisplaySleep: on("2")}, laptop, settings.AsExpected, "locks after 2 min"},
		{"a desktop whose display stays on 10 min, then locks at once", "screen-lock", map[catalog.Fact]settings.Fact{catalog.FactScreenLock: on("0"), catalog.FactDisplaySleep: on("10")}, desktop, settings.AsExpected, "locks when the display turns off"},
		{"a desktop that locks 10 min after the display turns off", "screen-lock", map[catalog.Fact]settings.Fact{catalog.FactScreenLock: on("600")}, desktop, settings.NeedsFix, "locks 10 min after the display turns off"},
		{"a screen that never locks", "screen-lock", map[catalog.Fact]settings.Fact{catalog.FactScreenLock: on("off"), catalog.FactDisplaySleep: on("2")}, laptop, settings.NeedsFix, "never locks"},
		{"a backup 2 d ago", "time-machine", map[catalog.Fact]settings.Fact{catalog.FactTimeMachine: on(now.Add(-48 * time.Hour).Format(time.RFC3339))}, laptop, settings.AsExpected, "last backup 2 d ago"},
		{"a backup 9 d ago", "time-machine", map[catalog.Fact]settings.Fact{catalog.FactTimeMachine: on(now.Add(-9 * 24 * time.Hour).Format(time.RFC3339))}, laptop, settings.NeedsFix, "last backup 9 d ago"},
		{"no backup ever", "time-machine", map[catalog.Fact]settings.Fact{catalog.FactTimeMachine: {State: catalog.StateAbsent, Value: "never"}}, laptop, settings.NeedsFix, "no backup yet"},
		{"a fact not read in its budget", "time-machine", map[catalog.Fact]settings.Fact{catalog.FactTimeMachine: {State: catalog.StateUnknown, Detail: "no answer in 2 s"}}, laptop, settings.CheckedUnknown, "no answer in 2 s"},
		{"a fact the Mac could not look at", "firewall", map[catalog.Fact]settings.Fact{}, laptop, settings.CheckedUnknown, "could not be read"},
		{"a laptop's display time unknown", "screen-lock", map[catalog.Fact]settings.Fact{catalog.FactScreenLock: on("0")}, laptop, settings.CheckedUnknown, "could not be read"},
	} {
		t.Run(c.name, func(t *testing.T) {
			it := checked(t, c.setting)
			row := settings.JudgeChecked(it, true, c.facts, c.expect, false)
			if row.Checked != c.want || row.Value != c.value {
				t.Fatalf("judged %s %q, want %s %q", row.Checked, row.Value, c.want, c.value)
			}
			if row.Group != settings.Checked || row.State != "" {
				t.Errorf("a checked row is in the Checked group with no Set by station state, got %s %q", row.Group, row.State)
			}
			if (row.Checked == settings.NeedsFix) != (row.Fix.Path == it.Fix) || row.Fix.Call != "" || row.Fix.Command != "" {
				t.Errorf("fix = %+v; a needs-fix row is fixed where macOS sets it, never by a station call", row.Fix)
			}
		})
	}
}

func TestJudgeCheckedLeavesALivesOnOrAnOffThisMacUnread(t *testing.T) {
	it := checked(t, "firewall")
	off := map[catalog.Fact]settings.Fact{catalog.FactFirewall: on("off")}
	expect := settings.Expect(machine.Capabilities{Battery: true}, false, time.Time{})
	if row := settings.JudgeChecked(it, false, off, expect, true); row.Checked != settings.CheckedNotForThisMachine || row.Why != "Mac only" {
		t.Errorf("on a VM: %s %q, want not for this machine, Mac only", row.Checked, row.Why)
	}
	if row := settings.JudgeChecked(it, true, off, expect, true); row.Checked != settings.CheckedOffThisMac {
		t.Errorf("off here: %s, want off-here, the firewall off left unjudged", row.Checked)
	}
}

func TestFindingsHoldACheckedSettingNotAsExpected(t *testing.T) {
	it := checked(t, "firewall")
	row := settings.JudgeChecked(it, true, map[catalog.Fact]settings.Fact{catalog.FactFirewall: on("off")}, settings.Expectation{}, false)
	got := settings.Findings([]settings.Row{row}, machine.Ref{Name: "jterrazz-macbook-pro"})
	if len(got) != 1 || got[0].Subject != "firewall" || got[0].Text != "Firewall: off" || got[0].Fix.Path != "System Settings > Network > Firewall" {
		t.Fatalf("findings = %+v, want Firewall off fixed in System Settings", got)
	}
	row.Checked = settings.CheckedOffThisMac
	if got := settings.Findings([]settings.Row{row}, machine.Ref{}); len(got) != 0 {
		t.Errorf("an off-here check still needs you: %+v", got)
	}
}
