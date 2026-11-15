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
		surveys := d.Disk.Watch(ctx)
		readings := make(chan disk.Reading)
		go func() {
			defer close(readings)
			for s := range surveys {
				r := disk.ReadingOf(total, free, s, d.Name)
				select {
				case readings <- r:
				case <-ctx.Done():
					return
				}
			}
		}()
		return call.Updates(ctx, readings), nil
	},
}
