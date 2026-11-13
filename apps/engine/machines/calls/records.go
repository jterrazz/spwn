package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func recordOf(r model.Record) model.VMRecord {
	return model.VMRecord{
		Name: r.Name, Image: r.Image, Version: r.Version, Lifetime: r.Lifetime, Want: r.Want, Network: r.Network, Check: r.Check, Env: r.Env,
		Resources: model.Resources{CPUs: r.CPUs, MemoryBytes: r.MemoryBytes, DiskBytes: r.DiskBytes},
		CreatedAt: r.CreatedAt, Birth: r.Birth, Owner: r.Owner, Agents: r.Agents, Template: r.Template, Mounts: r.Mounts,
		Purpose: r.Purpose, Creator: r.Creator,
	}
}

func storedOf(r model.VMRecord) model.Record {
	return model.Record{
		Name: r.Name, Image: r.Image, Version: r.Version, Lifetime: r.Lifetime, Want: r.Want, Network: r.Network, Check: r.Check, Env: r.Env,
		CPUs: r.Resources.CPUs, MemoryBytes: r.Resources.MemoryBytes, DiskBytes: r.Resources.DiskBytes, CreatedAt: r.CreatedAt,
		Birth: r.Birth, Owner: r.Owner, Agents: r.Agents, Template: r.Template, Mounts: r.Mounts,
		Purpose: r.Purpose, Creator: r.Creator,
	}
}

func (s *Machinist) stored() ([]model.Record, error) {
	stored, err := s.recs.List(context.Background())
	if err != nil {
		var unreadable *store.UnreadableError
		if errors.As(err, &unreadable) {
			return nil, &model.RecordError{Path: unreadable.Path, Cause: unreadable.Cause}
		}
		return nil, err
	}
	return stored, nil
}

func (s *Machinist) records() ([]model.VMRecord, error) {
	stored, err := s.stored()
	if err != nil {
		return nil, err
	}
	out := make([]model.VMRecord, 0, len(stored))
	for _, r := range stored {
		out = append(out, recordOf(r))
	}
	return out, nil
}

func (s *Machinist) kindOfImage(image string) catalog.MachineKind {
	for _, i := range s.o.Images {
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
	stored, err := s.stored()
	if err != nil || len(stored) == 0 {
		return nil, err
	}
	out := s.vmsOf(mac, stored, s.disks(ctx), live)
	for i, vm := range out {
		if vm.Lifetime == model.AlwaysOn && s.o.Uptime != nil {
			out[i].Uptime, _ = s.o.Uptime.Get(ctx, vm.Name)
		}
	}
	return out, nil
}

// value: the disks this Mac's hypervisor holds, by VM; none when it does
// not say.
func (s *Machinist) disks(ctx context.Context) map[string]contract.Disk {
	out := map[string]contract.Disk{}
	if s.engine == nil {
		return out
	}
	if disks, err := s.engine.Disks(ctx); err == nil {
		for _, d := range disks {
			out[d.VM] = d
		}
	}
	return out
}

// value: the VMs these records are, as the disks and the engine's view
// say, each birth judged by whether its create's process runs.
func (s *Machinist) vmsOf(mac string, stored []model.Record, disks map[string]contract.Disk, live map[string]EngineVM) []model.VM {
	out := make([]model.VM, 0, len(stored))
	for _, rec := range stored {
		r := recordOf(rec)
		info := disks[r.Name]
		vm := model.VM{
			Name: r.Name, Image: r.Image, Version: r.Version, Mac: mac, Kind: s.kindOfImage(r.Image),
			Lifetime: r.Lifetime, State: stateOf(info), Resources: r.Resources, SizeGB: int(info.Bytes / 1e9),
			Network: networkOf(r), Guest: "unknown", Want: r.Want, Check: model.ShownCheck(r.Check), CreatedAt: r.CreatedAt, Agents: r.Agents,
			Purpose: r.Purpose, Creator: model.CreatorOf(r.Creator),
		}
		if d, ok := live[r.Name]; ok && live != nil {
			if d.State != "" {
				vm.State = model.VMState(d.State)
			}
			vm.Guest, vm.IP, vm.Detail = d.Guest, d.IP, d.Detail
			vm.SSH, vm.Restarts, vm.CheckStatus, vm.BootAt = d.SSH, d.Restarts, d.Check, d.BootAt
			vm.Latency, vm.Restart = d.Latency, d.Restart
			vm.Cause, vm.LastLines, vm.ReadyAt, vm.Agent = d.Cause, d.LastLines, d.ReadyAt, d.Agent
		}
		out = append(out, model.Born(vm, r.Birth, s.runs(r.Owner)))
	}
	return out
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
