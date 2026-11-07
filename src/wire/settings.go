package wire

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"

	accountscalls "github.com/jterrazz/studio-station/src/accounts/calls"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/agents/adapter/claudecode"
	"github.com/jterrazz/studio-station/src/agents/adapter/codex"
	agentscontract "github.com/jterrazz/studio-station/src/agents/contract"
	"github.com/jterrazz/studio-station/src/settings/adapter/agenthooks"
	"github.com/jterrazz/studio-station/src/settings/adapter/gitaliases"
	"github.com/jterrazz/studio-station/src/settings/adapter/zshblock"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call/lanes"
	pagesettings "github.com/jterrazz/studio-station/src/pages/settings"
	"github.com/jterrazz/studio-station/src/platform/local"
	settingsdefaults "github.com/jterrazz/studio-station/src/settings/adapter/defaults"
	"github.com/jterrazz/studio-station/src/settings/adapter/dotfile"
	"github.com/jterrazz/studio-station/src/settings/adapter/git"
	"github.com/jterrazz/studio-station/src/settings/adapter/macos"
	"github.com/jterrazz/studio-station/src/settings/adapter/neversleeps"
	"github.com/jterrazz/studio-station/src/settings/adapter/touch"
	"github.com/jterrazz/studio-station/src/settings/adapter/vault"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
	settingscontract "github.com/jterrazz/studio-station/src/settings/contract"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
	settingscmd "github.com/jterrazz/studio-station/src/surface/cli/settings"
	settingsapply "github.com/jterrazz/studio-station/src/surface/cli/settings/apply"
	settingsdiff "github.com/jterrazz/studio-station/src/surface/cli/settings/diff"
	settingsoff "github.com/jterrazz/studio-station/src/surface/cli/settings/off"
)

func settingsCommand(env kit.Env, m machine) *cobra.Command {
	all := func() []string {
		out := make([]string, 0, len(catalog.Settings))
		for _, it := range catalog.Settings {
			out = append(out, it.Name)
		}
		return out
	}
	children := func() []*cobra.Command {
		return []*cobra.Command{
			settingsdiff.Command(settingsdiff.Deps{Names: all, ThisMachine: m.where}, env),
			settingsapply.Command(settingsapply.Deps{Names: all, ThisMachine: m.where, Now: m.sys.Clock.Now}, env),
			settingsoff.Command(settingsoff.Deps{Names: all, ThisMachine: m.where, Also: func(names []string) []string { return m.pauses(env, names) }}, env),
		}
	}
	return settingscmd.Command(settingscmd.Deps{Names: all, ThisMachine: m.where, Now: m.sys.Clock.Now, Label: m.label}, env, children()...)
}

// value: station config, config install and config uninstall, the words v2
// gave settings, settings apply and settings off.
func settingsAliases(env kit.Env, m machine) []*cobra.Command {
	all := func() []string { return nil }
	return []*cobra.Command{kit.Alias("config", settingscmd.Command(settingscmd.Deps{Names: all, ThisMachine: m.where, Now: m.sys.Clock.Now, Label: m.label}, env,
		kit.Alias("install", settingsapply.Command(settingsapply.Deps{Names: all, ThisMachine: m.where, Now: m.sys.Clock.Now}, env), "apply"),
		kit.Alias("uninstall", settingsoff.Command(settingsoff.Deps{Names: all, ThisMachine: m.where}, env), "off"),
	), "settings")}
}

// value: an item's values asked in a form at the owner's terminal.
type settingsAsk struct {
	settings *settingscalls.Catalogue
	ask      kit.Asker
}

func (a settingsAsk) Values(it catalog.Setting) (map[string]string, error) {
	return a.ask.Form(pagesettings.InputsForm(it.Name, it.Inputs, a.settings.Defaults(it.Name)))
}

