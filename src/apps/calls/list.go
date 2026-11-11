package calls

import (
	"context"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/section"
)

type ListIn struct{}

var List = call.Watch[Deps, ListIn, []apps.Report]{
	Name: "apps.list", Section: section.Apps, Where: call.AtTarget, Applies: apps.Applies,
	Run: func(ctx context.Context, d Deps, _ ListIn, _ call.Progress) (<-chan call.Update[[]apps.Report], error) {
		return call.Updates(ctx, d.Apps.Watch(ctx)), nil
	},
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
				return ShowOut{Report: apps.Report{App: t, State: catalog.StateAbsent}}, nil
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
	return last
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
