package calls

import (
	"context"
	"errors"

	model "github.com/jterrazz/studio-station/src/machines"
)

var errNoWaker = errors.New("no Wake-on-LAN on this build")

func (s *Machinist) Wake(ctx context.Context, name string, tell bool) error {
	cfg, err := s.prefs()
	if err != nil {
		return err
	}
	known := cfg.Macs[name]
	if known.Hardware == nil || known.Hardware.Ether == "" {
		return &model.NoHardwareAddressError{Name: name}
	}
	if s.o.Waker == nil {
		return errNoWaker
	}
	if err := s.o.Waker.Wake(ctx, known.Hardware.Ether); err != nil {
		return err
	}
	if !tell || known.TellBack {
		return nil
	}
	known.TellBack = true
	cfg.Macs[name] = known
	return s.p.Macs.Set(cfg.Macs)
}

func (s *Machinist) TellBack() map[string]bool {
	// value: the Macs the owner asked to be told of when they answer again.
	cfg, err := s.prefs()
	if err != nil {
		return nil
	}
	out := map[string]bool{}
	for name, r := range cfg.Macs {
		if r.TellBack {
			out[name] = true
		}
	}
	return out
}

func (s *Machinist) ToldBack(name string) error {
	cfg, err := s.prefs()
	if err != nil {
		return err
	}
	known, ok := cfg.Macs[name]
	if !ok || !known.TellBack {
		return nil
	}
	known.TellBack = false
	cfg.Macs[name] = known
	return s.p.Macs.Set(cfg.Macs)
}
