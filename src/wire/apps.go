package wire

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/platform/local"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/apps/adapter/app"
	"github.com/jterrazz/studio-station/src/apps/adapter/brew"
	"github.com/jterrazz/studio-station/src/apps/adapter/bun"
	"github.com/jterrazz/studio-station/src/apps/adapter/mas"
	"github.com/jterrazz/studio-station/src/apps/adapter/mise"
	"github.com/jterrazz/studio-station/src/apps/adapter/npm"
	"github.com/jterrazz/studio-station/src/apps/adapter/providedby"
	"github.com/jterrazz/studio-station/src/apps/adapter/script"
	"github.com/jterrazz/studio-station/src/apps/adapter/shell"
	"github.com/jterrazz/studio-station/src/apps/adapter/skills"
	"github.com/jterrazz/studio-station/src/apps/adapter/uv"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	pagesettings "github.com/jterrazz/studio-station/src/pages/settings"
	"github.com/jterrazz/studio-station/src/settings"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
	appscmd "github.com/jterrazz/studio-station/src/surface/cli/apps"
	appsinstall "github.com/jterrazz/studio-station/src/surface/cli/apps/install"
	appsremove "github.com/jterrazz/studio-station/src/surface/cli/apps/remove"
	appsupdate "github.com/jterrazz/studio-station/src/surface/cli/apps/update"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
	"github.com/jterrazz/studio-station/src/surface/cli/words"
)

func appsCommand(env kit.Env, m machine) *cobra.Command {
	names := placedNames(appsOf(env, m))
	return appscmd.Command(appscmd.Deps{Names: names, ThisMachine: m.where, Apps: m.cat.Apps}, env,
		appsinstall.Command(appsinstall.Deps{Names: names, ThisMachine: m.where, Now: m.sys.Clock.Now}, env),
		appsupdate.Command(appsupdate.Deps{ThisMachine: m.where, Now: m.sys.Clock.Now}, env),
		appsremove.Command(appsremove.Deps{Names: names, ThisMachine: m.where, Now: m.sys.Clock.Now}, env),
	)
}

// invariant: station tools and its verbs are the hidden v2 words of station
// apps for one release (SPEC §4.3); upgrade's --via is update's --method.
func appsAlias(env kit.Env, m machine) *cobra.Command {
	names := placedNames(appsOf(env, m))
	update := appsupdate.Via(appsupdate.Command(appsupdate.Deps{ThisMachine: m.where, Now: m.sys.Clock.Now}, env))
	return kit.Alias("tools", appscmd.Command(appscmd.Deps{Names: names, ThisMachine: m.where, Apps: m.cat.Apps}, env,
		kit.Alias("install", appsinstall.Command(appsinstall.Deps{Names: names, ThisMachine: m.where, Now: m.sys.Clock.Now}, env), "install"),
		kit.Alias("uninstall", appsremove.Command(appsremove.Deps{Names: names, ThisMachine: m.where, Now: m.sys.Clock.Now}, env), "remove"),
		kit.Alias("upgrade", update, "update"),
	), "apps")
}

func placedNames(s *appscalls.Shelf) func() []string {
	return func() []string {
		placed := s.Placed()
		out := make([]string, 0, len(placed))
		for _, t := range placed {
			out = append(out, t.Name)
		}
		return out
	}
}

func appsOf(env kit.Env, m machine) *appscalls.Shelf {
	var ask kit.Asker
	if env.Interactive {
		ask = env.Prompt
	}
	registry, bundler := installers(m, env.Interactive)
	return appscalls.NewShelf(appsMachine{managing(m)}, m.cat, configurer{settings: settingsOf(m), ask: ask}, appscalls.ShelfOptions{
		Pool:        m.pool,
		ThisMachine: m.kind,
		Root:        m.systemRoot(),
		Installers:  registry,
		Bundler:     bundler,
		VMs:         m.machines(env.Interactive),
		Logs:        filepath.Join(m.paths.State, "logs"),
	})
}

// invariant: every command station runs for apps, a manager's read, install or
// removal and an app's version read, runs from the owner's home with mise
// pinned to its global file, never from the folder station was started in: a
// project there that pins another node would pick which npm lists what node
// holds, while `mise unuse --global node` takes back the global one.
func managing(m machine) local.System {
	sys := m.sys
	home := sys.Env.Home()
	sys.Run = local.Anchored(sys.Run, home, mise.Pinned(home)...)
	return sys
}

