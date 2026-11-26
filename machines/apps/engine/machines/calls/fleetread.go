package calls

import (
	"context"
	"slices"

	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func (s *Machinist) AllMachines(ctx context.Context, thisMacOnly bool) (model.AllMachines, error) {
	return s.AllMachinesAsItFills(ctx, thisMacOnly, nil)
}

func (s *Machinist) AllMachinesAsItFills(ctx context.Context, thisMacOnly bool, sofar func(model.AllMachines)) (model.AllMachines, error) {
	// invariant: sofar sees this host first, then this host and every other
	// Mac that has answered, in name order, after each answer; the fleet
	// returned holds every Mac, a silent one as unreachable.
	cfg, err := s.prefs()
	if err != nil {
		return model.AllMachines{}, err
	}
	nodes := s.nodeMap(ctx)
	name := s.thisMac(cfg)
	info, live := s.engineView(ctx)
	vms, silent, err := s.listed(ctx, name, live)
	if err != nil {
		return model.AllMachines{}, err
	}
	var uses map[string]int64
	if s.o.Uses != nil {
		uses = s.o.Uses(ctx)
	}
	for i, vm := range vms {
		if _, own := nodes[vm.Name]; own && vm.Lifetime == model.AlwaysOn {
			vms[i].Device = vm.Name
		}
		if vm.State == model.Running {
			vms[i].UsedBytes = uses[vm.Name]
		}
	}
	user := cfg.Macs[cfg.ThisMachine].User
	if user == "" {
		user = s.h.User()
	}
	thisMac := model.MacView{
		Name: name, User: user, StationVersion: s.o.Version, ThisMac: true, Reachable: true, Node: nodes[name], Engine: info, VMs: vms,
	}
	if silent && s.o.ThisMachine == catalog.Mac {
		thisMac.SilentHypervisor = s.engine.Name()
		thisMac.SilentPackage = s.engine.Package()
	}
	if s.o.ThisMachine == catalog.Mac && s.o.LAN != nil {
		thisMac.LAN = s.o.LAN(ctx)
	}
	if s.o.ThisMachine == catalog.Mac {
		thisMac.Hardware, thisMac.Laptop = s.hardware(ctx), s.o.Laptop
		thisMac.Asleep = s.o.Asleep != nil && s.o.Asleep(ctx)
		if s.o.Awake != nil {
			thisMac.Awake = s.o.Awake(ctx)
		}
	}
	if thisMacOnly || s.o.ThisMachine != catalog.Mac {
		return model.AllMachines{Macs: []model.MacView{thisMac}, Read: s.h.Now()}, nil
	}
	others, refused := s.otherMacs()
	answers := map[string]model.MacView{}
	allMachines := func(all bool) model.AllMachines {
		f := model.AllMachines{Macs: []model.MacView{thisMac}, Read: s.h.Now(), Refused: refused}
		for _, n := range others {
			if _, ok := answers[n]; ok || all {
				f.Macs = append(f.Macs, s.view(n, cfg.Macs[n], nodes[n], answers))
			}
		}
		return f
	}
	if sofar != nil {
		sofar(allMachines(false))
	}
	if s.o.Ask != nil && len(others) > 0 {
		s.o.Ask(ctx, others, func(mac string, view model.MacView) {
			answers[mac] = view
			if sofar != nil {
				sofar(allMachines(false))
			}
		})
	}
	return allMachines(true), nil
}

func (s *Machinist) hardware(ctx context.Context) coremachine.Hardware {
	if s.o.Hardware == nil {
		return coremachine.Hardware{}
	}
	return s.o.Hardware(ctx)
}

func (s *Machinist) nodeMap(ctx context.Context) map[string]model.Node {
	if s.nodes == nil || s.o.ThisMachine != catalog.Mac {
		return nil
	}
	nodes, err := s.nodes.Peers(ctx)
	if err != nil {
		return nil
	}
	return nodes
}

// invariant: the daemon matters only where VMs can exist: on a host with its hypervisor; a
// daemon that does not answer degrades the view, never fails it.
func (s *Machinist) engineView(ctx context.Context) (info model.EngineView, live map[string]EngineVM) {
	if s.o.ThisMachine != catalog.Mac || s.engineAPI == nil {
		return model.EngineView{}, nil
	}
	if !s.engine.Installed() {
		return model.EngineView{}, nil
	}
	info, err := s.engineAPI.Hello(ctx)
	if err != nil {
		if !s.engineInstalled() {
			return model.EngineView{State: model.EngineNotInstalled}, nil
		}
		return model.EngineView{State: model.EngineStopped}, nil
	}
	live, err = s.engineAPI.VMs(ctx)
	if err != nil {
		return info, nil
	}
	return info, live
}

func (s *Machinist) view(name string, h model.MacRecord, node model.Node, answers map[string]model.MacView) model.MacView {
	far, ok := answers[name]
	return model.ViewOf(name, h, node, far, ok)
}

func (s *Machinist) Known(ctx context.Context) (records map[string]model.MacRecord, nodes map[string]model.Node) {
	// value: every other Mac as This Mac's records keep it, and the
	// tailnet's node of each Mac it sees, read now.
	records = map[string]model.MacRecord{}
	if cfg, err := s.prefs(); err == nil {
		for _, n := range s.OtherMacs() {
			records[n] = cfg.Macs[n]
		}
	}
	return records, s.nodeMap(ctx)
}

func (s *Machinist) Running(ctx context.Context) (map[string]catalog.MachineKind, error) {
	_, live := s.engineView(ctx)
	vms, err := s.vms(ctx, "", live)
	if err != nil {
		return nil, err
	}
	running := map[string]catalog.MachineKind{}
	for _, vm := range vms {
		if vm.State == model.Running {
			running[vm.Name] = vm.Kind
		}
	}
	return running, nil
}

func (s *Machinist) Remembers(mac, vm string) bool {
	cfg, err := s.prefs()
	if err != nil {
		return false
	}
	return slices.ContainsFunc(cfg.Macs[mac].VMs, func(v model.SeenVM) bool { return v.Name == vm })
}
