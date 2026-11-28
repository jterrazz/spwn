package contract

import (
	"errors"
	"io"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

// value: the folders of a shared source's repository a computer reads
// (ADR-109): its own folder of machines/, the kits and the packages;
// files/ is read by the `file` method, never here.

const (
	RepositoryMachines = "machines"
	RepositoryKits     = "kits"
	RepositoryPackages = "packages"
	ComputerFile       = "host.machine.yaml"
	KitSuffix          = ".kit.yaml"
)

// value: how much of a repository is read at most: the entries listed,
// files and folders alike, every folder of the walk counted, and the
// bytes of one file.

const (
	RepositoryEntries  = 1024
	RepositoryFileSize = 256 << 10
)

// value: a clone's entries, each opened where it stands: a link is never
// followed, and opening one answers an error that is ErrLink. A
// folder opened lists its entries (fs.ReadDirFile), a file reads its
// bytes, and each says what it is by its own Stat, never by its path's.
type clone interface {
	OpenUnfollowed(name string) (fs.File, error)
}

var ErrLink = errors.New("a link, never followed")

// value: a shared source's repository read from a folder: the clone it
// stands in, the folder of the repository its machines are declared in
// (`source.path`, empty for its root), the computer that reads it, by
// its own name, the source's name, and the words its files are named by
// (`studio@1a2b3c4`).

type Repository struct {
	Dir      clone
	At       string
	Sub      string
	Computer string
	Source   string
	Root     string
}

// value: a way into a repository that leaves its clone: a `source.path`
// that is absolute or climbs, or a link anywhere on the way to a file
// read.

type OutsideError struct {
	Source string
	// value: what is refused, as the repository names it, or as the
	// preference says it.
	At      string
	Problem string
}

func (e *OutsideError) Error() string {
	return "the source " + e.Source + " reads " + strconv.Quote(e.At) + " outside its clone: " + e.Problem
}

func (r Repository) Read() ([]machinefile.File, error) {
	// invariant: every file read is a folder entry of the clone listed from
	// its top, never a name joined from the repository's words, and it is
	// judged on the handle it is read from: opened with no link followed,
	// read only when that handle is a regular file of RepositoryFileSize
	// bytes at most, and never past them. Each folder is listed once, from
	// its own handle, and every entry listed counts against one budget,
	// RepositoryEntries, refused past it before the rest is listed. A
	// `source.path` is relative and never climbs. A computer reads its own
	// folder of machines/ alone, the one its own name says. The files come
	// in path order, named as every source names them,
	// `catalog/<category>/<name>.yaml` and `machinefiles/<name>.machine.yaml`,
	// each with its repository path.
	w := &walk{r: r, left: RepositoryEntries}
	at, repo := r.At, ""
	if r.Sub != "" && (!fs.ValidPath(r.Sub) || r.Sub == ".") {
		return nil, &OutsideError{Source: r.Source, At: r.Sub, Problem: "a source's `path` is a folder of it, relative and with no `..`"}
	}
	top, _, err := w.list(at, repo)
	if err != nil {
		return nil, err
	}
	if r.Sub != "" {
		for part := range strings.SplitSeq(r.Sub, "/") {
			next, ok, err := w.folder(top, at, repo, part)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, &UnavailableError{Source: r.Source, Cause: errors.New("its `path` " + strconv.Quote(r.Sub) + " names no folder of it")}
			}
			at, repo, top = next.at, next.repo, next.entries
		}
	}
	for _, read := range []func(top []fs.DirEntry, at, repo string) error{w.machines, w.kits, w.packages} {
		if err := read(top, at, repo); err != nil {
			return nil, err
		}
	}
	slices.SortStableFunc(w.out, func(a, b machinefile.File) int { return strings.Compare(a.Path, b.Path) })
	for i := 1; i < len(w.out); i++ {
		if a, b := w.out[i-1], w.out[i]; a.Path == b.Path {
			name, _, _ := catalog.MachinefileOf(a.Path)
			return nil, w.refused("machinefile", b.Repo, "is named "+name+", as "+r.Root+"/"+a.Repo+" is: a computer's library holds one Machinefile of a name")
		}
	}
	return w.out, nil
}

type walk struct {
	r Repository
	// value: how many more entries the walk may list.
	left int
	out  []machinefile.File
}

// value: a folder of the walk: where it is in the clone, its path in the
// repository, and its entries, listed once.
type listed struct {
	at, repo string
	entries  []fs.DirEntry
}

func (w *walk) refused(format, repo, problem string) error {
	file := w.r.Root
	if repo != "" {
		file += "/" + repo
	}
	return &machinefile.RefusedFileError{Format: format, Err: &catalog.FileError{File: file, Problem: problem}}
}

func (w *walk) link(repo string) error {
	return &OutsideError{Source: w.r.Source, At: repo, Problem: "it is a link: Machines reads a shared source inside its clone, never through a link"}
}

// value: the entry at `at` opened where it stands; a link there is
// refused, and one not there is none.
func (w *walk) open(at, repo string) (fs.File, error) {
	f, err := w.r.Dir.OpenUnfollowed(at)
	if errors.Is(err, ErrLink) {
		return nil, w.link(repo)
	}
	return f, err
}

// value: the entries of the folder at `at`, listed from its own handle,
// in name order, each counted against the walk's budget; ok is false
// when no folder stands there.
func (w *walk) list(at, repo string) (entries []fs.DirEntry, ok bool, err error) {
	f, err := w.open(at, repo)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, false, err
	}
	dir, isDir := f.(fs.ReadDirFile)
	if !info.IsDir() || !isDir {
		return nil, false, nil
	}
	entries, err = dir.ReadDir(w.left + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false, err
	}
	if len(entries) > w.left {
		return nil, false, w.refused("machinefile", repo, "holds more than "+strconv.Itoa(RepositoryEntries)+" entries to read")
	}
	w.left -= len(entries)
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, true, nil
}

