package calls

import (
	"context"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

func (s *Machinist) CheckEdit(name string, c model.Change) error {
	rec, err := s.VM(name, "edit")
	if err != nil {
		return err
	}
	switch {
	case c.None():
		return &model.NothingToChangeError{Name: name}
	case (c.Check != "" || c.NoCheck) && rec.Lifetime != model.AlwaysOn:
		return &model.KindRuleError{Why: errCheckForServers.Why, Fix: "a workspace's health is three steps: station machines " + name}
	case c.Check != "" && c.NoCheck:
		return &model.KindRuleError{Why: "--check and --no-check ask opposite things", Fix: "station machines edit " + name + " --check <url>"}
	case c.Check != "" && !validCheck(c.Check):
		return &model.BadCheckError{Value: c.Check}
	}
	return nil
}

func (s *Machinist) Edit(ctx context.Context, name string, c model.Change) (model.Edited, error) {
	// invariant: an edit writes the sizes the VM boots with; a running VM keeps
	// its own until its next start, and a disk is never shrunk. A server's
	// check is its record's: the next health run reads it.
	r := c.Resources
	if err := s.CheckEdit(name, c); err != nil {
		return model.Edited{}, err
	}
	rec, err := s.VM(name, "edit")
	if err != nil {
		return model.Edited{}, err
	}
	if r.DiskBytes != 0 && r.DiskBytes < rec.Resources.DiskBytes {
		return model.Edited{}, &model.DiskShrinksError{Name: name, Has: rec.Resources.DiskBytes}
	}
	if err := s.EngineUp(ctx); err != nil {
		return model.Edited{}, err
	}
	size := rec.Resources
	if r.CPUs != 0 {
		size.CPUs = r.CPUs
	}
	if r.MemoryBytes != 0 {
		size.MemoryBytes = r.MemoryBytes
	}
	if r.DiskBytes != 0 {
		size.DiskBytes = r.DiskBytes
	}
	if r != (model.Resources{}) {
		if err := s.engine.Resize(ctx, name, model.Size{CPUs: size.CPUs, Memory: size.MemoryBytes, Disk: size.DiskBytes}); err != nil {
			return model.Edited{}, err
		}
	}
	rec.Resources = size
	switch {
	case c.NoCheck:
		rec.Check = ""
	case c.Check != "":
		rec.Check = c.Check
	}
	if err := s.writeRecord(rec); err != nil {
		return model.Edited{}, err
	}
	return model.Edited{Name: name, Changed: r, Check: model.ShownCheck(c.Check), NoCheck: c.NoCheck, Running: s.engineState(ctx, name) == contract.DiskRunning}, nil
}
