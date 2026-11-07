package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/src/core/store"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/machines/contract"

	"github.com/jterrazz/studio-station/src/catalog"
)

func recordOf(r model.Record) model.VMRecord {
	return model.VMRecord{
		Name: r.Name, Image: r.Image, Version: r.Version, Lifetime: r.Lifetime, Want: r.Want, Network: r.Network, Check: r.Check, Env: r.Env,
		Resources: model.Resources{CPUs: r.CPUs, MemoryBytes: r.MemoryBytes, DiskBytes: r.DiskBytes},
		CreatedAt: r.CreatedAt,
	}
}

func storedOf(r model.VMRecord) model.Record {
	return model.Record{
		Name: r.Name, Image: r.Image, Version: r.Version, Lifetime: r.Lifetime, Want: r.Want, Network: r.Network, Check: r.Check, Env: r.Env,
		CPUs: r.Resources.CPUs, MemoryBytes: r.Resources.MemoryBytes, DiskBytes: r.Resources.DiskBytes, CreatedAt: r.CreatedAt,
	}
}

func (s *Machinist) records() ([]model.VMRecord, error) {
	stored, err := s.recs.List(context.Background())
	if err != nil {
		var unreadable *store.UnreadableError
		if errors.As(err, &unreadable) {
			return nil, &model.RecordError{Path: unreadable.Path, Cause: unreadable.Cause}
		}
		return nil, err
	}
	out := make([]model.VMRecord, 0, len(stored))
	for _, r := range stored {
		out = append(out, recordOf(r))
	}
	return out, nil
}

func kindOfImage(image string) catalog.MachineKind {
	for _, i := range catalog.Images {
		if i.Name == image {
			return i.Kind
		}
	}
	return catalog.Linux
}

func (s *Machinist) Records() ([]model.VMRecord, error) { return s.records() }

func (s *Machinist) vms(ctx context.Context, mac string, live map[string]EngineVM) ([]model.VM, error) {
	// invariant: the daemon's view wins; without it, no engine or a listing that
	// fails means nothing is running: every workspace reads stopped.
	records, err := s.records()
	if err != nil || len(records) == 0 {
		return nil, err
	}
	engine := map[string]contract.Disk{}
	if s.engine != nil {
		if disks, listErr := s.engine.Disks(ctx); listErr == nil {
			for _, d := range disks {
				engine[d.VM] = d
			}
		}
	}
	out := make([]model.VM, 0, len(records))
	for _, r := range records {
		info := engine[r.Name]
		vm := model.VM{
			Name: r.Name, Image: r.Image, Version: r.Version, Mac: mac, Kind: kindOfImage(r.Image),
			Lifetime: r.Lifetime, State: stateOf(info), Resources: r.Resources, SizeGB: int(info.Bytes / 1e9),
			Network: networkOf(r), Guest: "unknown", Want: r.Want, Check: r.Check, CreatedAt: r.CreatedAt,
		}
		if d, ok := live[r.Name]; ok && live != nil {
			if d.State != "" {
				vm.State = model.VMState(d.State)
			}
			vm.Guest, vm.IP, vm.Detail = d.Guest, d.IP, d.Detail
			vm.SSH, vm.Restarts, vm.CheckStatus, vm.BootAt = d.SSH, d.Restarts, d.Check, d.BootAt
			vm.Latency, vm.Restart = d.Latency, d.Restart
		}
		if r.Lifetime == model.AlwaysOn && s.o.Uptime != nil {
			vm.Uptime, _ = s.o.Uptime.Get(ctx, r.Name)
		}
		out = append(out, vm)
	}
	return out, nil
}

func networkOf(r model.VMRecord) string {
	if r.Network == "" {
		return model.NetworkNAT
	}
	return r.Network
}

func stateOf(d contract.Disk) model.VMState {
	switch d.State {
	case contract.DiskRunning:
		return model.Running
	case contract.DiskSuspended:
		return model.Suspended
	case contract.DiskStopped, diskMissing:
	}
	return model.Stopped
}
