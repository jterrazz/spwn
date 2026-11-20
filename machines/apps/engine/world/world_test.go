package world_test

import (
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func seeded(t *testing.T, doc string) *world.World {
	t.Helper()
	w, err := world.Decode([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func apply(t *testing.T, w *world.World, words ...string) error {
	t.Helper()
	ev, err := world.Parse(words)
	if err != nil {
		t.Fatalf("world %s: %v", strings.Join(words, " "), err)
	}
	return w.Apply(ev)
}

const laptop = `{"this_mac": "mbp", "macs": {"mbp": {"power": {"Lid": "open", "Source": "ac", "HasBattery": true, "Percent": 96}}, "mini": {"power": {"Source": "ac"}, "vms": {"linux-1": {"disk": true, "pid": 7}}}}}`

func TestAnEventOutsideTheClosedListIsRefusedByName(t *testing.T) {
	_, err := world.Parse([]string{"meteor", "mbp"})
	if err == nil || !strings.Contains(err.Error(), `unknown world event "meteor"`) {
		t.Errorf("an unknown event parsed as %v", err)
	}
	if _, err := world.Parse([]string{"reboot"}); err == nil || !strings.Contains(err.Error(), "world reboot takes <mac>") {
		t.Errorf("a reboot of no Mac parsed as %v", err)
	}
}

func TestUnpluggingAndClosingTheLidMoveTheLaptopsReading(t *testing.T) {
	w := seeded(t, laptop)
	for _, ev := range [][]string{{"unplug"}, {"close-lid"}, {"battery", "35"}, {"thermal", "heavy"}, {"low-power", "on"}} {
		if err := apply(t, w, ev...); err != nil {
			t.Fatalf("world %v: %v", ev, err)
		}
	}
	got := w.Macs["mbp"].Power.Reading
	want := awake.Reading{Lid: "closed", Source: "battery", HasBattery: true, Percent: 35, LowPower: true, Thermal: awake.ThermalHeavy}
	if got != want {
		t.Errorf("the laptop reads %+v, want %+v", got, want)
	}
}

func TestAMacWithoutTheCapabilityRefusesItsEvent(t *testing.T) {
	w := seeded(t, strings.Replace(laptop, `"this_mac": "mbp"`, `"this_mac": "mini"`, 1))
	if err := apply(t, w, "close-lid"); err == nil || !strings.Contains(err.Error(), "mini has no lid") {
		t.Errorf("a lid closed on a desktop as %v", err)
	}
	if err := apply(t, w, "unplug"); err == nil || !strings.Contains(err.Error(), "mini has no battery") {
		t.Errorf("a desktop unplugged as %v", err)
	}
}

func TestWhatStartsListeningInAVMStopsWhenUnlistened(t *testing.T) {
	w := seeded(t, laptop)
	if err := apply(t, w, "listen", "linux-1", "*:5173", "node"); err != nil {
		t.Fatal(err)
	}
	if err := apply(t, w, "listen", "mbp", "[::1]:8080", "python"); err != nil {
		t.Fatal(err)
	}
	vm := w.Macs["mini"].VMs["linux-1"].Listening
	if len(vm) != 1 || vm[0].Port != 5173 || vm[0].Bind != "*" || vm[0].Process != "node" {
		t.Errorf("linux-1 listens on %+v", vm)
	}
	if mac := w.Macs["mbp"].Listening; len(mac) != 1 || mac[0].Bind != "::1" || mac[0].Port != 8080 {
		t.Errorf("mbp listens on %+v", mac)
	}
	if err := apply(t, w, "unlisten", "linux-1", "5173"); err != nil {
		t.Fatal(err)
	}
	if left := w.Macs["mini"].VMs["linux-1"].Listening; len(left) != 0 {
		t.Errorf("linux-1 still listens on %+v", left)
	}
	if err := apply(t, w, "listen", "nowhere", "*:1", "x"); err == nil {
		t.Error("a machine of no world listened")
	}
}

func TestAFaultFailsItsCallTheTimesScriptedThenIsSpent(t *testing.T) {
	w := seeded(t, laptop)
	if err := apply(t, w, "fault", "Installer.Install", "2", "times"); err != nil {
		t.Fatal(err)
	}
	got := make([]bool, 0, 3)
	for range 3 {
		got = append(got, w.Fails("Installer.Install"))
	}
	if got[0] != true || got[1] != true || got[2] != false {
		t.Errorf("Installer.Install failed %v, want two failures then none", got)
	}
	if w.Fails("Jobs.Start") {
		t.Error("an unscripted call failed")
	}
	if err := apply(t, w, "fault", "Jobs.Start"); err != nil {
		t.Fatal(err)
	}
	first, second := w.Fails("Jobs.Start"), w.Fails("Jobs.Start")
	if !first || !second {
		t.Error("a fault with no count stopped failing")
	}
}

func TestAHoldHoldsOneCallAndNeverFailsIt(t *testing.T) {
	w := seeded(t, laptop)
	if err := apply(t, w, "hold", "Bundler.Bundle"); err != nil {
		t.Fatal(err)
	}
	if w.Fails("Bundler.Bundle") {
		t.Error("a held call failed")
	}
	if !w.Holds("Bundler.Bundle") {
		t.Error("the held call was not held")
	}
	if w.Holds("Bundler.Bundle") {
		t.Error("a hold held a second call")
	}
}

func TestAHoldTheWorldDeclaresForOneVMKeepsItsCallsForTheWorldsLife(t *testing.T) {
	w := seeded(t, `{"this_mac": "mbp", "macs": {"mbp": {}, "mini": {}}, "script": [{"call": "Hypervisor.Boot", "hold": true, "mac": "mini", "vm": "linux-2"}]}`)
	if w.Holds("Hypervisor.Boot") {
		t.Error("a hold declared for one VM held the next call of any VM")
	}
	for range 3 {
		if !w.Keeps("Hypervisor.Boot", "mini", "linux-2") {
			t.Fatal("a hold declared for linux-2 on mini let one of its calls go")
		}
	}
	if w.Keeps("Hypervisor.Boot", "mbp", "linux-2") || w.Keeps("Hypervisor.Boot", "mini", "linux-3") || w.Keeps("Hypervisor.Clone", "mini", "linux-2") {
		t.Error("a hold declared for linux-2's boot on mini kept another call")
	}
	if w.Fails("Hypervisor.Boot") {
		t.Error("a declared hold failed its call")
	}
}

func TestAnUnholdLetsGoOfTheHoldsItNamesAndOfNothingElse(t *testing.T) {
	two := `{"this_mac": "mbp", "macs": {"mbp": {}, "mini": {}}, "script": [{"call": "Hypervisor.Boot", "hold": true, "mac": "mini", "vm": "linux-2"}, {"call": "Hypervisor.Clone", "hold": true, "mac": "mbp", "vm": "linux-4"}, {"call": "Jobs.Start"}]}`
	w := seeded(t, two)
	if err := apply(t, w, "unhold", "mini", "linux-2"); err != nil {
		t.Fatal(err)
	}
	if w.Keeps("Hypervisor.Boot", "mini", "linux-2") || !w.Keeps("Hypervisor.Clone", "mbp", "linux-4") || !w.Fails("Jobs.Start") {
		t.Errorf("an unhold of linux-2 on mini left %+v", w.Script)
	}
	w = seeded(t, two)
	if err := apply(t, w, "hold", "Bundler.Bundle"); err != nil {
		t.Fatal(err)
	}
	if err := apply(t, w, "unhold"); err != nil {
		t.Fatal(err)
	}
	if len(w.Script) != 1 || !w.Fails("Jobs.Start") {
		t.Errorf("an unhold of every hold left %+v", w.Script)
	}
	for range 2 {
		if err := apply(t, w, "unhold"); err != nil {
			t.Errorf("an unhold with nothing held failed: %v", err)
		}
	}
	if err := apply(t, w, "unhold", "mini", "linux-9"); err != nil {
		t.Errorf("an unhold of a VM nothing holds failed: %v", err)
	}
	if err := apply(t, w, "unhold", "nowhere", "linux-2"); err == nil {
		t.Error("an unhold on a Mac the world lacks was taken")
	}
	if err := apply(t, w, "unhold", "mini"); err == nil {
		t.Error("an unhold naming a Mac and no VM was taken")
	}
}

func TestAWorldReadsBackAsItWasWritten(t *testing.T) {
	w := seeded(t, laptop)
	w.Mac("mbp").Listening = []ports.Listener{{Port: 5000, Bind: "*", Process: "ControlCenter", PID: 1}}
	data, err := w.Encode()
	if err != nil {
		t.Fatal(err)
	}
	again := seeded(t, string(data))
	if l := again.Macs["mbp"].Listening; len(l) != 1 || l[0].Process != "ControlCenter" || again.Macs["mbp"].Power.Percent != 96 {
		t.Errorf("the world read back as %+v", again.Macs["mbp"])
	}
}

func TestAnotherHypervisorStandsInWhereStationsOwnWas(t *testing.T) {
	w := seeded(t, `{"this_mac": "mbp", "macs": {"mbp": {}, "bare": {"hypervisor": "none"}}}`)
	w.UseHypervisor("vz")
	if got := w.Macs["mbp"].Hypervisor; got != "vz" {
		t.Errorf("the Mac with station's hypervisor has %q, want vz", got)
	}
	if got := w.Macs["bare"].Hypervisor; got != "none" {
		t.Errorf("the Mac with no hypervisor has %q, want none still", got)
	}
}

func TestTwoMacsHoldingOneVMNameAnswerTheFirstByName(t *testing.T) {
	for range 64 {
		w := seeded(t, `{"this_mac": "mbp", "macs": {"mbp": {"vms": {"linux-1": {"disk": true}}}, "aa-mini": {"vms": {"linux-1": {"disk": true}}}, "zz-mini": {"vms": {"linux-1": {"disk": true}}}}}`)
		if got := w.Holder("linux-1"); got != "aa-mini" {
			t.Fatalf("Holder(linux-1) = %q, want aa-mini", got)
		}
	}
}

func TestAnEngineLogEventAppendsALineToItsMacsLog(t *testing.T) {
	w := seeded(t, laptop)
	for _, line := range []string{"engine started", "vm linux-1 up"} {
		if err := apply(t, w, "engine-log", "mini", line); err != nil {
			t.Fatal(err)
		}
	}
	if got := w.Macs["mini"].EngineLog; strings.Join(got, "|") != "engine started|vm linux-1 up" || len(w.Macs["mbp"].EngineLog) != 0 {
		t.Errorf("logs = %q on mini, %q on mbp; want the two lines on mini alone", got, w.Macs["mbp"].EngineLog)
	}
	if err := apply(t, w, "engine-log", "nowhere", "x"); err == nil {
		t.Error("a line for a Mac the world does not know was taken")
	}
}
