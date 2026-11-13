package wire

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/app"
	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/apt"
	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/brew"
	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/bun"
	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/githubrelease"
	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/mas"
	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/mise"
	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/npm"
	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/providedby"
	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/script"
	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/shell"
	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/skills"
	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/uv"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	pagesettings "github.com/jterrazz/studio-station/apps/engine/pages/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	appscmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/apps"
	appsinstall "github.com/jterrazz/studio-station/apps/engine/surface/cli/apps/install"
	appsmove "github.com/jterrazz/studio-station/apps/engine/surface/cli/apps/move"
	appsremove "github.com/jterrazz/studio-station/apps/engine/surface/cli/apps/remove"
	appsupdate "github.com/jterrazz/studio-station/apps/engine/surface/cli/apps/update"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/words"
)

func appsCommand(env kit.Env, m machine) *cobra.Command {
	names := placedNames(appsOf(env, m))
	return appscmd.Command(appscmd.Deps{Names: names, ThisMachine: m.where, Apps: m.cat.Apps}, env,
		appsinstall.Command(appsinstall.Deps{Names: names, ThisMachine: m.where, Now: m.sys.Clock.Now}, env),
		appsupdate.Command(appsupdate.Deps{ThisMachine: m.where, Now: m.sys.Clock.Now, Managers: m.cat.ManagerKeys()}, env),
		appsremove.Command(appsremove.Deps{Names: names, ThisMachine: m.where, Now: m.sys.Clock.Now}, env),
		appsmove.Command(appsmove.Deps{Names: names, ThisMachine: m.where, Now: m.sys.Clock.Now, Managers: m.cat.ManagerKeys()}, env),
	)
}

// invariant: station tools and its verbs are the hidden v2 words of station
// apps for one release (SPEC §4.3); upgrade's --via is update's --manager.
func appsAlias(env kit.Env, m machine) *cobra.Command {
	names := placedNames(appsOf(env, m))
	update := appsupdate.Via(appsupdate.Command(appsupdate.Deps{ThisMachine: m.where, Now: m.sys.Clock.Now, Managers: m.cat.ManagerKeys()}, env))
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
	return appscalls.NewShelf(appsMachineOf(m), m.cat, configurer{settings: settingsOf(m), ask: ask}, appscalls.ShelfOptions{
		Pool:         m.pool,
		ThisMachine:  m.kind,
		Root:         m.systemRoot(),
		Installers:   registry,
		Bundler:      bundler,
		VMs:          m.machines(env.Interactive),
		Logs:         filepath.Join(m.paths.State, "logs"),
		Kept:         mise.Kept(m.sys.Env.Home(), m.vars),
		HeldByLedger: ledgerOf{m.stores.ledger}.Recorded,
	})
}

// value: the ledger as the apps family reads it: whether it holds an
// install of an app by a method.
type ledgerOf struct{ l ledgerStore }

func (l ledgerOf) Recorded(ctx context.Context, name string, m catalog.Method) (bool, error) {
	if l.l == nil {
		return false, nil
	}
	all, err := l.l.List(ctx)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(all, func(in machinefile.Installed) bool { return in.Package == name && in.Method == string(m) }), nil
}

// invariant: every command station runs for apps, a manager's read, install or
// removal and an app's version read, runs from the owner's home in an
// environment station builds, mise's shims first and pinned to its global
// file, never from the folder nor on the PATH station was started with: a
// project there that pins another node would pick which npm lists what node
// holds, while `mise unuse --global node` takes back the global one. Whether
// an item is on this machine another way is no manager's answer: it is
// looked for on the owner's PATH (appsMachine).
func managing(m machine) local.System {
	sys := m.sys
	home := sys.Env.Home()
	sys.Run = local.Anchored(sys.Run, home, mise.Environment(home, m.vars, npm.PrefixVar))
	return sys
}

