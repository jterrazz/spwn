package settingsapply

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/page"
	pagesettings "github.com/jterrazz/studio-station/src/pages/settings"
	"github.com/jterrazz/studio-station/src/settings"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	// value: the catalog this run reads: which settings need root.
	Catalog     catalog.Catalog
	Names       func() []string
	ThisMachine page.Context
	Now         func() time.Time
}

type Input struct {
	Names       []string
	AllMachines bool
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "apply [<setting...>]",
		Short:   "Apply settings here; bare, every one not applied or changed since; one with values asks them on a terminal",
		Example: "  station settings apply\n  station settings apply starship-dotfiles\n  station settings apply claude-token\n  station settings apply starship-dotfiles --all-machines",
		Args:    cobra.ArbitraryArgs,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.AllMachines, "all-machines", false, "on every Mac and workspace; servers are skipped, admin settings said with their command")
		},
		Bind:     func(in *Input, args []string) error { in.Names = args; return nil },
		Asks:     true,
		Confirm:  func(in *Input) string { return pagesettings.ApplyConfirm(in.Names).Question },
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if in.AllMachines {
				return everyMachine(ctx, env, d, in, io)
			}
			var changes []settings.Change
			err := io.Confirmed(func(answer string) (err error) {
				changes, err = call.Invoke(ctx, env.Calls, env.Target(), settingscalls.Apply, settingscalls.ApplyIn{Names: in.Names, Interactive: io.Interactive, Confirm: answer}, nil)
				return err
			})
			if err != nil {
				return err
			}
			if err := io.Page(pagesettings.Changes("apply", changes, d.ThisMachine), nil); err != nil {
				return err
			}
			return partial(changes, "apply")
		},
	}, env)
}

func everyMachine(ctx context.Context, env kit.Env, d Deps, in *Input, io kit.IO) error {
	// invariant: every Mac and running workspace in turn, This Mac first; a
	// server and a stopped workspace are skipped and said, an admin setting
	// is never applied but said with the command that applies it there, a
	// Mac that does not answer is a failure.
	list, err := call.Read(ctx, env.Calls, env.Self, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
	if err != nil {
		return err
	}
	livesOn, servers := settings.LivesOnAll(list.All)
	var applies []pagesettings.MachineApply
	failed := 0
	for _, p := range livesOn {
		a := pagesettings.MachineApply{Machine: p.Name}
		switch p.Away {
		case settings.Silent:
			a.Err = p.Name + " does not answer"
			failed++
		case settings.Stopped:
			a.Skipped = "stopped"
		default:
			names := in.Names
			if p.Mac {
				names, a.Admin = admins(d.Catalog, in.Names, p)
			}
			if len(names) == 0 && len(in.Names) > 0 {
				break
			}
			err := applyLiving(ctx, env, p, names, io, &a)
			var refused *fail.Error
			switch {
			case errors.As(err, &refused) && refused.Type == fail.NotForThisMachine:
				a.Skipped = kit.Translate(err, "settings apply").Message
			case err != nil:
				a.Err = kit.Translate(err, "settings apply").Message
				failed++
			}
			if brokenOf(a.Changes) > 0 {
				failed++
			}
		}
		applies = append(applies, a)
	}
	for _, s := range servers {
		applies = append(applies, pagesettings.MachineApply{Machine: s, Skipped: "server"})
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := io.Page(pagesettings.EveryMachine(applies, d.ThisMachine), nil); err != nil {
		return err
	}
	if failed > 0 {
		return kit.PartialFailure{Failed: failed, Total: len(applies), Noun: "machines", FirstHint: "station machines"}
	}
	return nil
}

// invariant: a setting that does not live on the machine is dropped and
// said, and the rest are applied there; the machine is asked again, as it
// alone knows which of its settings live there.
func applyLiving(ctx context.Context, env kit.Env, p settings.LivesOn, names []string, io kit.IO, a *pagesettings.MachineApply) error {
	for {
		err := io.Confirmed(func(answer string) (err error) {
			a.Changes, err = call.Invoke(ctx, env.Calls, p.Ref, settingscalls.Apply, settingscalls.ApplyIn{Names: names, Interactive: io.Interactive, Confirm: answer}, nil)
			return err
		})
		var refused *fail.Error
		if !errors.As(err, &refused) || refused.Code != settingscalls.NotForThisMachine || !slices.Contains(names, refused.Subject) {
			return err
		}
		a.Elsewhere = append(a.Elsewhere, pagesettings.Elsewhere{Name: refused.Subject, Why: kit.Translate(err, "settings apply").Message})
		names = slices.DeleteFunc(slices.Clone(names), func(n string) bool { return n == refused.Subject })
		if len(names) == 0 {
			return nil
		}
	}
}

// value: the names a Mac is asked to apply, and the admin settings among
// them, each with the command that applies it on that Mac's own terminal.
func admins(cat catalog.Catalog, names []string, p settings.LivesOn) ([]string, []pagesettings.AdminSetting) {
	var asked []string
	var admin []pagesettings.AdminSetting
	for _, n := range names {
		if it, _ := cat.Setting(n); !it.NeedsRoot {
			asked = append(asked, n)
			continue
		}
		admin = append(admin, pagesettings.AdminSetting{Name: n, Command: kit.AdminApply(n, p.Name, p.ThisMachine)})
	}
	return asked, admin
}

func brokenOf(changes []settings.Change) int {
	n := 0
	for _, c := range changes {
		if c.Result == catalog.StateBroken {
			n++
		}
	}
	return n
}

func partial(changes []settings.Change, verb string) error {
	failed, hint := 0, ""
	for _, c := range changes {
		if c.Result == catalog.StateBroken {
			if failed == 0 {
				hint = "re-run it: station settings " + verb + " " + c.Item
			}
			failed++
		}
	}
	if failed == 0 {
		return nil
	}
	return kit.PartialFailure{Failed: failed, Total: len(changes), Noun: "settings", FirstHint: hint}
}
