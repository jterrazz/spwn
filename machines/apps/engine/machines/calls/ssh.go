package calls

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

func (s *Machinist) SSH(ctx context.Context, name string) (model.SSHTarget, error) {
	// value: the ssh command that reaches a machine of All machines, as the
	// owner's ssh blocks write it: a Mac as its own login, a VM, here or on
	// another Mac, as the VM user.
	if err := s.CheckMac(); err != nil {
		return model.SSHTarget{}, err
	}
	cfg, err := s.prefs()
	if err != nil {
		return model.SSHTarget{}, err
	}
	m, err := s.lookup(name)
	var unknown *model.UnknownMachineError
	if errors.As(err, &unknown) {
		if mac, ok := s.macOf(ctx, name); ok {
			user, bad := s.guestUser(cfg, mac)
			if bad != nil {
				return model.SSHTarget{}, bad
			}
			return sshTarget(user, name)
		}
		if bad := s.heldByARefusedMac(cfg, name); bad != nil {
			return model.SSHTarget{}, bad
		}
	}
	if err != nil {
		return model.SSHTarget{}, err
	}
	if m.Kind == catalog.Mac {
		return sshTarget(m.User, m.Name)
	}
	user, bad := s.vmUser(cfg)
	if bad != nil {
		return model.SSHTarget{}, bad
	}
	return sshTarget(user, m.Name)
}

// invariant: a target's user and name are each one word ssh reads as no
// option, by the rule its ssh block is written by, whatever preferences.json
// holds by hand.
func sshTarget(user, name string) (model.SSHTarget, error) {
	if why := contract.Unwritable(contract.MachineBlock{Name: name, User: user}); why != "" {
		return model.SSHTarget{}, &model.BadSSHTargetError{Why: why}
	}
	command, err := catalog.SSHCommand(user, name)
	if err != nil {
		return model.SSHTarget{}, &model.BadSSHTargetError{Why: err.Error()}
	}
	return model.SSHTarget{User: user, Address: name, Command: command}, nil
}

func (s *Machinist) heldByARefusedMac(cfg prefsView, vm string) error {
	_, refused := s.otherMacs()
	for _, r := range refused {
		if !slices.ContainsFunc(cfg.Macs[r.Name].VMs, func(v model.SeenVM) bool { return v.Name == vm }) {
			continue
		}
		why := contract.Unwritable(contract.MachineBlock{Name: vm, User: cfg.Macs[r.Name].User})
		return &model.BadSSHTargetError{Why: cmp.Or(why, r.Why)}
	}
	return nil
}
