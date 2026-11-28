package calls

import (
	"context"
	"errors"

	model "github.com/jterrazz/studio-station/apps/engine/machines"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

func (s *Machinist) SSHBlocks(ctx context.Context, f model.AllMachines) error {
	// hazard: the blocks of machines the fleet no longer holds are pruned only when
	// every host answered; an unreachable host's VMs are unknown, not gone. A
	// VM user the rule refuses writes no block at all, so the file keeps the
	// last blocks it was right about.
	cfg, err := s.prefs()
	if err != nil {
		return err
	}
	var blocks []contract.MachineBlock
	complete := true
	key := cfg.Macs[cfg.ThisMachine].Key
	if why := contract.UnwritableKey(cfg.ThisMachine, key); why != "" {
		return &model.BadSSHTargetError{Why: why}
	}
	for _, mac := range f.Macs {
		complete = complete && mac.Reachable
		if !mac.ThisMac {
			h := cfg.Macs[mac.Name]
			blocks = append(blocks, contract.MachineBlock{Name: mac.Name, User: h.User, Key: h.Key})
		}
		for _, vm := range mac.VMs {
			user, bad := s.guestUser(cfg, mac.Name)
			if bad != nil {
				return bad
			}
			// invariant: every VM has the block of its full address; the bare
			// name's block is This Mac's own VM of that name, else the one VM
			// of All machines that bears it, and none when two other Macs do.
			address := vm.Name + "." + mac.Name
			blocks = append(blocks, contract.MachineBlock{Name: address, User: user, Key: key, Behind: mac.Name, ThisMac: mac.ThisMac, VM: vm.Name})
			if mac.ThisMac || model.Address(f, mac.Name, vm.Name) == vm.Name {
				blocks = append(blocks, contract.MachineBlock{Name: vm.Name, User: user, Key: key, Behind: mac.Name, ThisMac: mac.ThisMac})
			}
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
