package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/images"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

func (s *Builder) owner() *machines.Process {
	if s.procs == nil {
		return nil
	}
	p, err := s.procs.Self()
	if err != nil {
		return nil
	}
	return &p
}

// value: a build's mark is the display of its step: a Mac that cannot write
// it still builds.
func (s *Builder) mark(m images.Making) {
	if s.marks != nil {
		_ = s.marks.Mark(context.Background(), m)
	}
}

func (s *Builder) clear(image string) {
	if s.marks != nil {
		_ = s.marks.Clear(context.Background(), image)
	}
}

func (s *Builder) making(ctx context.Context) []images.Making {
	// invariant: a build is making an image while the process it was marked
	// in runs; a mark whose process is gone, killed or its Mac restarted, is
	// no build, and marks that cannot be read are none: a mark is display
	// only.
	if s.marks == nil || s.procs == nil {
		return nil
	}
	marked, err := s.marks.Marked(ctx)
	if err != nil {
		return nil
	}
	var out []images.Making
	for _, m := range marked {
		if m.Owner != nil && s.procs.Runs(*m.Owner) {
			out = append(out, m)
		}
	}
	return out
}
