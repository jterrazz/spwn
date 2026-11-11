package wire

import (
	"cmp"
	"context"
	"io/fs"
	"path"

	setupcalls "github.com/jterrazz/studio-station/src/setup/calls"

	"github.com/jterrazz/studio-station/src/disk"

	"github.com/spf13/cobra"

	activitylinux "github.com/jterrazz/studio-station/src/activity/adapter/linux"
	activitymacos "github.com/jterrazz/studio-station/src/activity/adapter/macos"
	activitycontract "github.com/jterrazz/studio-station/src/activity/contract"
	"github.com/jterrazz/studio-station/src/apps/adapter/brew"
	"github.com/jterrazz/studio-station/src/catalog"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	diskcalls "github.com/jterrazz/studio-station/src/disk/calls"

	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	overviewcalls "github.com/jterrazz/studio-station/src/overview/calls"
	"github.com/jterrazz/studio-station/src/platform/darwin"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/remote/adapter/unix"
	"github.com/jterrazz/studio-station/src/settings"
	settingscli "github.com/jterrazz/studio-station/src/settings/adapter/cli"
	settingslinux "github.com/jterrazz/studio-station/src/settings/adapter/linux"
	settingsmacos "github.com/jterrazz/studio-station/src/settings/adapter/macos"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
	settingscontract "github.com/jterrazz/studio-station/src/settings/contract"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
	overviewcmd "github.com/jterrazz/studio-station/src/surface/cli/overview"
)

func overviewCommand(env kit.Env, m machine) *cobra.Command {
	return overviewcmd.Command(m.overviewDeps(false), env)
}

// value: the one Overview `overview --on <vm>` and `machines <vm>` both
// print (MA-06): a VM is read from All machines.
func (m machine) overviewDeps(status bool) overviewcmd.Deps {
	return overviewcmd.Deps{ThisMachine: m.where, Status: status, AllMachines: func(ctx context.Context) (model.AllMachines, error) {
		return m.machines(false).AllMachines(ctx, false)
	}}
}

func statusCommand(env kit.Env, m machine) *cobra.Command {
	return kit.Alias("status", overviewcmd.Command(m.overviewDeps(true), env), "overview")
}

func statusDeps(env kit.Env, m machine) overviewcalls.Sections {
	h := m.sys
	d := overviewcalls.Sections{
		Pool:    m.pool,
		Catalog: m.cat,
		Refused: m.refused,
		Clock:   h.Clock,
		ThisMac: overviewcalls.ThisMac{
			Kind: m.kind, Hostname: h.Env.Hostname(), OS: h.Env.GOOS(), Arch: h.Env.GOARCH(), User: h.Env.User(), Shell: h.Env.Shell(),
			Home: h.Env.Home(), Root: m.systemRoot(), Version: Version(), Commit: Commit(),
			Agent: func() bool {
				_, err := h.FS.Stat(catalog.EnginePlist(h.Env.Home()))
				return err == nil
			},
		},
		Name:       m.stores.name(),
		Identity:   m.stores.identity(),
		Owner:      settingscli.New(h.Run, h.FS, settingscli.Options{Home: h.Env.Home(), Pool: m.pool}),
		Tailnet:    m.tailnetStatus(),
		Awake:      statusAwake{s: m.awakeWaker()},
		Apps:       appsOf(env, m),
		Settings:   settingsRead{settingsOf(m)},
		Setup:      setupRead{setupOf(env, m)},
		Disk:       diskRead{m.diskOf(env)},
		Engine:     m.macEngine(),
		Hypervisor: m.hypervisor(),
	}
	meter := m.meter()
	listening := m.listeners("/")
	facts := m.facts()
	if m.kind != catalog.Linux {
		// value: the status page reads listeners, jobs and app updates on a Mac only.
		d.Listeners, d.Jobs, d.Updates = listening, m.jobs(overviewcalls.ReadBudget), brew.New(h.Run)
		d.Servers = hostedServers{d: m.macEngine()}
	}
	d.Facts, d.Meter, d.Battery = facts, meter, m.powerPort
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
	ask := unix.New(guestRelay(s), nil)
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

type settingsRead struct{ *settingscalls.Catalogue }

func (settingsRead) Placed(r settings.Report) (string, catalog.State) { return r.Item.Name, r.State }

type setupRead struct{ s *setupcalls.Provisioner }

func (r setupRead) Check(ctx context.Context) ([]overview.Step, error) {
	report, err := r.s.Check(ctx)
	if err != nil {
		return nil, err
	}
	steps := make([]overview.Step, 0, len(report.Steps))
	for _, s := range report.Steps {
		steps = append(steps, overview.Step{Name: s.Name, State: s.State, Detail: s.Detail})
	}
	return steps, nil
}

type diskRead struct{ s *diskcalls.Surveyor }

func (r diskRead) Watch(ctx context.Context) <-chan overviewcalls.DiskReading[disk.MisplacedReport] {
	out := make(chan overviewcalls.DiskReading[disk.MisplacedReport])
	go func() {
		defer close(out)
		for report := range r.s.Watch(ctx) {
			select {
			case out <- diskOf(report):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

const levelInstalled catalog.Level = "installed"

// invariant: one row per storage row and level it holds bytes at, then its
// installed size, then `other`, the caches no row claims.
func diskOf(report disk.Survey) overviewcalls.DiskReading[disk.MisplacedReport] {
	rows := []overview.Disk{}
	for _, r := range report.Rows {
		for _, level := range []catalog.Level{catalog.LevelCache, catalog.LevelData} {
			if b := r.Bytes[level]; b > 0 {
				rows = append(rows, overview.Disk{Name: r.Entry.Name, Level: level, Bytes: b})
			}
		}
		if r.Install > 0 {
			rows = append(rows, overview.Disk{Name: r.Entry.Name, Level: levelInstalled, Bytes: r.Install})
		}
	}
	if report.Other > 0 {
		rows = append(rows, overview.Disk{Name: "other", Level: catalog.LevelCache, Bytes: report.Other})
	}
	return overviewcalls.DiskReading[disk.MisplacedReport]{Rows: rows, Misplaced: append(make([]disk.MisplacedReport, 0, len(report.Misplaced)), report.Misplaced...)}
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
