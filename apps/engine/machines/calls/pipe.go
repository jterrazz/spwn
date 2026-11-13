package calls

import (
	"context"
	"errors"
	"strconv"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

const pipeIPWait = 5 * time.Second

func (s *Machinist) CheckPipe(ctx context.Context, name string) (model.VMRecord, error) {
	if err := s.CheckMac(); err != nil {
		return model.VMRecord{}, err
	}
	m, err := s.lookup(name)
	var unknown *model.UnknownMachineError
	if errors.As(err, &unknown) {
		if mac, ok := s.macOf(ctx, name); ok {
			return model.VMRecord{}, &model.NotThisMacError{Name: name, Mac: mac}
		}
	}
	if err != nil {
		return model.VMRecord{}, err
	}
	return s.VM(m.Name, "pipe")
}

func (s *Machinist) macOf(ctx context.Context, name string) (string, bool) {
	allMachines, err := s.AllMachines(ctx, false)
	if err != nil {
		return "", false
	}
	for _, h := range allMachines.Macs {
		for _, vm := range h.VMs {
			if vm.Name == name {
				return h.Name, true
			}
		}
	}
	return "", false
}

func (s *Machinist) Pipe(ctx context.Context, name string, pipe func(ctx context.Context, ip string, port int) error) error {
	// invariant: the VM's NAT ip is read at each pipe, never written into a block:
	// the hypervisor's DHCP lease may change across runs; the pipe needs no daemon.
	rec, err := s.CheckPipe(ctx, name)
	if err != nil {
		return err
	}
	if s.engineState(ctx, name) != contract.DiskRunning {
		return &model.StoppedError{Name: name, Lifetime: rec.Lifetime}
	}
	addr, err := s.engine.Address(ctx, name, pipeIPWait)
	if err != nil {
		return &model.SSHDClosedError{Name: name, Addr: name, Cause: err}
	}
	ip := addr.String()
	port := s.o.GuestPort
	if port == 0 {
		port = 22
	}
	if err := pipe(ctx, ip, port); err != nil {
		return &model.SSHDClosedError{Name: name, Addr: ip + ":" + strconv.Itoa(port), Cause: err}
	}
	return nil
}
