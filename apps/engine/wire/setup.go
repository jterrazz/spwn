package wire

import (
	"context"
	"errors"
	"strings"

	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	pagesetup "github.com/jterrazz/studio-station/apps/engine/pages/setup"

	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	accountscontract "github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	"github.com/jterrazz/studio-station/apps/engine/setup"
	setupcalls "github.com/jterrazz/studio-station/apps/engine/setup/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/apply"
	initcmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/init"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

func initCommands(env kit.Env, m machine) []*cobra.Command {
	d := initcmd.Deps{ThisMachine: m.where, Apply: func(ctx context.Context, name string, io kit.IO) error {
		return apply.Run(ctx, apply.Deps{ThisMachine: m.where}, env, apply.Input{Machinefile: name}, io)
	}}
	// value: setup, init's hidden word for one release.
	retired := kit.Alias("setup", kit.Command(initcmd.Leaf(d, env, initcmd.SetupMoved), env), "init")
	return []*cobra.Command{initcmd.Command(d, env), retired}
}

func setupOf(env kit.Env, m machine) *setupcalls.Provisioner {
	allMachines := m.machines(env.Interactive)
	registrar := allMachinesRegistrar{name: m.stores.name(), machines: allMachines}
	return setupcalls.NewProvisioner(setupMachine{m.sys}, setupcalls.ProvisionerPrefs{Name: m.stores.name(), Identity: m.stores.identity(), Macs: m.stores.macs(), VMUser: m.stores.vmUser(), ShareList: m.stores.shareList(), Ports: m.stores.ports(), Stores: m.stores.migrators()}, setupVault{m.vault()}, registrar, setupcalls.ProvisionerOptions{
		Kind:            m.kind,
		Server:          m.profile.Kind == coremachine.Server,
		PreferencesFile: m.stores.prefs.File(), State: m.paths.State,
		// hazard: a spec points STATION_EXECUTABLE at a stub so the detached re-exec is
		// observable; unset, the running binary is the one re-executed.
		Executable:      m.or("STATION_EXECUTABLE", m.sys.Env.Executable()),
		Detached:        m.vars["STATION_XDG_DETACHED"] == "1",
		Apps:            m.cat.Apps,
		Expand:          m.expander(),
		Files:           m.appFiles(),
		Configurer:      setupSettingsWriter{settings: settingsOf(m)},
		MovePreferences: m.stores.prefs.MoveRetired,
		Zshrc:           m.zshrc(),
	})
}

type setupSettingsWriter struct{ settings *settingscalls.Catalogue }

func (c setupSettingsWriter) Configure(ctx context.Context, pkg string, interactive bool) error {
	_, err := c.settings.Install(ctx, pkg, nil, interactive)
	return err
}

type allMachinesRegistrar struct {
	name     setupcontract.NamePrefs
	machines *machinescalls.Machinist
}

func (r allMachinesRegistrar) ThisMac() (string, bool) {
	thisMac, err := r.name.Get()
	return thisMac, err == nil && thisMac != ""
}

func (r allMachinesRegistrar) Register(name string) error { return r.machines.Init(name) }

type setupInit struct{ *setupcalls.Provisioner }

func (s setupInit) Init(ctx context.Context, name string, interactive bool, ask func() (string, error)) (setup.Report, error) {
	return s.Run(ctx, setupcalls.Request{Name: name, Interactive: interactive, AskName: ask})
}

// value: the name asked in a form at the owner's terminal, from the one
// this machine would take.
type setupAsk struct {
	setup *setupcalls.Provisioner
	ask   kit.Asker
}

func (a setupAsk) Name(ctx context.Context) (string, error) {
	values, err := a.ask.Form(pagesetup.NameForm(a.setup.DefaultName(ctx)))
	return strings.TrimSpace(values["name"]), err
}

// value: the keychain as setup's retirement forgets from it: a Mac without
// its command keeps nothing.
type setupVault struct{ v accountscontract.Vault }

func (s setupVault) Forget(ctx context.Context, name string) error {
	err := s.v.Forget(ctx, name)
	if errors.Is(err, local.ErrNotFound) {
		return accounts.ErrNotKept
	}
	return err
}
