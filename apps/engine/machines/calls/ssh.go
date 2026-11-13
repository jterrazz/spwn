package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
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
		if _, ok := s.macOf(ctx, name); ok {
			return sshTarget(s.vmUser(cfg), name), nil
		}
	}
	if err != nil {
		return model.SSHTarget{}, err
	}
	return sshTarget(m.User, m.Name), nil
}

func sshTarget(user, mac string) model.SSHTarget {
	return model.SSHTarget{User: user, Address: mac, Command: catalog.SSHCommand(user, mac)}
}
