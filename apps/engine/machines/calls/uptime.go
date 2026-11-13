package calls

import (
	"context"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

func (s *Machinist) MarkUptime(ctx context.Context, m model.UptimeMark) error {
	// invariant: machines.uptime-mark, the engine-only call of SPEC B5.2: the
	// keep job's one way to a server's record, so the store keeps one
	// writer; no surface reaches it, and a mark that changes nothing writes
	// nothing.
	if s.o.Uptime == nil {
		return nil
	}
	was, err := s.o.Uptime.Get(ctx, m.Server)
	if err != nil {
		return err
	}
	now, changed := was.Marked(m)
	if !changed {
		return nil
	}
	return s.o.Uptime.Put(ctx, m.Server, now)
}
