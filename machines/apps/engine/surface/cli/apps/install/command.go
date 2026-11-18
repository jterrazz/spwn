package appsinstall

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageapps "github.com/jterrazz/studio-station/apps/engine/pages/apps"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	Names       func() []string
	ThisMachine page.Context
	Now         func() time.Time
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:      "install <app...>",
		Short:    "Install apps here, in dependency order, then the settings each one brings",
		Example:  "  station apps install jq fd\n  station apps install frontend-design",
		Args:     cobra.MinimumNArgs(1),
		Bind:     bind,
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			return Run(ctx, d, env, in.Names, io)
		},
	}, env)
}

func Run(ctx context.Context, d Deps, env kit.Env, names []string, io kit.IO) error {
	// value: the install of apps and what it prints, for this leaf and for
	// the terminal that answers a missing need with one.
	start := d.Now()
	steps := kit.Steps(io)
	outcomes, err := call.Invoke(ctx, env.Calls, env.Target(), appscalls.Install, appscalls.InstallIn{Names: names, Interactive: io.Interactive}, steps)
	steps.Stop()
	if err != nil {
		return err
	}
	if err := io.Page(pageapps.Outcomes("install", kit.Worded(outcomes), ran(env, d.Now().Sub(start)), d.ThisMachine), nil); err != nil {
		return err
	}
	return partial(outcomes, "station apps install ")
}

func partial(outcomes []apps.Outcome, repair string) error {
	n, first := apps.FailedOf(outcomes)
	if n == 0 {
		return nil
	}
	return kit.PartialFailure{Failed: n, Total: len(outcomes), Noun: "apps", FirstHint: repair + first}
}

func ran(env kit.Env, took time.Duration) pageapps.Run {
	run := pageapps.Run{Took: took}
	if env.Far() {
		run.On = env.Target().Name
	}
	return run
}
