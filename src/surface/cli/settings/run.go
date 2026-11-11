package settingscmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/page"
	pagesettings "github.com/jterrazz/studio-station/src/pages/settings"
	"github.com/jterrazz/studio-station/src/settings"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

func Run(env kit.Env, children ...*cobra.Command) *cobra.Command {
	// value: v2's run family, hidden for one version (SPEC §4.3): its
	// shortcuts are the Git and Docker shortcuts settings now.
	cmd := kit.Group("run", "v2's shortcuts, now settings", "", env, children...)
	cmd.Hidden = true
	return cmd
}

// value: v2's run git shortcut word of an alias of the Git shortcuts; the
// others kept their word.
func oldGitWord(alias string) string {
	switch alias {
	case "lg":
		return "log"
	case "pushup":
		return "push"
	case "st":
		return "status"
	}
	return alias
}

type runGitIn struct{ Words []string }

func RunGit(env kit.Env, aliases []catalog.GitAlias) *cobra.Command {
	// value: v2's run git <shortcut>, now git's own alias (SPEC §4.3): it
	// runs that alias through the Git shortcuts setting.
	leaves := make([]*cobra.Command, 0, len(aliases))
	for _, a := range aliases {
		leaves = append(leaves, gitLeaf(env, oldGitWord(a.Name), a.Name))
	}
	return kit.Group(catalog.GitShortcutsGroup, "Git shortcuts, now git's own aliases", "", env, leaves...)
}

func gitLeaf(env kit.Env, old, alias string) *cobra.Command {
	return kit.AliasShell(kit.Command(kit.Leaf[runGitIn]{
		Use:     old,
		Short:   catalog.GitShortcutLine(alias),
		Args:    cobra.ArbitraryArgs,
		RawArgs: true,
		Bind: func(in *runGitIn, args []string) error {
			in.Words = args
			return nil
		},
		Run: func(ctx context.Context, in *runGitIn, _ kit.IO) error {
			_, err := call.Invoke(ctx, env.Calls, env.Target(), settingscalls.Git, settingscalls.GitIn{Alias: alias, Words: in.Words}, nil)
			return err
		},
	}, env), catalog.GitShortcutLine(alias))
}

type runContainersIn struct{}

// value: v2's run shortcut of containers by its old word, the shell
// function that replaced it and what it removes.
type containerShortcut struct{ old, function, removes string }

// value: alias of v2's run shortcuts of containers, each kept hidden by
// its old word.
func containerShortcuts() []containerShortcut {
	return []containerShortcut{
		{"rm", "dclean", "every container"},
		{"rmi", "dclean-images", "every image"},
		{"reset", "dreset", "every container and image"},
	}
}

func RunContainers(env kit.Env, thisMac page.Context) *cobra.Command {
	// value: v2's run shortcuts of containers (rm, rmi, reset), now shell
	// functions: the same steps, asking as before.
	leaves := make([]*cobra.Command, 0, len(containerShortcuts()))
	for _, sc := range containerShortcuts() {
		leaves = append(leaves, containerLeaf(env, thisMac, sc))
	}
	return kit.Group(catalog.ContainerShortcutsWas, "Container shortcuts, now shell functions", "", env, leaves...)
}

func containerLeaf(env kit.Env, thisMac page.Context, sc containerShortcut) *cobra.Command {
	return kit.AliasShell(kit.Command(kit.Leaf[runContainersIn]{
		Use:     sc.old,
		Short:   sc.function + ": remove " + sc.removes,
		Args:    cobra.NoArgs,
		Asks:    true,
		Confirm: func(*runContainersIn) string { return "remove " + sc.removes + "?" },
		Verb:    func(*runContainersIn) string { return "remove " + sc.removes },
		Run: func(ctx context.Context, _ *runContainersIn, io kit.IO) error {
			var outcome settings.ContainersOutcome
			err := io.Confirmed(func(answer string) (err error) {
				outcome, err = call.Invoke(ctx, env.Calls, env.Target(), settingscalls.Containers, settingscalls.ContainersIn{Name: sc.function, Confirm: answer}, nil)
				return err
			})
			if err != nil {
				return err
			}
			return io.Page(pagesettings.Removed(outcome, "run "+catalog.ContainerShortcutsWas+" "+sc.old, thisMac), nil)
		},
	}, env), sc.function)
}