// value: the folder named `name` among the entries of `at`, listed, and
// whether it is there; a link of that name is refused, and an entry that
// is no folder is none.
func (w *walk) folder(entries []fs.DirEntry, at, repo, name string) (listed, bool, error) {
	i := slices.IndexFunc(entries, func(e fs.DirEntry) bool { return e.Name() == name })
	if i < 0 {
		return listed{}, false, nil
	}
	return w.enter(entries[i], at, repo)
}

// value: the entry e of `at` listed as a folder; a link is refused, and
// an entry that is no folder is none.
func (w *walk) enter(e fs.DirEntry, at, repo string) (listed, bool, error) {
	sub, subRepo := path.Join(at, e.Name()), path.Join(repo, e.Name())
	if e.Type()&fs.ModeSymlink != 0 {
		return listed{}, false, w.link(subRepo)
	}
	if !e.IsDir() {
		return listed{}, false, nil
	}
	held, ok, err := w.list(sub, subRepo)
	return listed{at: sub, repo: subRepo, entries: held}, ok, err
}

// value: the entries of a folder meant to be read, each judged by its
// listing (no link, a regular file) before it is opened, and again on
// its handle when it is (read).
func (w *walk) files(in listed) ([]fs.DirEntry, error) {
	var out []fs.DirEntry
	for _, e := range in.entries {
		if e.IsDir() || !meant(e.Name()) {
			continue
		}
		p := path.Join(in.repo, e.Name())
		switch t := e.Type(); {
		case t&fs.ModeSymlink != 0:
			return nil, w.link(p)
		case !t.IsRegular():
			return nil, w.refused("machinefile", p, "is no regular file: a source holds files and folders alone")
		}
		out = append(out, e)
	}
	return out, nil
}

func (w *walk) read(in listed, e fs.DirEntry, logical string, slot catalog.Slot) error {
	p := path.Join(in.repo, e.Name())
	f, err := w.open(path.Join(in.at, e.Name()), p)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return w.refused("machinefile", p, "is no regular file: a source holds files and folders alone")
	}
	if info.Size() > RepositoryFileSize {
		return w.refused("machinefile", p, "is "+strconv.FormatInt(info.Size(), 10)+" bytes: a source's file is "+strconv.Itoa(RepositoryFileSize)+" at most")
	}
	body, err := io.ReadAll(io.LimitReader(f, RepositoryFileSize+1))
	if err != nil {
		return err
	}
	if len(body) > RepositoryFileSize {
		return w.refused("machinefile", p, "is past "+strconv.Itoa(RepositoryFileSize)+" bytes: a source's file is "+strconv.Itoa(RepositoryFileSize)+" at most")
	}
	w.out = append(w.out, machinefile.File{Path: logical, Body: body, Repo: p, Slot: slot})
	return nil
}

func (w *walk) machines(top []fs.DirEntry, at, repo string) error {
	machines, ok, err := w.folder(top, at, repo, RepositoryMachines)
	if err != nil || !ok || !catalog.IsMachinefileName(w.r.Computer) {
		return err
	}
	mine, ok, err := w.folder(machines.entries, machines.at, machines.repo, w.r.Computer)
	if err != nil || !ok {
		return err
	}
	entries, err := w.files(mine)
	if err != nil {
		return err
	}
	for _, e := range entries {
		vm, named := strings.CutSuffix(e.Name(), catalog.MachinefileSuffix)
		name, slot := vm, catalog.SlotVM
		switch {
		case !named:
			return w.refused("machinefile", path.Join(mine.repo, e.Name()), "is no machine file: a computer's folder holds `"+ComputerFile+"` and <vm>"+catalog.MachinefileSuffix)
		case e.Name() == ComputerFile:
			name, slot = w.r.Computer, catalog.SlotComputer
		}
		if err := w.read(mine, e, "machinefiles/"+name+catalog.MachinefileSuffix, slot); err != nil {
			return err
		}
	}
	return nil
}

func (w *walk) kits(top []fs.DirEntry, at, repo string) error {
	kits, ok, err := w.folder(top, at, repo, RepositoryKits)
	if err != nil || !ok {
		return err
	}
	entries, err := w.files(kits)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name, named := strings.CutSuffix(e.Name(), KitSuffix)
		if !named {
			return w.refused("machinefile", path.Join(kits.repo, e.Name()), "is no kit: a kit is kits/<name>"+KitSuffix)
		}
		if err := w.read(kits, e, "machinefiles/"+name+catalog.MachinefileSuffix, catalog.SlotKit); err != nil {
			return err
		}
	}
	return nil
}

func (w *walk) packages(top []fs.DirEntry, at, repo string) error {
	packages, ok, err := w.folder(top, at, repo, RepositoryPackages)
	if err != nil || !ok {
		return err
	}
	for _, c := range packages.entries {
		category, ok, err := w.enter(c, packages.at, packages.repo)
		if err != nil {
			return err
		}
		if !ok {
			if meant(c.Name()) {
				return w.refused("catalog", path.Join(packages.repo, c.Name()), "is no package file: a package is packages/<category>/<name>.yaml")
			}
			continue
		}
		files, err := w.files(category)
		if err != nil {
			return err
		}
		for _, e := range files {
			if err := w.read(category, e, "catalog/"+c.Name()+"/"+e.Name(), ""); err != nil {
				return err
			}
		}
	}
	return nil
}