func settingsOf(m machine) *settingscalls.Catalogue {
	return settingscalls.NewCatalogue(catalog.Settings, settingscalls.CatalogueOptions{
		ThisMachine: m.kind,
		Admin:       m.sys.Env.Euid() == 0,
		Has:         m.has,
		Defaults:    settingsDefaults(m),
		Appliers:    settingAppliers(m),
	})
}

// value: the Applier of every effect of the catalogue on this machine; in
// a world, every procedure is a fake over the Mac's slice: no world runs a
// procedure of the real machine.
func settingAppliers(m machine) []settingscontract.Applier {
	// invariant: "dotfiles" is the embedded root, so Sub cannot fail.
	files, _ := fs.Sub(catalog.DotfilesFS, "dotfiles")
	appliers := []settingscontract.Applier{
		touch.New(m.sys.FS, m.sys.Env.Home()),
		gitaliases.New(m.sys.FS, m.sys.Env.Home()),
		zshblock.New(m.sys.FS, zshblock.Options{Dir: m.zshDir(), Ours: files}),
		dotfile.New(m.sys.FS, m.sys.Run, dotfile.Options{Home: m.sys.Env.Home(), Ours: files}),
		neversleeps.New(m.stores.neverSleeps()),
		agenthooks.New(m.has,
			agenthooks.Hook{App: claudecode.Program, File: claudecode.File, Hooks: m.agentHooks(claudecode.ID)},
			agenthooks.Hook{App: codex.Program, File: codex.File, Hooks: m.agentHooks(codex.ID)},
		),
	}
	if m.world != nil {
		return append(appliers, m.world.procedureAppliers()...)
	}
	mac := macos.NewMac(m.sys.Run, m.sys.FS, macos.MacOptions{
		Home:             m.sys.Env.Home(),
		DNSProfile:       filepath.Join(m.paths.State, "dns", "quad9-dns.mobileconfig"),
		BackupExclusions: []string{m.paths.Roots.Cache, filepath.Join(m.paths.Data, "vms")},
		Jobs:             m.jobs(settingscalls.ReadBudget),
		Identity:         m.stores.identity(),
	})
	appliers = append(appliers, mac.Appliers()...)
	appliers = append(appliers, settingsdefaults.NewDock(m.sys.Run)...)
	return append(appliers,
		vault.NewClaudeToken(m.sys.Run, accountscalls.NewShareList(m.stores.shareList())),
		git.NewSigning(m.sys.Run, m.stores.identity()),
		git.NewLogin(m.sys.Run, m.sharedRead()),
	)
}

// value: the app is installed here: its program is on the PATH.
func (m machine) has(app string) bool {
	_, ok := m.sys.Run.LookPath(app)
	return ok
}

// value: the hook of one app's agents in this VM's own settings file; a
// world's is the fake over its VM.
func (m machine) agentHooks(app agents.App) agentscontract.Hooks {
	if m.world != nil {
		return m.world.hooks(app)
	}
	if app == codex.ID {
		return codex.New(m.sys.FS, m.sys.Env.Home(), m.sys.Clock.Now)
	}
	return claudecode.New(m.sys.FS, m.sys.Env.Home(), m.sys.Clock.Now)
}

// value: a read run once per run of the shared pool, whoever asks it.
func (m machine) sharedRead() git.Read {
	if m.pool == nil {
		return nil
	}
	return func(ctx context.Context, c local.Cmd) (local.Output, error) {
		return lanes.Shared(ctx, m.pool, lanes.Command(c.Name, c.Args), func(ctx context.Context) (local.Output, error) { return m.sys.Run.Output(ctx, c) })
	}
}

func settingsDefaults(m machine) map[string]map[string]string {
	defaults := map[string]map[string]string{}
	for _, it := range catalog.Settings {
		for _, in := range it.Inputs {
			if v := strings.TrimSpace(m.vars[in.DefaultEnv]); in.DefaultEnv != "" && v != "" {
				if defaults[it.Name] == nil {
					defaults[it.Name] = map[string]string{}
				}
				defaults[it.Name][in.Name] = v
			}
		}
	}
	return defaults
}
