package awakecmd

import (
	"context"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	awakecalls "github.com/jterrazz/studio-station/apps/engine/awake/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageawake "github.com/jterrazz/studio-station/apps/engine/pages/awake"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	Now         func() time.Time
	Zone        *time.Location
	// value: every Mac's Awake, This Mac first.
	Every func(ctx context.Context) (pageawake.AllMachines, error)
	// value: the short name a Mac is said by in a command.
	Label func(mac string) string
}

type Input struct {
	Words        string
	Lid, Battery bool
	// value: the VM whose agent the keep-awake follows.
	Run string
}

func by(in *Input) string {
	words := "station awake " + in.Words
	if in.Lid {
		words += " --lid"
	}
	if in.Battery {
		words += " --battery"
	}
	if in.Run != "" {
		words += " --until-done " + in.Run
	}
	return strings.TrimSpace(words)
}

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[Input]{
		Use:     "awake [<for>]",
		Short:   "Keep a Mac from sleeping: every Mac's mode, a keep-awake for a while, or off",
		Example: "  station awake\n  station awake 4h --lid\n  station awake until 18:00\n  station awake --until-done linux-1\n  station awake off",
		Group:   kit.OnAMachine,
		Args:    cobra.MaximumNArgs(2),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.Lid, "lid", false, "stay awake with the lid closed too (the screen still turns off)")
			f.BoolVar(&in.Battery, "battery", false, "stay awake when unplugged")
			f.StringVar(&in.Run, "until-done", "", "stay awake until the agent working in this VM is done (24h at most)")
		},
		Bind:  func(in *Input, args []string) error { in.Words = strings.Join(args, " "); return nil },
		Read:  func(in *Input) bool { return in.Words == "" && !in.Lid && !in.Battery && in.Run == "" },
		Pages: true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if in.Words == "" && !in.Lid && !in.Battery && in.Run == "" {
				return read(ctx, d, env, io)
			}
			return keep(ctx, d, env, in, io)
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}

func read(ctx context.Context, d Deps, env kit.Env, io kit.IO) error {
	f := pageawake.AllMachines{Now: d.Now(), Zone: d.Zone}
	if env.Far() {
		r, err := call.Read(ctx, env.Calls, env.Target(), awakecalls.Read, awakecalls.ReadIn{}, nil)
		if err != nil {
			return err
		}
		f.Reports = []awake.Report{r}
	} else {
		every, err := d.Every(ctx)
		if err != nil {
			return err
		}
		f = every
	}
	f.Label, f.Far = d.Label, env.Far()
	return io.Page(pageawake.Page(f, d.ThisMachine), pageawake.Contract(f))
}

func keep(ctx context.Context, d Deps, env kit.Env, in *Input, io kit.IO) error {
	words := in.Words
	if words == "" && in.Run == "" {
		words = "4h"
	}
	req := awake.Request{Lid: in.Lid, Battery: in.Battery, By: by(in), Words: words, Run: in.Run}
	if words != "" {
		term, err := awake.For(words, d.Now(), d.Zone)
		req.For = term
		if err != nil {
			req.For = awake.Longest + time.Minute
		}
	}
	a, err := call.Invoke(ctx, env.Calls, env.Target(), awakecalls.Keep, req, nil)
	if err != nil {
		return err
	}
	return io.Page(pageawake.Kept(a, d.Label(a.Mac), d.Now(), d.Zone, d.ThisMachine), a)
}
