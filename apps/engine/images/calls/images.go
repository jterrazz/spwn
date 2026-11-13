package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/images"
)

type ListIn struct{}

var List = call.Call[Deps, ListIn, []images.Report]{
	Name: "images.list", Where: call.AtTarget, Applies: images.Applies,
	Run: func(ctx context.Context, d Deps, _ ListIn, _ call.Progress) ([]images.Report, error) {
		return d.Images.List(ctx)
	},
}

type BuildIn struct {
	// value: no name builds every image missing or whose recipe changed.
	Names []string
	// value: the linux station binary the image carries; empty takes the
	// release's.
	Binary string
}

var Build = call.Call[Deps, BuildIn, []images.Record]{
	Name: "images.build", Where: call.InEngine, Applies: images.Applies, Needs: images.Needs(),
	Errors: []fail.Code{Unknown},
	Run: func(ctx context.Context, d Deps, in BuildIn, p call.Progress) ([]images.Record, error) {
		steps := stepper(p)
		records, err := d.Images.Build(ctx, in.Names, in.Binary, steps.step)
		steps.end(err)
		return records, worded(err)
	},
}

type UpdateIn struct {
	// value: no name updates every image held here.
	Names []string
}

var Update = call.Call[Deps, UpdateIn, []images.Update]{
	Name: "images.update", Where: call.InEngine, Applies: images.Applies, Needs: images.Needs(),
	Errors: []fail.Code{Unknown},
	Run: func(ctx context.Context, d Deps, in UpdateIn, p call.Progress) ([]images.Update, error) {
		steps := stepper(p)
		updates, err := d.Images.Update(ctx, in.Names, steps.step)
		steps.end(err)
		return updates, worded(err)
	},
}

type steps struct {
	p    call.Progress
	last string
}

func stepper(p call.Progress) *steps { return &steps{p: p} }

func (s *steps) step(label string) {
	if s.last != "" {
		s.p.Done(s.last, nil)
	}
	s.last = label
	s.p.Step(label, "")
}

func (s *steps) end(err error) {
	if s.last != "" {
		s.p.Done(s.last, err)
	}
}

type DeleteIn struct {
	Image string
	// value: empty deletes every version of the image.
	Version string
	Confirm string
}

var Delete = call.Call[Deps, DeleteIn, []images.Record]{
	Name: "images.delete", Where: call.InEngine, Applies: images.Applies, Needs: images.Needs(), Confirm: confirm.DeleteImage,
	Errors: []fail.Code{Unknown, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in DeleteIn, _ call.Progress) ([]images.Record, error) {
		if err := d.Images.CheckDelete(ctx, in.Image, in.Version); err != nil {
			return nil, worded(err)
		}
		name := in.Image
		if in.Version != "" {
			name += "@" + in.Version
		}
		if err := confirm.Require(confirm.For(confirm.DeleteImage, d.Kind), name, in.Confirm); err != nil {
			return nil, err
		}
		return d.Images.Delete(ctx, in.Image, in.Version)
	},
}
