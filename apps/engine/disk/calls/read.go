package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/disk"
)

type ReadIn struct{}

var Read = call.Watch[Deps, ReadIn, disk.Reading]{
	Name: "disk.read", Section: section.Disk, Where: call.AtTarget, Applies: disk.Applies,
	Run: func(ctx context.Context, d Deps, _ ReadIn, _ call.Progress) (<-chan call.Update[disk.Reading], error) {
		total, free, err := d.Disk.Volume(ctx)
		if err != nil {
			return nil, err
		}
		surveys, helds := d.Disk.Watch(ctx), d.Disk.Held(ctx)
		readings := make(chan disk.Reading)
		go func() {
			// invariant: a reading is sent once the survey and the hypervisor
			// have each given their frame, then on each newer value of either,
			// until both have ended.
			defer close(readings)
			var s disk.Survey
			var h disk.HeldReading
			surveyed, asked := false, false
			for surveys != nil || helds != nil {
				select {
				case got, ok := <-surveys:
					if !ok {
						surveys = nil
						continue
					}
					s, surveyed = got, true
				case got, ok := <-helds:
					if !ok {
						helds = nil
						continue
					}
					h, asked = got, true
				case <-ctx.Done():
					return
				}
				if !surveyed || !asked {
					continue
				}
				select {
				case readings <- disk.ReadingOf(total, free, s, h, d.Name):
				case <-ctx.Done():
					return
				}
			}
		}()
		return call.Updates(ctx, readings), nil
	},
}