func installers(m machine, terminal bool) (contract.Installers, contract.Bundler) {
	run := managing(m).Run
	runtimes := mise.New(run)
	bundle := brew.New(run)
	registry := contract.Installers{}
	for _, i := range []contract.Installer{
		bundle.Formulae(), bundle.Casks(),
		mas.New(run), npm.New(run, bundle, runtimes.Where), uv.New(run, bundle),
		bun.New(run), runtimes, apt.New(run, m.sys.FS, filepath.Join(m.paths.Data, "apt"), terminal),
		skills.New(skills.Options{
			Run: run, FS: m.sys.FS, Net: m.sys.Net, Home: m.sys.Env.Home(),
			APIURL: m.or("STATION_GITHUB_API_URL", "https://api.github.com"),
			Token:  m.or("GITHUB_TOKEN", m.vars["GH_TOKEN"]),
		}),
		script.New(run, m.sys.Env.Home(), terminal), shell.New(run), app.New(), providedby.New(run),
		githubrelease.New(githubrelease.Options{
			FS: m.sys.FS, Net: m.sys.Net, Clock: m.sys.Clock,
			APIURL: m.or("STATION_GITHUB_API_URL", "https://api.github.com"),
			Token:  m.or("GITHUB_TOKEN", m.vars["GH_TOKEN"]),
			OS:     m.sys.Env.GOOS(), Arch: m.sys.Env.GOARCH(),
			Bin: m.paths.Bin, Store: filepath.Join(m.paths.Data, "releases"),
		}),
	} {
		registry[i.Method()] = i
	}
	var bundler contract.Bundler = bundle
	if m.world != nil {
		// invariant: a world's installers are its fakes, one per method the
		// real registry has, each lined when the real one rides the bundle.
		managers := map[catalog.Method]contract.Installer{}
		for method, real := range registry {
			_, lined := real.(contract.Lined)
			managers[method] = real
			registry[method] = m.world.installer(method, lined)
		}
		m.world.managers(managers)
		bundler = m.world.bundler()
	}
	registry[catalog.MethodGitHubRelease] = githubrelease.Own(registry[catalog.MethodGitHubRelease], githubrelease.Held{Key: own(m.cat, m.kind)})
	return registry, bundler
}

// value: the key station's own release is held by, the one the catalog's
// github-release manager installs by here: the running binary came from it.
func own(cat catalog.Catalog, kind catalog.MachineKind) string {
	if t, ok := cat.App(string(catalog.MethodGitHubRelease)); ok {
		if in, ok := t.InstallOn(kind); ok {
			return in.Key()
		}
	}
	return ""
}

// invariant: a Then item that asks values or a terminal is skipped off one, as a
// row naming the command that applies it later, never as a failure.
type configurer struct {
	settings *settingscalls.Catalogue
	ask      kit.Asker
}

func (c configurer) Configure(ctx context.Context, pkg string, interactive bool) (apps.Step, error) {
	found, err := c.settings.Find([]string{pkg})
	if err != nil {
		return apps.Step{}, err
	}
	it := found[0]
	if c.settings.NeedsInput(it) && !interactive {
		return apps.Step{Package: pkg, Result: catalog.StateAbsent, Detail: "needs a terminal: station settings apply " + pkg}, nil
	}
	var values map[string]string
	if len(it.Inputs) > 0 && c.ask != nil {
		if values, err = c.ask.Form(pagesettings.InputsForm(pkg, it.Inputs, c.settings.Defaults(pkg))); err != nil {
			return apps.Step{}, err
		}
	}
	change, err := c.settings.Install(ctx, pkg, values, interactive)
	var root *settings.NeedsRootError
	if errors.As(err, &root) {
		return apps.Step{Package: pkg, Result: catalog.StateAbsent, Detail: "needs root: " + words.SettingsAsRoot("apply", pkg)}, nil
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
	return apps.Step{Package: pkg, Result: catalog.StateOK, Detail: detail}, nil
}

func appsMachineOf(m machine) appsMachine {
	return appsMachine{h: m.sys, run: managing(m).Run}
}

// value: h is the machine as station was started on it, whose PATH is
// where an item is looked for (nvm's node is there and on no manager's
// PATH); run is the runner of the commands station runs for apps.
type appsMachine struct {
	h   local.System
	run local.Runner
}

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
	run, c := a.run, local.Cmd{Name: name, Args: args, Timeout: budget}
	if _, ok := run.LookPath(name); !ok {
		// invariant: a command the owner's PATH alone has, an item present
		// another way, is read where it was found, from the owner's home.
		run, c.Dir = a.h.Run, a.h.Env.Home()
	}
	out, err := run.Output(ctx, c)
	if err != nil {
		return "", 0, false
	}
	return out.Stdout + out.Stderr, out.ExitCode, true
}

func (a appsMachine) RunSealed(ctx context.Context, bin, name string, args []string, budget time.Duration) (output string, code int, ran bool) {
	out, err := a.run.Output(ctx, local.Cmd{Name: name, Args: args, Timeout: budget, Env: []string{"PATH=" + bin}, Sealed: true})
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
