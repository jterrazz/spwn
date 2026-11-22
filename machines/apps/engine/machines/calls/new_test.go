package calls_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	machinesfake "github.com/jterrazz/studio-station/apps/engine/machines/adapter/fake"
	machines "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	machinescontract "github.com/jterrazz/studio-station/apps/engine/machines/contract"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

// value: the machine files of a Mac: base-ubuntu's; docs-bot, the
// owner's own, extending it; and base-macos's, the built-in macOS
// image's own file.
type files struct{}

func (files) Template(_ context.Context, name string) (model.Template, error) {
	switch name {
	case catalog.LinuxImage:
		return model.Template{Name: name, OS: catalog.OSUbuntu, Lifetime: model.Ephemeral, Size: model.Resources{CPUs: 4, MemoryBytes: 8 << 30, DiskBytes: 20e9}}, nil
	case "docs-bot":
		return model.Template{Name: name, OS: catalog.OSUbuntu, Lifetime: model.Persistent, Size: model.Resources{MemoryBytes: 3 << 30}, Chain: []string{catalog.LinuxImage, name}}, nil
	case catalog.MacOSImage:
		return model.Template{Name: name, OS: catalog.OSMacOS, Lifetime: model.Persistent, Size: model.Resources{CPUs: 4, MemoryBytes: 8 << 30, DiskBytes: 80e9}}, nil
	}
	return model.Template{}, &model.UnknownImageError{Image: name}
}

func (files) Drift(context.Context, string, string) ([]string, error) { return nil, nil }

func unasking(t *testing.T) *machines.Machinist {
	t.Helper()
	w := &world.World{ThisMac: "mac"}
	prefs := memoryPrefs{}
	p := machines.Prefs{Name: setupcontract.Name(prefs), Macs: machinescontract.Macs(prefs), VMUser: machinescontract.VMUser(prefs)}
	if err := p.Name.Set("mac"); err != nil {
		t.Fatal(err)
	}
	linux := catalog.Image{Name: catalog.LinuxImage, Machinefile: catalog.LinuxImage, Recipe: "images/linux.sh", Kind: catalog.Linux, Distro: catalog.DistroUbuntu, Defaults: catalog.ImageDefaults{CPUs: 4, MemoryBytes: 4 << 30, DiskBytes: 20e9}}
	macos := catalog.Image{Name: catalog.MacOSImage, Machinefile: catalog.MacOSImage, Kind: catalog.MacOS, Distro: catalog.DistroMacOS, Defaults: catalog.ImageDefaults{CPUs: 4, MemoryBytes: 8 << 30, DiskBytes: 80e9}}
	return machines.New(&stillMac{at: time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)}, machinesfake.HypervisorOf(w, "mac", "hypervisor", nil), p, machinesfake.StoreOf(w, "mac"), noTailnet{}, &images{built: true}, &idleEngine{}, nil, machines.Options{
		ThisMachine: catalog.Mac, Images: []catalog.Image{linux, macos}, KnownHosts: "/state/known_hosts", Logs: "/state/logs", Templates: files{},
		Hardware: func(context.Context) coremachine.Hardware { return coremachine.Hardware{Memory: 16 << 30, CPUs: 8} },
	})
}

