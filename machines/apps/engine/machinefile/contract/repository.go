package contract

import (
	"errors"
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

// value: how much of a repository is read at most: the files read, and
// the bytes of one, each judged before it is read.

const (
	RepositoryFiles    = 1024
	RepositoryFileSize = 256 << 10
)

// value: a shared source's repository read from a folder: the clone it
// stands in, the folder of the repository its machines are declared in
// (`source.path`, empty for its root), the computer that reads it, by
// its own name, the source's name, and the words its files are named by
// (`studio@1a2b3c4`).

type Repository struct {
	Dir      folders
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
	// its top, never a name joined from the repository's words: no link is
	// followed, a `source.path` is relative and never climbs, and nothing is
	// read past RepositoryFiles files or a file past RepositoryFileSize
	// bytes. A computer reads its own folder of machines/ alone, the one
	// its own name says. The files come in path order, named as every
	// source names them, `catalog/<category>/<name>.yaml` and
	// `machinefiles/<name>.machine.yaml`, each with its repository path.
	w := &walk{r: r}
	at, repo := r.At, ""
	if r.Sub != "" {
		if !fs.ValidPath(r.Sub) || r.Sub == "." {
			return nil, &OutsideError{Source: r.Source, At: r.Sub, Problem: "a source's `path` is a folder of it, relative and with no `..`"}
		}
		for part := range strings.SplitSeq(r.Sub, "/") {
			next, ok, err := w.folder(at, repo, part)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, &UnavailableError{Source: r.Source, Cause: errors.New("its `path` " + strconv.Quote(r.Sub) + " names no folder of it")}
			}
			at, repo = next, path.Join(repo, part)
		}
	}
	for _, read := range []func(at, repo string) error{w.machines, w.kits, w.packages} {
		if err := read(at, repo); err != nil {
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
	r   Repository
	out []machinefile.File
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

// value: the folder of `at` named `name`, and whether it is there; a link
// of that name is refused, and an entry that is no folder is none.
func (w *walk) folder(at, repo, name string) (dir string, ok bool, err error) {
	entries, err := list(w.r.Dir, at)
	if err != nil {
		return "", false, err
	}
	for _, e := range entries {
		if e.Name() != name {
			continue
		}
		if e.Type()&fs.ModeSymlink != 0 {
			return "", false, w.link(path.Join(repo, name))
		}
		return path.Join(at, name), e.IsDir(), nil
	}
	return "", false, nil
}

// value: the entries of a folder that are meant to be read, each judged
// (no link, a regular file, under the bounds) before it is.
func (w *walk) files(at, repo string) ([]fs.DirEntry, error) {
	entries, err := list(w.r.Dir, at)
	if err != nil {
		return nil, err
	}
	var out []fs.DirEntry
	for _, e := range entries {
		if e.IsDir() || !meant(e.Name()) {
			continue
		}
		p := path.Join(repo, e.Name())
		switch t := e.Type(); {
		case t&fs.ModeSymlink != 0:
			return nil, w.link(p)
		case !t.IsRegular():
			return nil, w.refused("machinefile", p, "is no regular file: a source holds files and folders alone")
		}
		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		if info.Size() > RepositoryFileSize {
			return nil, w.refused("machinefile", p, "is "+strconv.FormatInt(info.Size(), 10)+" bytes: a source's file is "+strconv.Itoa(RepositoryFileSize)+" at most")
		}
		if len(w.out)+len(out) == RepositoryFiles {
			return nil, w.refused("machinefile", w.r.Sub, "holds more than "+strconv.Itoa(RepositoryFiles)+" files to read")
		}
		out = append(out, e)
	}
	return out, nil
}

func (w *walk) read(at, repo string, e fs.DirEntry, logical string, slot catalog.Slot) error {
	body, err := w.r.Dir.ReadFile(path.Join(at, e.Name()))
	if err != nil {
		return err
	}
	w.out = append(w.out, machinefile.File{Path: logical, Body: body, Repo: path.Join(repo, e.Name()), Slot: slot})
	return nil
}

func (w *walk) machines(at, repo string) error {
	machines, ok, err := w.folder(at, repo, RepositoryMachines)
	if err != nil || !ok || !catalog.IsMachinefileName(w.r.Computer) {
		return err
	}
	repo = path.Join(repo, RepositoryMachines)
	mine, ok, err := w.folder(machines, repo, w.r.Computer)
	if err != nil || !ok {
		return err
	}
	repo = path.Join(repo, w.r.Computer)
	entries, err := w.files(mine, repo)
	if err != nil {
		return err
	}
	for _, e := range entries {
		vm, named := strings.CutSuffix(e.Name(), catalog.MachinefileSuffix)
		name, slot := vm, catalog.SlotVM
		switch {
		case !named:
			return w.refused("machinefile", path.Join(repo, e.Name()), "is no machine file: a computer's folder holds `"+ComputerFile+"` and <vm>"+catalog.MachinefileSuffix)
		case e.Name() == ComputerFile:
			name, slot = w.r.Computer, catalog.SlotComputer
		}
		if err := w.read(mine, repo, e, "machinefiles/"+name+catalog.MachinefileSuffix, slot); err != nil {
			return err
		}
	}
	return nil
}

func (w *walk) kits(at, repo string) error {
	kits, ok, err := w.folder(at, repo, RepositoryKits)
	if err != nil || !ok {
		return err
	}
	repo = path.Join(repo, RepositoryKits)
	entries, err := w.files(kits, repo)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name, named := strings.CutSuffix(e.Name(), KitSuffix)
		if !named {
			return w.refused("machinefile", path.Join(repo, e.Name()), "is no kit: a kit is kits/<name>"+KitSuffix)
		}
		if err := w.read(kits, repo, e, "machinefiles/"+name+catalog.MachinefileSuffix, catalog.SlotKit); err != nil {
			return err
		}
	}
	return nil
}

func (w *walk) packages(at, repo string) error {
	packages, ok, err := w.folder(at, repo, RepositoryPackages)
	if err != nil || !ok {
		return err
	}
	repo = path.Join(repo, RepositoryPackages)
	entries, err := list(w.r.Dir, packages)
	if err != nil {
		return err
	}
	for _, c := range entries {
		category, ok, err := w.folder(packages, repo, c.Name())
		if err != nil {
			return err
		}
		if !ok {
			if meant(c.Name()) {
				return w.refused("catalog", path.Join(repo, c.Name()), "is no package file: a package is packages/<category>/<name>.yaml")
			}
			continue
		}
		in := path.Join(repo, c.Name())
		files, err := w.files(category, in)
		if err != nil {
			return err
		}
		for _, e := range files {
			if err := w.read(category, in, e, "catalog/"+c.Name()+"/"+e.Name(), ""); err != nil {
				return err
			}
		}
	}
	return nil
}
