package calls

import (
	"context"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
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

func (s *Machinist) NotCreated(name string) bool {
	rec, err := s.VM(name, "")
	return err == nil && rec.Birth != nil && rec.Birth.Failed
}

func (s *Machinist) CheckStop(name string) error {
	_, err := s.VM(name, engine.VMStop)
	return err
}

func (s *Machinist) Start(ctx context.Context, name string) error {
	// invariant: a start of a VM whose create failed is that create's
	// retry: it runs the VM, then its set-up from its first boot on, as the
	// create would have; its birth goes once the set-up answers, and
	// otherwise holds the new failure (its step, cause and console lines
	// replacing the old), which the start answers as its error. No VM
	// is run for a VM user its rule refuses (vmUserRefused).
	journal.Mark(ctx, s.h.Now(), "checks")
	rec, err := s.VM(name, engine.VMStart)
	if err != nil {
		return err
	}
	if err := s.EngineUp(ctx); err != nil {
		return err
	}
	if err := s.vmUserRefused(); err != nil {
		return err
	}
	if _, err := s.hypervisorState(ctx, name); err != nil {
		return err
	}
	img, _ := s.imageOf(ctx, rec.Image)
	retry := rec.Birth != nil && rec.Birth.Failed
	journal.Mark(ctx, s.h.Now(), "boot")
	if err := s.engineAPI.Do(ctx, engine.VMStart, name); err != nil {
		if !retry {
			return err
		}
		again := s.unhealthy(ctx, name, err)
		if ctx.Err() == nil {
			s.failBirth(name, *model.BirthAt(model.BornBooting, rec.Birth.Steps == len(rec.Birth.Step.Enum()), s.h.Now()), again)
		}
		return again
	}
	if !retry {
		return nil
	}
	journal.Mark(ctx, s.h.Now(), "reachable")
	if err := s.setUp(ctx, img, model.CreateRequest{Name: name, Lifetime: rec.Lifetime}); err != nil {
		failed := &model.UnhealthyError{Name: name, Step: model.StepSSH, Detail: strings.TrimSpace(err.Error())}
		if ctx.Err() == nil {
			s.failBirth(name, *rec.Birth, failed)
		}
		return failed
	}
	_ = s.engineAPI.Do(ctx, engine.VMReady, name)
	return s.writeBirth(name, func(stored *model.Record) {
		if stored.Birth != nil && stored.Birth.Failed {
			stored.Birth = nil
		}
	})
}

// invariant: no VM is run for a VM user its rule refuses, whatever its
// image: its start and its restart refuse it before the VM is touched, as a
// create does.
func (s *Machinist) vmUserRefused() error {
	cfg, err := s.prefs()
	if err != nil {
		return err
	}
	_, bad := s.vmUser(cfg)
	return bad
}

func (s *Machinist) Stop(ctx context.Context, name string) error {
	// invariant: a stop keeps the workspace, ephemeral or persistent (SP-5), and
	// like every action it is refused while the engine is stopped.
	journal.Mark(ctx, s.h.Now(), "checks")
	if err := s.CheckStop(name); err != nil {
		return err
	}
	if err := s.EngineUp(ctx); err != nil {
		return err
	}
	journal.Mark(ctx, s.h.Now(), "poweroff")
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
	return nil
}

func (s *Machinist) restartVM(ctx context.Context, name string, say func(phase, detail string)) (model.RestartReport, error) {
	_, found := s.VM(name, engine.VMRestart)
	if found != nil {
		return model.RestartReport{}, found
	}
	if err := s.EngineUp(ctx); err != nil {
		return model.RestartReport{}, err
	}
	if err := s.vmUserRefused(); err != nil {
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
		return model.RestartReport{}, s.sshFails(ctx, name, "none after 30 s")
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
	if s.beingCreated(r) {
		return &model.BeingCreatedError{Name: r.Name, Mac: s.ThisMacName()}
	}
	return nil
}

func (s *Machinist) beingCreated(r model.VMRecord) bool {
	return r.Birth != nil && r.Birth.Known() && s.runs(r.Owner)
}

func (s *Machinist) Creating(name string) (model.Birth, bool) {
	// value: how far the create of a VM of this Mac is while it still
	// runs; false once it ended, or for a name no VM of this Mac holds.
	r, err := s.VM(name, "")
	if err != nil || !s.beingCreated(r) {
		return model.Birth{}, false
	}
	return *r.Birth, true
}

func (s *Machinist) Births(ctx context.Context, name string) (<-chan model.Birth, bool) {
	// invariant: every step a VM's create reaches, the one it is at first,
	// read from the record machines.list reads its Birth from; closed when
	// the create ends or ctx does. False for a VM no create of this Mac is
	// making.
	birth, ok := s.Creating(name)
	if !ok {
		return nil, false
	}
	out := make(chan model.Birth)
	go func() {
		defer close(out)
		var sent *model.Birth
		for ; ok; birth, ok = s.Creating(name) {
			if sent == nil || !sent.Same(birth) {
				select {
				case out <- birth:
				case <-ctx.Done():
					return
				}
				at := birth
				sent = &at
			}
			if s.h.Sleep(ctx, pollEvery) != nil {
				return
			}
		}
	}()
	return out, true
}

func (s *Machinist) Delete(ctx context.Context, name string) (model.DeleteReport, error) {
	if err := s.CheckDelete(name); err != nil {
		return model.DeleteReport{}, err
	}
	if err := s.EngineUp(ctx); err != nil {
		return model.DeleteReport{}, err
	}
	if _, err := s.hypervisorState(ctx, name); err != nil {
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
	// invariant: a hypervisor that does not answer refuses the delete with
	// nothing erased: its silence never reads as a disk already gone.
	report := model.DeleteReport{Name: name}
	state, err := s.hypervisorState(ctx, name)
	if err != nil {
		return report, err
	}
	if state == contract.DiskRunning {
		if stopErr := s.engine.Shutdown(ctx, name, hardStopWait); stopErr != nil {
			return report, stopErr
		}
		if state, err = s.hypervisorState(ctx, name); err != nil {
			return report, err
		}
	}
	missing := state == diskMissing
	if !missing {
		if err := s.engine.Delete(ctx, name); err != nil {
			return report, err
		}
	}
	if missing {
		// invariant: a disk already gone, deleted by hand, leaves no entry in
		// the hypervisor's ledger: a VM made by hand later under the name is
		// never taken for the one station cloned.
		if err := s.engine.Forget(ctx, name); err != nil {
			report.Warnings = append(report.Warnings, err.Error())
		}
	}
	if s.rel != nil {
		if err := s.rel.ReleaseVM(ctx, name); err != nil {
			report.Warnings = append(report.Warnings, err.Error())
		}
	}
	s.removeFiles(name, true)
	if s.o.Uptime != nil {
		if err := s.o.Uptime.Delete(ctx, name); err != nil {
			report.Warnings = append(report.Warnings, err.Error())
		}
	}
	if err := blocksFailure(s.o.SSHConfig.Remove(ctx, name, name+"."+s.ThisMacName()), false); err != nil {
		report.Warnings = append(report.Warnings, err.Error())
	}
	return report, nil
}
