package wire

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/disk"
	diskdocker "github.com/jterrazz/studio-station/apps/engine/disk/adapter/docker"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"

	"github.com/spf13/cobra"

	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/disk/adapter/apfs"
	"github.com/jterrazz/studio-station/apps/engine/disk/adapter/posix"
	diskcalls "github.com/jterrazz/studio-station/apps/engine/disk/calls"
	diskcontract "github.com/jterrazz/studio-station/apps/engine/disk/contract"
	imagescalls "github.com/jterrazz/studio-station/apps/engine/images/calls"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	machinescontract "github.com/jterrazz/studio-station/apps/engine/machines/contract"
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
// the machine records, the images' versions and the hypervisor's ledger of
// what it cloned, from one listing of its disks; what nothing uses among
// them; a Mac without the hypervisor has none to tell.
type diskStation struct{ m machine }

// value: what a leftover is judged from, read at once: every disk the
// hypervisor lists, the names the machine records hold, and the names
// station's hypervisor cloned.
type leftoverFacts struct {
	disks   []machinescontract.Disk
	named   map[string]bool
	cloned  map[string]bool
	unread  string
	records []model.Record
}

func (s diskStation) facts(ctx context.Context) (leftoverFacts, error) {
	h := s.m.hypervisor()
	disks, err := h.Disks(ctx)
	if err != nil {
		return leftoverFacts{}, err
	}
	f := leftoverFacts{disks: disks, named: map[string]bool{}, cloned: map[string]bool{}}
	f.records, err = s.m.stores.machines.List(ctx)
	var unreadable *store.UnreadableError
	if errors.As(err, &unreadable) {
		f.unread = s.m.expander().Display(unreadable.Path)
		f.records, err = s.m.stores.machines.ListReadable(ctx)
	}
	if err != nil {
		return leftoverFacts{}, err
	}
	for _, r := range f.records {
		f.named[r.Name] = true
	}
	cloned, err := h.Cloned(ctx)
	if err != nil {
		return leftoverFacts{}, err
	}
	for _, name := range cloned {
		f.cloned[name] = true
	}
	return f, nil
}

func (f leftoverFacts) leftover(d machinescontract.Disk) bool {
	// invariant: a leftover is a stopped disk station's hypervisor cloned
	// and has not deleted, that no machine record names and the images
	// family does not own; a disk made or renamed by hand, a test VM, one
	// that runs or whose state is unknown is never one.
	return f.cloned[d.VM] && !f.named[d.VM] && !imagescalls.Owns(d.VM) && !disk.TestVM(d.VM) && d.State == machinescontract.DiskStopped
}

func (s diskStation) Held(ctx context.Context) (*disk.Held, error) {
	h := s.m.hypervisor()
	if !h.Installed() {
		return nil, nil
	}
	f, err := s.facts(ctx)
	if err != nil {
		return nil, err
	}
	sizes := make(map[string]int64, len(f.disks))
	for _, d := range f.disks {
		sizes[d.VM] = d.Bytes
	}
	reports, err := s.m.images().ListOf(ctx, f.disks)
	if err != nil {
		return nil, err
	}
	held := disk.Held{Unread: f.unread}
	for _, r := range f.records {
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
			held.Unused = append(held.Unused, disk.Freeable{Holder: disk.HolderImages, Kind: disk.FreeImage, Name: r.Record.Name + "@" + r.Record.Version, Bytes: b})
		}
	}
	for _, d := range f.disks {
		if f.leftover(d) {
			held.Unused = append(held.Unused, disk.Freeable{Holder: disk.HolderLeftovers, Kind: disk.FreeDisk, Name: d.VM, Bytes: d.Bytes})
		}
	}
	homes, kept := s.oldHomes(ctx, h.Name())
	held.Unused, held.Kept = append(held.Unused, homes...), kept
	if f.unread != "" {
		// invariant: while a record cannot be read, nothing is unused: its
		// disk may be any of them.
		held.Unused = nil
	}
	return &held, nil
}

