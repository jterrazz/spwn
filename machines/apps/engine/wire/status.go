package wire

import (
	"cmp"
	"context"
	"io/fs"
	"path"

	"github.com/spf13/cobra"

	activitylinux "github.com/jterrazz/studio-station/apps/engine/activity/adapter/linux"
	activitymacos "github.com/jterrazz/studio-station/apps/engine/activity/adapter/macos"
	activitycontract "github.com/jterrazz/studio-station/apps/engine/activity/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	overviewcalls "github.com/jterrazz/studio-station/apps/engine/overview/calls"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/unix"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	settingslinux "github.com/jterrazz/studio-station/apps/engine/settings/adapter/linux"
	settingsmacos "github.com/jterrazz/studio-station/apps/engine/settings/adapter/macos"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	settingscontract "github.com/jterrazz/studio-station/apps/engine/settings/contract"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	overviewcmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/overview"
)

func overviewCommand(env kit.Env, m machine) *cobra.Command {
	return overviewcmd.Command(m.overviewDeps(), env)
}

// value: the one Overview `overview --on <vm>` and `machines <vm>` both
// print (MA-06): a VM is read from All machines.
func (m machine) overviewDeps() overviewcmd.Deps {
	return overviewcmd.Deps{ThisMachine: m.where, AllMachines: func(ctx context.Context) (model.AllMachines, error) {
		return m.machines(false).AllMachines(ctx, false)
	}}
}

func statusDeps(m machine) overviewcalls.Sections {
	h := m.sys
	d := overviewcalls.Sections{
		Pool:    m.pool,
		Catalog: m.cat,
		Refused: m.refused,
		Clock:   h.Clock,
		ThisMac: overviewcalls.ThisMac{
			Kind: m.kind, Hostname: h.Env.Hostname(), OS: h.Env.GOOS(), Arch: h.Env.GOARCH(), User: h.Env.User(), Shell: h.Env.Shell(),
			Home: h.Env.Home(), Root: m.systemRoot(), Version: Version(), Commit: Commit(),
			Agent: m.engineInstalled,
		},
		Name:     m.stores.name(),
		Settings: settingsRead{settingsOf(m), m.stores.applied},
		Facts:    m.facts(),
		Meter:    m.meter(),
	}
	if m.kind != catalog.Linux {
		d.Servers = hostedServers{d: m.macEngine()}
	}
	return d
}

// value: the Meter of this machine's kernel, or of its world.
func (m machine) meter() activitycontract.Meter {
	switch {
	case m.world != nil:
		return m.world.meter()
	case m.kind == catalog.Linux:
		return activitylinux.New(rootFS{fs: m.sys.FS, root: cmp.Or(m.systemRoot(), "/")}, m.sys.Clock)
	}
	return activitymacos.New(darwin.NewTables(m.vars), m.sys.Clock)
}

// value: the Inspector of this machine's facts: its system's own tools, or
// its world's.
func (m machine) facts() settingscontract.Inspector {
	switch {
	case m.world != nil:
		return m.world.inspector()
	case m.kind == catalog.Linux:
		return settingslinux.New(m.sys.Run)
	}
	return settingsmacos.New(m.sys.Run, m.sys.FS, m.sys.Clock, settingsmacos.Options{Root: m.systemRoot(), Jobs: m.jobs(settingscalls.ReadBudget)})
}

type hostedServers struct{ d macEngine }

func (s hostedServers) Hosted(ctx context.Context) bool {
	live, err := s.d.VMs(ctx)
	if err != nil {
		return false
	}
	ask := unix.New(guestRelay(s), nil, nil, nil)
	for name, vm := range live {
		if vm.State != engine.StateRunning {
			continue
		}
		if kind, err := ask.KindOf(ctx, name); err == nil && kind == coremachine.Server {
			return true
		}
	}
	return false
}

type settingsRead struct {
	*settingscalls.Catalogue
	applied settingscontract.AppliedStore
}

func (settingsRead) Placed(r settings.Report) (string, catalog.State) { return r.Package.Name, r.State }

func (s settingsRead) Recorded(ctx context.Context, setting string) bool {
	if s.applied == nil {
		return false
	}
	_, err := s.applied.Get(ctx, setting)
	return err == nil
}

// value: the machine's files under its root, as the Linux Meter reads /proc.
type rootFS struct {
	fs   local.FS
	root string
}

func (r rootFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
}

func (r rootFS) ReadFile(name string) ([]byte, error) { return r.fs.ReadFile(path.Join(r.root, name)) }

func (r rootFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return r.fs.ReadDir(path.Join(r.root, name))
}
