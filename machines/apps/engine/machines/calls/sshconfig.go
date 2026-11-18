package calls

import (
	"context"
	"errors"

	model "github.com/jterrazz/studio-station/apps/engine/machines"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

func (s *Machinist) SSHBlocks(ctx context.Context, f model.AllMachines) error {
	// hazard: the blocks of machines the fleet no longer holds are pruned only when
	// every host answered; an unreachable host's VMs are unknown, not gone.
	cfg, err := s.prefs()
	if err != nil {
		return err
	}
	var blocks []contract.MachineBlock
	complete := true
	key := cfg.Macs[cfg.ThisMachine].Key
	for _, mac := range f.Macs {
		complete = complete && mac.Reachable
		if !mac.ThisMac {
			h := cfg.Macs[mac.Name]
			blocks = append(blocks, contract.MachineBlock{Name: mac.Name, User: h.User, Key: h.Key})
		}
		for _, vm := range mac.VMs {
			blocks = append(blocks, contract.MachineBlock{Name: vm.Name, User: s.vmUser(cfg), Key: key, Behind: mac.Name, ThisMac: mac.ThisMac})
		}
	}
	write := s.o.SSHConfig.Put
	if complete {
		write = s.o.SSHConfig.Replace
	}
	return blocksFailure(write(ctx, blocks), false)
}

func blocksFailure(err error, rolledBack bool) error {
	if err == nil {
		return nil
	}
	var refused *contract.SSHConfigError
	if errors.As(err, &refused) {
		return &model.SSHConfigError{Path: refused.Path, Cause: refused.Cause, RolledBack: rolledBack}
	}
	return &model.SSHConfigError{Cause: err, RolledBack: rolledBack}
}