func (s diskStation) oldHomes(ctx context.Context, hypervisor string) ([]disk.Freeable, []disk.Kept) {
	x := s.m.expander()
	var out []disk.Freeable
	var kept []disk.Kept
	for _, st := range s.oldHomesOf(hypervisor) {
		if vms := s.yours(st); len(vms) > 0 {
			kept = append(kept, disk.KeptOf(st.Path, x.Display(x.Expand(st.Move[0].To))+"/"+filepath.Base(st.Yours[0]), vms))
			continue
		}
		bytes := int64(-1)
		s.m.disk().Sizes(ctx, []string{x.Expand(st.Path)}, func(_ int, b int64) { bytes = b })
		out = append(out, disk.Freeable{Holder: disk.HolderLeftovers, Kind: disk.FreeFolder, Name: st.Path, Bytes: max(bytes, 0)})
	}
	return out, kept
}

func (s diskStation) yours(st catalog.Misplaced) []string {
	x := s.m.expander()
	dirs := make([]string, 0, len(st.Yours))
	for _, dir := range st.Yours {
		dirs = append(dirs, x.Expand(dir))
	}
	return vmsIn(s.m.sys.FS, dirs)
}

func vmsIn(fsys local.FS, dirs []string) []string {
	// invariant: a folder of the owner's that cannot be read holds VMs as
	// far as a free knows, so the old home it is under is kept.
	var vms []string
	for _, dir := range dirs {
		names, err := diskMachine{local.System{FS: fsys}}.Names(dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return []string{filepath.Base(dir) + " (" + err.Error() + ")"}
		}
		vms = append(vms, names...)
	}
	return vms
}

func (s diskStation) oldHomesOf(hypervisor string) []catalog.Misplaced {
	// invariant: an old home is a misplaced folder of the hypervisor's own
	// package that moves whole, once the folder it moves to holds anything:
	// its move is then refused, and the hypervisor never reads it again.
	// While that folder is empty it is the misplaced folder its move
	// carries. One that holds VMs of the owner's is kept, never freed.
	x := s.m.expander()
	var out []catalog.Misplaced
	for _, app := range s.m.cat.Apps {
		if app.Name != hypervisor {
			continue
		}
		for _, st := range app.Disk.Misplaced {
			if len(st.Move) != 1 || st.Move[0].From != st.Path {
				continue
			}
			if _, err := s.m.sys.FS.Stat(x.Expand(st.Path)); err != nil {
				continue
			}
			if held, err := s.m.sys.FS.ReadDir(x.Expand(st.Move[0].To)); err == nil && len(held) > 0 {
				out = append(out, st)
			}
		}
	}
	return out
}

func (s diskStation) Free(ctx context.Context, frees []disk.Freeable) []disk.Outcome {
	// invariant: each one is judged again just before it goes, once every
	// machine record still reads: an image version through the images
	// family's delete, whose in-use rule refuses it there; a disk only while
	// it is still a leftover, read from the records, the hypervisor's
	// listing and its ledger at that moment, then through the hypervisor,
	// which refuses a running one; the old home only while its process does
	// not run and it holds no VM once set aside.
	return freeEach(ctx, frees, s.readable, s.freeOne)
}

func freeEach(ctx context.Context, frees []disk.Freeable, readable func(context.Context) error, one func(context.Context, disk.Freeable) error) []disk.Outcome {
	// invariant: before each one goes, every machine record is read again,
	// and while one cannot be read nothing more goes, of any kind: its disk
	// may be the one, and an image's in-use rule reads only what it can.
	out := make([]disk.Outcome, 0, len(frees))
	for _, f := range frees {
		err := readable(ctx)
		if err == nil {
			err = one(ctx, f)
		}
		o := disk.Outcome{Name: f.Name, Result: catalog.StateOK, Detail: "freed", Freed: f.Bytes}
		if err != nil {
			o = disk.Outcome{Name: f.Name, Result: catalog.StateBroken, Reason: err.Error()}
		}
		out = append(out, o)
	}
	return out
}

