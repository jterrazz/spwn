package wire

import (
	"context"
	"strings"

	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	pagesetup "github.com/jterrazz/studio-station/apps/engine/pages/setup"

	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"

	"github.com/spf13/cobra"

	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	"github.com/jterrazz/studio-station/apps/engine/setup"
	setupcalls "github.com/jterrazz/studio-station/apps/engine/setup/calls"
	initcmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/init"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/machinefile/apply"
)

func initCommands(env kit.Env, m machine) []*cobra.Command {
	d := initcmd.Deps{ThisMachine: m.where, Resolve: func(ctx context.Context, name string) error {
		return apply.Resolve(ctx, env, apply.Input{Machinefile: name})
	}, Apply: func(ctx context.Context, name string, io kit.IO) error {
		return apply.Run(ctx, apply.Deps{ThisMachine: m.where}, env, apply.Input{Machinefile: name}, io)
	}}
	return []*cobra.Command{initcmd.Command(d, env)}
}

func setupOf(env kit.Env, m machine) *setupcalls.Provisioner {
	allMachines := m.machines(env.Interactive)
	registrar := allMachinesRegistrar{name: m.stores.name(), machines: allMachines}
	return setupcalls.NewProvisioner(setupMachine{m.sys}, setupcalls.ProvisionerPrefs{Name: m.stores.name()}, registrar, setupcalls.ProvisionerOptions{
		Kind:            m.kind,
		Server:          m.profile.Kind == coremachine.Server,
		PreferencesFile: m.stores.prefs.File(), State: m.paths.State,
		Apps:       m.cat.Apps,
		Expand:     m.expander(),
		Files:      m.appFiles(),
		Configurer: setupSettingsWriter{settings: settingsOf(m)},
		Size: func(ctx context.Context, p string) int64 {
			bytes := int64(-1)
			m.disk().Sizes(ctx, []string{p}, func(_ int, b int64) { bytes = b })
			return bytes
		},
		Zshrc: m.zshrc(),
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

func (r allMachinesRegistrar) Register(name string) (string, error) { return r.machines.Init(name) }

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
