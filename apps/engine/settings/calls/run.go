package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type GitAliases interface {
	// value: one alias of the Git shortcuts, run by git with the words given.
	Git(ctx context.Context, alias string, words []string) error
}

type Shortcuts interface {
	// value: a shortcut that deletes every container, image, or both.
	Containers(ctx context.Context, name string) (settings.ContainersOutcome, error)
}

type GitIn struct {
	Alias string
	Words []string
}

var Git = call.Call[Deps, GitIn, struct{}]{
	Name: "settings.run-git", Section: section.Settings, Where: call.InProcess, Applies: settings.Applies,
	Errors: []fail.Code{NotApplied},
	Run: func(ctx context.Context, d Deps, in GitIn, _ call.Progress) (struct{}, error) {
		// invariant: v2's run git is git's own alias, once the Git shortcuts
		// are applied here; before, it is refused with the setting that
		// brings them.
		it, err := find(d, catalog.GitShortcutsSetting, true)
		if err != nil {
			return struct{}{}, err
		}
		if row, _ := judge(ctx, d, it, offThisMac(d)); row.State != settings.StateApplied && row.State != settings.StateChangedSince {
			return struct{}{}, fail.New(fail.Refused, NotApplied, it.Name, nil)
		}
		return struct{}{}, d.GitAliases.Git(ctx, in.Alias, in.Words)
	},
}

type ContainersIn struct {
	Name    string
	Confirm string
}

var Containers = call.Call[Deps, ContainersIn, settings.ContainersOutcome]{
	Name: "settings.run-containers", Section: section.Settings, Where: call.AtTarget, Applies: settings.Applies, Confirm: confirm.Remove,
	Errors: []fail.Code{confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in ContainersIn, _ call.Progress) (settings.ContainersOutcome, error) {
		if err := confirm.Require(confirm.For(confirm.Remove, d.Kind), in.Name, in.Confirm); err != nil {
			return settings.ContainersOutcome{}, err
		}
		return d.Shortcuts.Containers(ctx, in.Name)
	},
}
