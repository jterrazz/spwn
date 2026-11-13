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