func TestACreateAskedNothingButItsFileTakesWhatTheFileSays(t *testing.T) {
	m := unasking(t)
	got, err := m.Unasked(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	r := got.Request
	if r.From != catalog.LinuxImage || r.Image != catalog.LinuxImage || r.Size.MemoryBytes != 8<<30 || r.Size.DiskBytes != 20e9 || r.Lifetime != model.Ephemeral || got.Builtin != catalog.LinuxImage || got.Kind != catalog.Linux {
		t.Errorf("no file asked: %+v; want base-ubuntu's own 8 GB, ephemeral, a server offered of its image", got)
	}
	own, err := m.Unasked(t.Context(), "docs-bot")
	if err != nil {
		t.Fatal(err)
	}
	if own.Request.Size.MemoryBytes != 3<<30 || own.Request.Size.CPUs != 4 || own.Request.Lifetime != model.Persistent || own.Builtin != "" {
		t.Errorf("the owner's docs-bot: %+v; want its 3 GB and persistence, the image's 4 CPUs, and no server", own)
	}
	mac, err := m.Unasked(t.Context(), catalog.MacOSImage)
	if err != nil || mac.Request.Image != catalog.MacOSImage || mac.Kind != catalog.MacOS || mac.Builtin != "" {
		t.Errorf("base-macos: %+v, %v; want its own macOS image and no server, a server being Linux", mac, err)
	}
	if _, missing := m.Unasked(t.Context(), "nothing"); !errors.As(missing, new(*model.UnknownImageError)) {
		t.Errorf("a file that is not there: %v; want its refusal", missing)
	}
	name, err := m.NameFor(catalog.Linux, model.AllMachines{Macs: []model.MacView{{Name: "mac", ThisMac: true, VMs: []model.VM{{Name: "linux-1"}}}}})
	if err != nil || name != "linux-2" {
		t.Errorf("the name over a fleet holding linux-1: %q, %v; want linux-2", name, err)
	}
}

func TestAWorkspaceIsNamedApartFromTheMachinefilesItFollows(t *testing.T) {
	m := unasking(t)
	for name, refused := range map[string]bool{"docs-bot": true, catalog.LinuxImage: true, "docs-1": false} {
		_, err := m.CheckCreate(t.Context(), model.CreateRequest{Name: name, From: "docs-bot"})
		var named *model.TemplateNameError
		if errors.As(err, &named) != refused || refused && (named.Name != name || named.Template != "docs-bot") {
			t.Errorf("%s from docs-bot: %v; want refused %v", name, err, refused)
		}
	}
}

func TestACreateOfLessThanTwoGigabytesIsRefused(t *testing.T) {
	b := creating(t, true)
	for _, c := range []struct {
		memory int64
		ok     bool
	}{{2 << 30, true}, {2<<30 - 1, false}, {512 << 20, false}} {
		_, err := b.m.CheckCreate(t.Context(), model.CreateRequest{Image: catalog.LinuxImage, Size: model.Resources{MemoryBytes: c.memory}})
		if refused := errors.As(err, new(*model.TooLittleMemoryError)); refused == c.ok || c.ok && err != nil {
			t.Errorf("%d bytes: %v; want refused %v", c.memory, err, !c.ok)
		}
	}
}

// value: a sheet's fleet: This Mac, a laptop of 16 GB, 12 free; the mini
// with 1 GB free; the studio, silent.
func sheetFleet() model.AllMachines {
	this := model.MacView{Name: "macbook-pro", User: "you", ThisMac: true, Reachable: true, Laptop: true, Hardware: coremachine.Hardware{Memory: 16 << 30}}
	mini := model.MacView{Name: "mac-mini", Reachable: true, Hardware: coremachine.Hardware{Memory: 16 << 30}, VMs: []model.VM{
		{Name: "infra", Lifetime: model.AlwaysOn, Want: model.WantRunning, State: model.Running, Resources: model.Resources{MemoryBytes: 11 << 30}},
	}}
	studio := model.MacView{Name: "mac-studio", Hardware: coremachine.Hardware{Memory: 128 << 30}}
	return model.AllMachines{Macs: []model.MacView{this, mini, studio}}
}

type sheetMachines struct {
	machines.Machines
	builtin string
	fleet   func() model.AllMachines
}

func (s sheetMachines) AllMachinesAsItFills(_ context.Context, _ bool, sofar func(model.AllMachines)) (model.AllMachines, error) {
	f := sheetFleet()
	if s.fleet != nil {
		f = s.fleet()
	}
	sofar(model.AllMachines{Macs: f.Macs[:1]})
	return f, nil
}

func (s sheetMachines) Unasked(context.Context, string) (model.Unasked, error) {
	return model.Unasked{Request: model.CreateRequest{From: "base-ubuntu", Image: "base-ubuntu", Lifetime: model.Ephemeral, Size: model.Resources{CPUs: 4, MemoryBytes: 8 << 30}}, Kind: catalog.Linux, Builtin: s.builtin}, nil
}

func (sheetMachines) NameFor(_ catalog.MachineKind, f model.AllMachines) (string, error) {
	return "linux-" + strconv.Itoa(len(f.Macs)), nil
}

func sheet(t *testing.T, d machines.Deps, in machines.NewIn) []machines.NewMachine {
	t.Helper()
	bound, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	updates, err := machines.NewSheet.Run(bound, d, in, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got []machines.NewMachine
	for {
		select {
		case u, open := <-updates:
			if !open {
				return got
			}
			if u.Err != nil {
				t.Fatal(u.Err)
			}
			got = append(got, u.Value)
		case <-bound.Done():
			t.Fatal("the sheet never ended")
		}
	}
}

func TestTheNewMachineSheetSaysWhatEachMacTakes(t *testing.T) {
	got := sheet(t, machines.Deps{Machines: sheetMachines{builtin: "base-ubuntu"}}, machines.NewIn{On: "mac-mini"})
	if len(got) != 2 || got[0].Name != "linux-1" || len(got[0].On) != 1 {
		t.Fatalf("%d updates %+v; want This Mac's, then the fleet's, the name given over each", len(got), got)
	}
	s := got[1]
	if s.Name != "linux-3" || s.Size.MemoryBytes != 8<<30 || s.Lifetime != model.Ephemeral || s.MinMemory != 2<<30 || s.PurposeMax != model.PurposeMax || len(s.Lifetimes) != 2 {
		t.Errorf("the sheet %+v: want the file's size and lifetime, the least memory and the purpose's limit", s)
	}
	if s.Mac != "macbook-pro" {
		t.Errorf("offered on %q: the mini asked for has 1 GB free, the studio is silent; want This Mac", s.Mac)
	}
	if s.Server == nil || s.Server.Image != "base-ubuntu" || s.Server.Size != model.ServerSize() {
		t.Fatalf("server %+v: want one of base-ubuntu, a server's size", s.Server)
	}
	this, mini, studio := s.On[0], s.On[1], s.On[2]
	if !this.Ready || this.Room == nil || this.MaxMemory != 12<<30 || this.Room.RoomFor != model.RoomForGate || this.Verdict != (verdict.Verdict{Tone: verdict.Plain}) {
		t.Errorf("This Mac %+v: want ready, 12 GB at most, room for a gate", this)
	}
	if this.Server == nil || this.Server.Fit != model.FitsIn || this.Server.Verdict != (verdict.Verdict{Tone: verdict.Warn, Sentence: "This Mac may sleep: a server pauses with it."}) {
		t.Errorf("a server on This Mac %+v: want it to fit, and the laptop's warning", this.Server)
	}
	if mini.Ready || mini.Room == nil || mini.MaxMemory != 1<<30 || mini.Verdict != (verdict.Verdict{Tone: verdict.Bad, Sentence: "No room on mac-mini. Choose the other one, or delete a stopped one first."}) {
		t.Errorf("the mini %+v: want its room, not ready, and why", mini)
	}
	if mini.Server == nil || mini.Server.Fit != model.NoRoom || mini.Server.Verdict.Tone != verdict.Plain {
		t.Errorf("a server on the mini %+v: want no room, and no warning", mini.Server)
	}
	if studio.Ready || studio.Room != nil || studio.Server != nil || studio.Verdict != (verdict.Verdict{Tone: verdict.Plain, Sentence: "mac-studio does not answer"}) {
		t.Errorf("the silent studio %+v: want no room, not ready, and why", studio)
	}
}

func TestAUsersFileOffersNoServer(t *testing.T) {
	got := sheet(t, machines.Deps{Machines: sheetMachines{}}, machines.NewIn{From: "docs-bot"})
	last := got[len(got)-1]
	if last.Server != nil {
		t.Errorf("server %+v from the owner's file: want none", last.Server)
	}
	for _, on := range last.On {
		if on.Server != nil {
			t.Errorf("%s offers a server %+v from the owner's file: want none", on.Mac, on.Server)
		}
	}
}

func TestTheSheetOpensOnTheMacAskedForThoughAnotherIsRoomier(t *testing.T) {
	roomy := func() model.AllMachines {
		f := sheetFleet()
		f.Macs[1].VMs, f.Macs[1].Hardware.Memory = nil, 64<<30
		return f
	}
	got := sheet(t, machines.Deps{Machines: sheetMachines{fleet: roomy}}, machines.NewIn{On: "macbook-pro"})
	if last := got[len(got)-1]; last.Mac != "macbook-pro" {
		t.Errorf("offered on %q: This Mac was asked for and takes a VM, though the mini has more free; want This Mac", last.Mac)
	}
}
