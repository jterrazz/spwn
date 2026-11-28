package calls

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

const pipeIPWait = 5 * time.Second

func (s *Machinist) CheckPipe(ctx context.Context, name string) (model.VMRecord, error) {
	if err := s.CheckMac(); err != nil {
		return model.VMRecord{}, err
	}
	address := name
	if ref, bad := machine.ParseAddress(name); bad == nil && ref.On == s.ThisMacName() {
		name = ref.Name
	}
	m, err := s.lookup(name)
	var unknown *model.UnknownMachineError
	if errors.As(err, &unknown) {
		mac, held := s.macOf(ctx, address)
		if held != nil {
			return model.VMRecord{}, held
		}
		if mac != "" {
			return model.VMRecord{}, &model.NotThisMacError{Name: name, Mac: mac}
		}
	}
	if err != nil {
		return model.VMRecord{}, err
	}
	return s.VM(m.Name, "pipe")
}

// value: the other Mac that holds a VM by its address: the Mac a full
// address names, when it lists that VM; else the one other Mac whose VM
// bears the bare name, two or more refused naming each; none, an empty
// name, when All machines cannot be read.
func (s *Machinist) macOf(ctx context.Context, name string) (string, error) {
	ref, err := machine.ParseAddress(name)
	if err != nil {
		return "", err
	}
	var macs []model.MacView
	if allMachines, listed := s.AllMachines(ctx, false); listed == nil {
		macs = allMachines.Macs
	}
	var holders, silent []string
	for _, h := range macs {
		if h.ThisMac || ref.On != "" && h.Name != ref.On {
			continue
		}
		if slices.ContainsFunc(h.VMs, func(vm model.VM) bool { return vm.Name == ref.Name }) {
			holders = append(holders, h.Name)
			if !h.Reachable {
				silent = append(silent, h.Name)
			}
		}
	}
	switch len(holders) {
	case 0:
		return "", nil
	case 1:
		return holders[0], nil
	}
	return "", call.Ambiguous(ref.Name, holders, silent)
}

func (s *Machinist) Pipe(ctx context.Context, name string, pipe func(ctx context.Context, ip string, port int) error) error {
	// invariant: the VM's NAT ip is read at each pipe, never written into a block:
	// the hypervisor's DHCP lease may change across runs; the pipe needs no daemon.
	rec, err := s.CheckPipe(ctx, name)
	if err != nil {
		return err
	}
	state, err := s.hypervisorState(ctx, name)
	if err != nil {
		return err
	}
	if state != contract.DiskRunning {
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
