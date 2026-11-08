package calls

import (
	"context"
	"slices"

	model "github.com/jterrazz/studio-station/src/machines"

	"github.com/jterrazz/studio-station/src/catalog"
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
	vms, err := s.vms(ctx, name, live)
	if err != nil {
		return model.AllMachines{}, err
	}
	for i, vm := range vms {
		if _, own := nodes[vm.Name]; own && vm.Lifetime == model.AlwaysOn {
			vms[i].Device = vm.Name
		}
	}
	user := cfg.Macs[cfg.ThisMachine].User
	if user == "" {
		user = s.h.User()
	}
	thisMac := model.MacView{
		Name: name, User: user, StationVersion: s.o.Version, ThisMac: true, Reachable: true, Node: nodes[name], Engine: info, VMs: vms,
	}
	if s.o.ThisMachine == catalog.Mac && s.o.LAN != nil {
		thisMac.LAN = s.o.LAN(ctx)
	}
	if thisMacOnly || s.o.ThisMachine != catalog.Mac {
		return model.AllMachines{Macs: []model.MacView{thisMac}}, nil
	}
	others := s.OtherMacs()
	answers := map[string]model.MacView{}
	allMachines := func(all bool) model.AllMachines {
		f := model.AllMachines{Macs: []model.MacView{thisMac}, Read: s.h.Now()}
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
	version, err := s.engineAPI.Ping(ctx)
	if err != nil {
		installed := s.h.Exists(catalog.EnginePlist(s.h.Home()))
		return model.EngineView{Known: true, Installed: installed}, nil
	}
	live, err = s.engineAPI.VMs(ctx)
	if err != nil {
		return model.EngineView{Known: true, Running: true, Version: version}, nil
	}
	return model.EngineView{Known: true, Running: true, Version: version}, live
}

func (s *Machinist) view(name string, h model.MacRecord, node model.Node, answers map[string]model.MacView) model.MacView {
	far, ok := answers[name]
	if !ok {
		silent := model.MacView{Name: name, User: h.User, Node: node, Answered: h.Answered, LAN: h.LAN}
		for _, vm := range h.VMs {
			silent.VMs = append(silent.VMs, model.VM{
				Name: vm.Name, Mac: name, Lifetime: vm.Lifetime, State: model.UnknownState,
				Resources: model.Resources{CPUs: vm.CPUs, MemoryBytes: vm.MemoryBytes}, Agents: vm.Agents,
			})
		}
		return silent
	}
	view := model.MacView{Name: name, User: h.User, StationVersion: far.StationVersion, Reachable: true, LAN: far.LAN, Node: far.Node, Engine: far.Engine}
	for _, vm := range far.VMs {
		vm.Mac = name
		view.VMs = append(view.VMs, vm)
	}
	if node.IP != "" || node.Online {
		view.Node = node
	}
	return view
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
