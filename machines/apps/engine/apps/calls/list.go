package calls

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

type ListIn struct {
	// value: the run's shared reading of the apps dropped first, so they
	// are read again: what an interrupted install did is seen.
	Fresh bool
}

var List = call.Watch[Deps, ListIn, []apps.Report]{
	Name: "apps.list", Section: section.Apps, Where: call.AtTarget, Applies: apps.Applies,
	Run: func(ctx context.Context, d Deps, in ListIn, _ call.Progress) (<-chan call.Update[[]apps.Report], error) {
		if in.Fresh {
			d.Apps.Forget()
		}
		return call.Updates(ctx, placedFrom(ctx, d, d.Apps.Watch(ctx))), nil
	},
}

func originOf(d Deps, name string) machinefile.Origin {
	if d.Origin == nil {
		return machinefile.Origin{Name: name, Kind: machinefile.Builtin}
	}
	return d.Origin(name)
}

func stamped(d Deps, frame []apps.Report) []apps.Report {
	out := slices.Clone(frame)
	for i := range out {
		out[i].Origin = originOf(d, out[i].App.Name)
	}
	return out
}

func placedFrom(ctx context.Context, d Deps, frames <-chan []apps.Report) <-chan []apps.Report {
	out := make(chan []apps.Report)
	go func() {
		defer close(out)
		for frame := range frames {
			select {
			case out <- stamped(d, frame):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

type ShowIn struct{ Name string }

type ShowOut struct {
	Report apps.Report
	// value: the app lives on this machine; one that does not is shown with
	// where it lives instead.
	ThisMachine bool
}

var Show = call.Call[Deps, ShowIn, ShowOut]{
	Name: "apps.show", Section: section.Apps, Where: call.AtTarget, Applies: apps.Applies,
	Errors: []fail.Code{Unknown},
	Run: func(ctx context.Context, d Deps, in ShowIn, _ call.Progress) (ShowOut, error) {
		for _, r := range read(ctx, d) {
			if r.App.Name == in.Name {
				return ShowOut{Report: r, ThisMachine: true}, nil
			}
		}
		for _, t := range d.Known {
			if t.Name == in.Name {
				return ShowOut{Report: apps.Report{App: t, State: catalog.StateAbsent, Origin: originOf(d, t.Name)}}, nil
			}
		}
		if err := ctx.Err(); err != nil {
			return ShowOut{}, err
		}
		return ShowOut{}, coded(&apps.UnknownError{Name: in.Name})
	},
}

func read(ctx context.Context, d Deps) []apps.Report {
	var last []apps.Report
	for frame := range d.Apps.Watch(ctx) {
		last = frame
	}
	return stamped(d, last)
}

func installed(ctx context.Context, d Deps, names []string) error {
	// invariant: an app that is not installed is refused before anything
	// runs, before any question is asked.
	for _, r := range read(ctx, d) {
		for _, name := range names {
			if r.App.Name == name && !apps.StateOf(r).Installed() {
				return coded(&apps.NotInstalledError{App: name})
			}
		}
	}
	return ctx.Err()
}
