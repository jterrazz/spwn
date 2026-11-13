package calls

import (
	"context"
	"errors"
	"maps"
	"sort"
	"strings"
	"time"

	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

func (s *Machinist) Lookup(_ context.Context, name string) (Machine, error) { return s.lookup(name) }

func (s *Machinist) lookup(name string) (Machine, error) {
	cfg, err := s.prefs()
	if err != nil {
		return Machine{}, err
	}
	if h, ok := cfg.Macs[name]; ok {
		return Machine{Name: name, Kind: catalog.Mac, ThisMachine: name == cfg.ThisMachine, User: h.User, Key: h.Key, LAN: h.LAN}, nil
	}
	records, err := s.records()
	if err != nil {
		return Machine{}, err
	}
	for _, r := range records {
		if r.Name == name {
			return Machine{Name: name, Kind: s.kindOfImage(r.Image), Mac: cfg.ThisMachine, User: s.vmUser(cfg), Lifetime: r.Lifetime}, nil
		}
	}
	// value: a Mac answers to its short name too, the one All machines shows
	// (mac-mini for jterrazz-mac-mini).
	if full := s.o.User + "-" + name; s.o.User != "" {
		if h, ok := cfg.Macs[full]; ok {
			return Machine{Name: full, Kind: catalog.Mac, ThisMachine: full == cfg.ThisMachine, User: h.User, Key: h.Key, LAN: h.LAN}, nil
		}
	}
	return Machine{}, &model.UnknownMachineError{Name: name}
}

func (s *Machinist) Remote(name string) (Machine, error) {
	// invariant: another machine is never This Mac, by any of its names,
	// registered or not.
	if cfg, err := s.prefs(); err == nil {
		if thisMac := s.thisMac(cfg); name == thisMac || (s.o.User != "" && s.o.User+"-"+name == thisMac) {
			return Machine{}, &model.ThisMachineError{Name: name}
		}
	}
	m, err := s.lookup(name)
	if err != nil {
		return Machine{}, err
	}
	if m.ThisMachine {
		return Machine{}, &model.ThisMachineError{Name: name}
	}
	return m, nil
}

func (s *Machinist) RemoteMac(name, verb string) (Machine, error) {
	m, err := s.Remote(name)
	var thisMac *model.ThisMachineError
	if errors.As(err, &thisMac) {
		thisMac.Verb = verb
	}
	if err != nil {
		return Machine{}, err
	}
	if m.Kind != catalog.Mac {
		return Machine{}, &model.NotAMacError{Name: name, Verb: verb}
	}
	return m, nil
}

func (s *Machinist) Init(name string) error {
	if !validName(name) {
		return &model.BadNameError{Name: name}
	}
	cfg, err := s.prefs()
	if err != nil {
		return err
	}
	if _, ok := cfg.Macs[name]; !ok {
		macs := maps.Clone(cfg.Macs)
		if macs == nil {
			macs = map[string]model.MacRecord{}
		}
		macs[name] = model.MacRecord{}
		if err := s.p.Macs.Set(macs); err != nil {
			return err
		}
	}
	if cfg.VMUser == "" {
		return s.p.VMUser.Set(s.o.User)
	}
	return nil
}

func (s *Machinist) AddMac(ctx context.Context, name string, h model.MacRecord) error {
	if err := s.CheckMac(); err != nil {
		return err
	}
	if !validName(name) {
		return &model.BadNameError{Name: name}
	}
	cfg, err := s.prefs()
	if err != nil {
		return err
	}
	if _, ok := cfg.Macs[name]; ok {
		return &model.NameTakenError{Name: name}
	}
	records, err := s.records()
	if err != nil {
		return err
	}
	for _, r := range records {
		if r.Name == name {
			return &model.NameTakenError{Name: name, VM: true}
		}
	}
	before := cfg.Macs
	macs := maps.Clone(cfg.Macs)
	if macs == nil {
		macs = map[string]model.MacRecord{}
	}
	macs[name] = h
	if err := s.p.Macs.Set(macs); err != nil {
		return err
	}
	if err := s.o.SSHConfig.Put(ctx, []contract.MachineBlock{{Name: name, User: h.User, Key: h.Key}}); err != nil {
		_ = s.p.Macs.Set(before)
		return blocksFailure(err, true)
	}
	return nil
}

func (s *Machinist) ForgetMac(ctx context.Context, name string) error {
	if err := s.CheckMac(); err != nil {
		return err
	}
	m, err := s.RemoteMac(name, "forget")
	if err != nil {
		return err
	}
	cfg, err := s.prefs()
	if err != nil {
		return err
	}
	delete(cfg.Macs, m.Name)
	if err := s.p.Macs.Set(cfg.Macs); err != nil {
		return err
	}
	return blocksFailure(s.o.SSHConfig.Remove(ctx, m.Name), false)
}

func (s *Machinist) thisMac(cfg prefsView) string {
	if cfg.ThisMachine != "" {
		return cfg.ThisMachine
	}
	return strings.ToLower(s.h.Hostname())
}

func (s *Machinist) OtherMacs() []string {
	// value: every host of the fleet but this one, by name.
	cfg, err := s.prefs()
	if err != nil {
		return nil
	}
	thisMac := s.thisMac(cfg)
	others := make([]string, 0, len(cfg.Macs))
	for n := range cfg.Macs {
		if n != thisMac && n != cfg.ThisMachine {
			others = append(others, n)
		}
	}
	sort.Strings(others)
	return others
}

func (s *Machinist) Seen(name string) (model.MacRecord, bool) {
	// value: another Mac as this one last had its answer, kept on its record.
	cfg, err := s.prefs()
	if err != nil {
		return model.MacRecord{}, false
	}
	h, ok := cfg.Macs[name]
	return h, ok
}

func (s *Machinist) Names() []string {
	cfg, err := s.prefs()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(cfg.Macs))
	for n := range cfg.Macs {
		names = append(names, n)
	}
	if records, err := s.records(); err == nil {
		for _, r := range records {
			names = append(names, r.Name)
		}
	}
	sort.Strings(names)
	return names
}

func (s *Machinist) Remember(_ context.Context, f model.AllMachines) error {
	cfg, err := s.prefs()
	if err != nil {
		return err
	}
	changed := false
	for _, h := range f.Macs {
		known, ok := cfg.Macs[h.Name]
		if h.ThisMac || !h.Reachable || !ok {
			continue
		}
		seen := make([]model.SeenVM, 0, len(h.VMs))
		for _, vm := range model.Ordered(h.VMs) {
			seen = append(seen, model.SeenVM{Name: vm.Name, Kind: vm.Kind, Lifetime: vm.Lifetime, CPUs: vm.Resources.CPUs, MemoryBytes: vm.Resources.MemoryBytes, Agents: vm.Agents})
		}
		known.Answered, known.VMs = s.h.Now().UTC().Truncate(time.Second), seen
		known.Laptop, known.Asleep, known.Hardware = h.Laptop, h.Asleep, nil
		if h.Hardware != (coremachine.Hardware{}) {
			hw := h.Hardware
			known.Hardware = &hw
		}
		if h.LAN != "" {
			known.LAN = h.LAN
		}
		cfg.Macs[h.Name], changed = known, true
	}
	if !changed {
		return nil
	}
	return s.p.Macs.Set(cfg.Macs)
}
