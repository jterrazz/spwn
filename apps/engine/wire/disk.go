package wire

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/disk"
	diskdocker "github.com/jterrazz/studio-station/apps/engine/disk/adapter/docker"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"

	"github.com/spf13/cobra"

	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/disk/adapter/apfs"
	"github.com/jterrazz/studio-station/apps/engine/disk/adapter/posix"
	diskcalls "github.com/jterrazz/studio-station/apps/engine/disk/calls"
	diskcontract "github.com/jterrazz/studio-station/apps/engine/disk/contract"
	imagescalls "github.com/jterrazz/studio-station/apps/engine/images/calls"
	diskcmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/disk"
	diskerase "github.com/jterrazz/studio-station/apps/engine/surface/cli/disk/erase"
	diskfree "github.com/jterrazz/studio-station/apps/engine/surface/cli/disk/free"
	diskmove "github.com/jterrazz/studio-station/apps/engine/surface/cli/disk/move"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

func diskCommand(env kit.Env, m machine) *cobra.Command {
	names := m.diskOf(env).Names
	return diskcmd.Command(diskcmd.Deps{Names: names, ThisMachine: m.where}, env,
		diskfree.Command(diskfree.Deps{Names: names, ThisMachine: m.where}, env),
		diskerase.Command(diskerase.Deps{Names: names, ThisMachine: m.where}, env),
		diskmove.Command(diskmove.Deps{ThisMachine: m.where, Home: m.sys.Env.Home()}, env),
	)
}

func (m machine) diskOf(env kit.Env) *diskcalls.Surveyor {
	var station diskcalls.Station
	if m.profile.Kind == coremachine.Computer {
		station = diskStation{m}
	}
	return diskcalls.NewSurveyor(diskMachine{m.sys}, m.cat.Apps, diskcalls.SurveyorOptions{
		Disk:        m.disk(),
		Containers:  diskdocker.New(m.sys.Run),
		Pool:        m.pool,
		ThisMachine: m.kind,
		Expand:      m.expander(),
		Files:       m.appFiles(),
		Inventory:   inventory{apps: appsOf(env, m)},
		Configurer:  setupSettingsWriter{settings: settingsOf(m)},
		Station:     station,
	})
}

func (m machine) disk() diskcontract.DiskUsage {
	if m.world != nil {
		return m.world.disk()
	}
	if m.profile.System.OS == coremachine.Linux {
		return posix.New(m.sys.Run)
	}
	return apfs.New(m.sys.Run, m.systemRoot())
}

func (m machine) expander() disk.Expander {
	return disk.Expander{Home: m.sys.Env.Home(), Roots: m.paths.Roots, Root: m.systemRoot()}
}

func (m machine) appFiles() map[catalog.AppFile]string {
	return catalog.AppFilePaths(m.sys.Env.Home(), m.paths.Roots.Prefs, m.kind, m.vars)
}

// value: the disks this Mac's hypervisor holds for station, told apart by
// the machine records and the images' versions; a Mac without the
// hypervisor holds none.
type diskStation struct{ m machine }

func (s diskStation) Held(ctx context.Context) (disk.Held, error) {
	h := s.m.hypervisor()
	if !h.Installed() {
		return disk.Held{}, nil
	}
	disks, err := h.Disks(ctx)
	if err != nil {
		return disk.Held{}, err
	}
	sizes := make(map[string]int64, len(disks))
	for _, d := range disks {
		sizes[d.VM] = d.Bytes
	}
	records, err := s.m.stores.machines.ListReadable(ctx)
	if err != nil {
		return disk.Held{}, err
	}
	reports, err := s.m.images().List(ctx)
	if err != nil {
		return disk.Held{}, err
	}
	var held disk.Held
	for _, r := range records {
		if b, ok := sizes[r.Name]; ok {
			held.Machines = append(held.Machines, disk.MachineDisk{Name: r.Name, Bytes: b})
		}
	}
	for _, r := range reports {
		name := imagescalls.DiskName(r.Record.Name, r.Record.Version)
		b, ok := sizes[name]
		if !ok {
			continue
		}
		held.Images = append(held.Images, disk.MachineDisk{Name: name, Bytes: b})
		if r.Unused() {
			held.Unused += b
		}
	}
	return held, nil
}

type inventory struct{ apps *appscalls.Shelf }

func (i inventory) Installed(ctx context.Context, each func(map[string]bool)) {
	for reports := range i.apps.Watch(ctx) {
		out := make(map[string]bool, len(reports))
		for _, r := range reports {
			if !r.Pending {
				out[r.App.Name] = r.State == catalog.StateOK || r.State == catalog.StateDegraded
			}
		}
		each(out)
	}
}

type diskMachine struct{ h local.System }

func (d diskMachine) Now() time.Time { return d.h.Clock.Now() }
func (d diskMachine) Sleep(ctx context.Context, t time.Duration) error {
	return d.h.Clock.Sleep(ctx, t)
}
func (d diskMachine) Glob(pattern string) []string {
	matches, _ := d.h.FS.Glob(pattern)
	return matches
}
func (d diskMachine) Exists(path string) bool {
	_, err := d.h.FS.Stat(path)
	return err == nil
}
func (d diskMachine) Names(dir string) ([]string, error) {
	entries, err := d.h.FS.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}
func (d diskMachine) ReadFile(path string) ([]byte, error) { return d.h.FS.ReadFile(path) }
func (d diskMachine) Prune(dir string)                     { _ = d.h.FS.Remove(dir) }
func (d diskMachine) Has(name string) bool {
	_, ok := d.h.Run.LookPath(name)
	return ok
}

func (d diskMachine) Run(ctx context.Context, name string, args, env []string) (diskcalls.Ran, error) {
	ran, err := d.h.Run.Attach(ctx, local.Cmd{Name: name, Args: args, Env: env}, local.ToStderr)
	if errors.Is(err, local.ErrNotFound) {
		return diskcalls.Ran{}, nil
	}
	return diskcalls.Ran{Found: true, Code: ran.ExitCode, LastStderr: ran.LastStderr}, err
}

func (d diskMachine) Keep(file string, data []byte) error {
	if err := d.h.FS.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	tmp := file + ".tmp"
	if err := d.h.FS.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return d.h.FS.Rename(tmp, file)
}
