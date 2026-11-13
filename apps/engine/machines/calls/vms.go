package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
)

func (s *Machinist) VM(name, verb string) (model.VMRecord, error) {
	if err := s.CheckMac(); err != nil {
		return model.VMRecord{}, err
	}
	m, err := s.lookup(name)
	if err != nil {
		return model.VMRecord{}, err
	}
	if m.Kind == catalog.Mac {
		return model.VMRecord{}, &model.NotAVMError{Name: name, Verb: verb}
	}
	records, err := s.records()
	if err != nil {
		return model.VMRecord{}, err
	}
	for _, r := range records {
		if r.Name == name {
			return r, nil
		}
	}
	return model.VMRecord{}, &model.UnknownMachineError{Name: name}
}

func (s *Machinist) Log(ctx context.Context, name string) ([]string, error) {
	if _, err := s.VM(name, "read the log of"); err != nil {
		return nil, err
	}
	lines, err := s.engine.Console(ctx, name, 0)
	if lines == nil {
		return nil, err
	}
	kept := make([]string, len(lines))
	for i, line := range lines {
		kept[i] = secret.Kept(line)
	}
	return kept, err
}

func (s *Machinist) LifetimeOf(name string) model.Lifetime {
	rec, err := s.VM(name, "")
	if err != nil {
		return ""
	}
	return rec.Lifetime
}

func (s *Machinist) CheckStart(name string) error {
	_, err := s.VM(name, engine.VMStart)
	return err
}

func (s *Machinist) CheckStop(name string) error {
	_, err := s.VM(name, engine.VMStop)
	return err
}

func (s *Machinist) Start(ctx context.Context, name string) error {
	if err := s.CheckStart(name); err != nil {
		return err
	}
	if err := s.EngineUp(ctx); err != nil {
		return err
	}
	return s.engineAPI.Do(ctx, engine.VMStart, name)
}

func (s *Machinist) KeepAgents(name string, agents []string) error {
	rec, err := s.VM(name, engine.VMStop)
	if err != nil {
		return err
	}
	rec.Agents = agents
	return s.writeRecord(rec)
}

func (s *Machinist) Stop(ctx context.Context, name string) error {
	// invariant: a stop keeps the workspace, ephemeral or persistent (SP-5), and
	// like every action it is refused while the engine is stopped.
	if err := s.CheckStop(name); err != nil {
		return err
	}
	if err := s.EngineUp(ctx); err != nil {
		return err
	}
	return s.engineAPI.Do(ctx, engine.VMStop, name)
}

func (s *Machinist) StopDirect(ctx context.Context, name string) error {
	// value: the stop the engine's own Stop everything takes for a run that
	// outlived the engine.
	rec, err := s.VM(name, engine.VMStop)
	if err != nil {
		return err
	}
	rec.Want = model.WantStopped
	if err := s.writeRecord(rec); err != nil {
		return err
	}
	if err := s.powerOff(ctx, name); err != nil {
		return err
	}
	s.retireAgent(ctx, name)
	return nil
}

func (s *Machinist) restartVM(ctx context.Context, name string, say func(phase, detail string)) (model.RestartReport, error) {
	if _, err := s.VM(name, engine.VMRestart); err != nil {
		return model.RestartReport{}, err
	}
	if err := s.EngineUp(ctx); err != nil {
		return model.RestartReport{}, err
	}
	started := s.h.Now()
	say("restarting", "poweroff, then start")
	if err := s.engineAPI.Do(ctx, engine.VMRestart, name); err != nil {
		return model.RestartReport{}, err
	}
	say("back", "the engine runs it again")
	addr, err := s.engine.Address(ctx, name, ipWait)
	if err != nil {
		return model.RestartReport{}, &model.UnhealthyError{Name: name, Step: model.StepAddress, Detail: "none after 60 s"}
	}
	sshCtx, cancel := context.WithTimeout(ctx, sshWait)
	defer cancel()
	if out, err := s.engineAPI.GuestExec(sshCtx, name, "true", ""); err != nil || out.ExitCode != 0 {
		return model.RestartReport{}, &model.UnhealthyError{Name: name, Step: model.StepSSH, Detail: "none after 30 s"}
	}
	return model.RestartReport{Name: name, VM: true, IP: addr.String(), Waited: s.h.Now().Sub(started)}, nil
}

func (s *Machinist) CheckDelete(name string) error {
	// invariant: a VM whose create still runs is no one else's to delete:
	// its create would go on writing a VM that is gone.
	r, err := s.VM(name, engine.VMDelete)
	if err != nil {
		return err
	}
	return s.notBeingCreated(r)
}

func (s *Machinist) notBeingCreated(r model.VMRecord) error {
	if r.Birth != nil && r.Birth.Known() && s.runs(r.Owner) {
		return &model.BeingCreatedError{Name: r.Name, Mac: s.ThisMacName()}
	}
	return nil
}

func (s *Machinist) Delete(ctx context.Context, name string) (model.DeleteReport, error) {
	if err := s.CheckDelete(name); err != nil {
		return model.DeleteReport{}, err
	}
	if err := s.EngineUp(ctx); err != nil {
		return model.DeleteReport{}, err
	}
	if err := s.engineAPI.Do(ctx, engine.VMDelete, name); err != nil {
		return model.DeleteReport{}, err
	}
	return model.DeleteReport{Name: name}, nil
}

func (s *Machinist) Remove(ctx context.Context, name string) (model.DeleteReport, error) {
	// invariant: Remove is the whole delete, run by the engine: the run stops,
	// the disk, record, files and block go, and the VM's leases and their serve
	// entries are released.
	report := model.DeleteReport{Name: name}
	s.retireAgent(ctx, name)
	if s.engineState(ctx, name) == contract.DiskRunning {
		if err := s.engine.Shutdown(ctx, name, hardStopWait); err != nil {
			return report, err
		}
	}
	if s.engineState(ctx, name) != diskMissing {
		if err := s.engine.Delete(ctx, name); err != nil {
			return report, err
		}
	}
	if s.rel != nil {
		if err := s.rel.ReleaseVM(ctx, name); err != nil {
			report.Warnings = append(report.Warnings, err.Error())
		}
	}
	s.removeFiles(name, true)
	if s.o.Terminals != nil {
		if err := s.o.Terminals.Remove(ctx, name); err != nil {
			report.Warnings = append(report.Warnings, err.Error())
		}
	}
	if s.o.Uptime != nil {
		if err := s.o.Uptime.Delete(ctx, name); err != nil {
			report.Warnings = append(report.Warnings, err.Error())
		}
	}
	if err := blocksFailure(s.o.SSHConfig.Remove(ctx, name), false); err != nil {
		report.Warnings = append(report.Warnings, err.Error())
	}
	return report, nil
}