func (s diskStation) readable(ctx context.Context) error {
	_, err := s.m.stores.machines.List(ctx)
	var unreadable *store.UnreadableError
	if errors.As(err, &unreadable) {
		return &disk.UnreadRecordError{Path: s.m.expander().Display(unreadable.Path)}
	}
	return err
}

func (s diskStation) freeOne(ctx context.Context, f disk.Freeable) error {
	switch f.Kind {
	case disk.FreeImage:
		image, version, _ := strings.Cut(f.Name, "@")
		_, err := s.m.images().Delete(ctx, image, version)
		return err
	case disk.FreeFolder:
		return s.eraseOldHome(ctx, f.Name)
	case disk.FreeDisk:
		return s.deleteLeftover(ctx, f.Name)
	}
	return fmt.Errorf("%s is of no kind free-unused frees", f.Name)
}

func (s diskStation) deleteLeftover(ctx context.Context, name string) error {
	f, err := s.facts(ctx)
	if err != nil {
		return err
	}
	if f.unread != "" {
		return &disk.UnreadRecordError{Path: f.unread}
	}
	i := slices.IndexFunc(f.disks, func(d machinescontract.Disk) bool { return d.VM == name })
	if i < 0 || !f.leftover(f.disks[i]) {
		return fmt.Errorf("%s is no longer a leftover: kept", name)
	}
	return s.m.hypervisor().Delete(ctx, name)
}

func (s diskStation) eraseOldHome(ctx context.Context, folder string) error {
	for _, st := range s.oldHomesOf(s.m.hypervisor().Name()) {
		if st.Path != folder {
			continue
		}
		if vms := s.yours(st); len(vms) > 0 {
			return fmt.Errorf("%s holds VMs of yours (%s): kept", folder, strings.Join(vms, ", "))
		}
		if st.Busy != "" && s.m.disk().Running(ctx, st.Busy) {
			return &disk.RunningError{Name: s.m.hypervisor().Name(), Process: st.Busy, Verb: "erase", Subject: folder}
		}
		x := s.m.expander()
		yours := make([]string, 0, len(st.Yours))
		for _, dir := range st.Yours {
			yours = append(yours, x.Expand(dir))
		}
		return eraseAside(ctx, s.m.disk(), s.m.sys.FS, x.Expand(folder), yours, x.Display)
	}
	return fmt.Errorf("%s is no old home of the hypervisor", folder)
}

// value: where an old home stands while its free judges it the last time,
// beside it on its volume.
const asideSuffix = ".station-freeing"

func eraseAside(ctx context.Context, d diskcontract.DiskUsage, fsys local.FS, folder string, yours []string, display func(string) string) error {
	// invariant: the folder leaves its path in one rename before anything
	// goes, so nothing lands in it by that path meanwhile; the VMs of the
	// owner's are read again under the new path, and one there (a VM that
	// landed since the last read) brings the folder back whole, never
	// erased. Only a folder that holds none is erased.
	aside := folder + asideSuffix
	if err := d.Move(ctx, folder, aside); err != nil {
		return fmt.Errorf("%s cannot be set aside to be freed (%w): kept", display(folder), err)
	}
	moved := make([]string, 0, len(yours))
	for _, dir := range yours {
		if rel, err := filepath.Rel(folder, dir); err == nil && !strings.HasPrefix(rel, "..") {
			moved = append(moved, filepath.Join(aside, rel))
		}
	}
	if vms := vmsIn(fsys, moved); len(vms) > 0 {
		if err := d.Move(ctx, aside, folder); err != nil {
			return fmt.Errorf("%s holds VMs of yours (%s) and stays at %s: %w", display(folder), strings.Join(vms, ", "), display(aside), err)
		}
		return fmt.Errorf("%s holds VMs of yours (%s): kept", display(folder), strings.Join(vms, ", "))
	}
	return d.Erase(ctx, []string{aside})
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