func installers(m machine, terminal bool) (contract.Installers, contract.Bundler) {
	run := managing(m).Run
	runtimes := mise.New(run)
	bundle := brew.New(run, func() []string { return runtimes.ShimsFirst(m.sys.Env.Home(), m.vars) })
	registry := contract.Installers{}
	for _, i := range []contract.Installer{
		bundle.Formulae(), bundle.Casks(),
		mas.New(run), npm.New(run, bundle), uv.New(run, bundle),
		bun.New(run), runtimes,
		skills.New(skills.Options{
			Run: run, FS: m.sys.FS, Net: m.sys.Net, Home: m.sys.Env.Home(),
			APIURL: m.or("STATION_GITHUB_API_URL", "https://api.github.com"),
			Token:  m.or("GITHUB_TOKEN", m.vars["GH_TOKEN"]),
		}),
		script.New(run, terminal), shell.New(run), app.New(), providedby.New(run),
	} {
		registry[i.Method()] = i
	}
	if m.world != nil {
		// invariant: a world's installers are its fakes, one per method the
		// real registry has, each lined when the real one rides the bundle.
		managers := map[catalog.Method]contract.Manager{}
		for method, real := range registry {
			_, lined := real.(contract.Lined)
			managers[method] = real.Manager()
			registry[method] = m.world.installer(method, lined)
		}
		m.world.managers(managers)
		return registry, m.world.bundler()
	}
	return registry, bundle
}

// invariant: a Then item that asks values or a terminal is skipped off one, as a
// row naming the command that applies it later, never as a failure.
type configurer struct {
	settings *settingscalls.Catalogue
	ask      kit.Asker
}

func (c configurer) Configure(ctx context.Context, item string, interactive bool) (apps.Step, error) {
	found, err := c.settings.Find([]string{item})
	if err != nil {
		return apps.Step{}, err
	}
	it := found[0]
	if c.settings.NeedsInput(it) && !interactive {
		return apps.Step{Item: item, Result: catalog.StateAbsent, Detail: "needs a terminal: station settings apply " + item}, nil
	}
	var values map[string]string
	if len(it.Inputs) > 0 && c.ask != nil {
		if values, err = c.ask.Form(pagesettings.InputsForm(item, it.Inputs, c.settings.Defaults(item))); err != nil {
			return apps.Step{}, err
		}
	}
	change, err := c.settings.Install(ctx, item, values, interactive)
	var root *settings.NeedsRootError
	if errors.As(err, &root) {
		return apps.Step{Item: item, Result: catalog.StateAbsent, Detail: "needs root: " + words.SettingsAsRoot("apply", item)}, nil
	}
	if err != nil {
		return apps.Step{}, err
	}
	detail := ""
	switch n := len(change.Steps); {
	case n == 1:
		detail = change.Steps[0].Label
	case n > 1:
		detail = change.Steps[0].Label + " and " + strconv.Itoa(n-1) + " more"
	}
	return apps.Step{Item: item, Result: catalog.StateOK, Detail: detail}, nil
}

type appsMachine struct{ h local.System }

func (a appsMachine) Now() time.Time { return a.h.Clock.Now() }
func (a appsMachine) Sleep(ctx context.Context, d time.Duration) error {
	return a.h.Clock.Sleep(ctx, d)
}
func (a appsMachine) Home() string { return a.h.Env.Home() }
func (a appsMachine) Glob(pattern string) []string {
	matches, _ := a.h.FS.Glob(pattern)
	return matches
}
func (a appsMachine) LookPath(name string) (string, bool) { return a.h.Run.LookPath(name) }
func (a appsMachine) Exists(p string) bool {
	_, err := a.h.FS.Stat(p)
	return err == nil
}

func (a appsMachine) RunBriefly(ctx context.Context, name string, args []string, budget time.Duration) (output string, code int, ran bool) {
	out, err := a.h.Run.Output(ctx, local.Cmd{Name: name, Args: args, Timeout: budget})
	if err != nil {
		return "", 0, false
	}
	return out.Stdout + out.Stderr, out.ExitCode, true
}

func (a appsMachine) Keep(file, line string) error {
	before, err := a.h.FS.ReadFile(file)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := a.h.FS.MkdirAll(path.Dir(file), 0o700); err != nil {
		return err
	}
	return a.h.FS.WriteFile(file, append(before, line...), 0o600)
}
