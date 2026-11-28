package fake

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Source struct {
	w   *world.World
	mac string
}

func UserOf(w *world.World, mac string) Source {
	// value: the owner's files of a Mac: its station files under catalog/ and
	// machinefiles/, as `world edit` gives them.
	return Source{w: w, mac: mac}
}

func (s Source) Read(context.Context) (machinefile.Tree, error) {
	s.w.Lock()
	defer s.w.Unlock()
	return tree(machinefile.User, "~/.config/station", s.w.Mac(s.mac).Files), nil
}

type Shared struct {
	w *world.World
	// value: the Mac whose clone of the shared source is read, and the
	// computer that reads its repository, by its own name.
	mac string
	src contract.SharedSource
}

func SharedOf(w *world.World, mac string, src contract.SharedSource) Shared {
	// value: the shared source over the world's repository at its URL, read
	// from the Mac's clone of it as the real one is (contract.Repository),
	// cloned at the repository's head when the
	// Mac has none; a scripted `Source.Pull` is a fetch that does not answer.
	return Shared{w: w, mac: mac, src: src}
}

var (
	errNoRepository = errors.New("no repository at its url")
	errUnreachable  = errors.New("its url does not answer")
)

func (s Shared) Read(context.Context) (machinefile.Tree, error) {
	s.w.Lock()
	defer s.w.Unlock()
	at, err := s.clone()
	if err != nil {
		return machinefile.Tree{}, err
	}
	root := s.src.Name + "@" + at[:min(7, len(at))]
	files, err := contract.Repository{Dir: held(s.w.Sources[s.src.URL].Commits[at]), At: ".", Sub: s.src.Path, Computer: s.mac, Source: s.src.Name, Root: root}.Read()
	return machinefile.Tree{Kind: machinefile.Shared, Root: root, Files: files, Source: &catalog.LockSource{Name: s.src.Name, Commit: at}}, err
}

// value: a commit's files, by slash path, as the folders a clone holds:
// every file regular, a folder each path's every prefix, `.` the top.

type held map[string]string

func (h held) ReadFile(name string) ([]byte, error) {
	body, ok := h[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(body), nil
}

func (h held) ReadDir(name string) ([]fs.DirEntry, error) {
	prefix := name + "/"
	if name == "." {
		prefix = ""
	}
	seen := map[string]bool{}
	var out []fs.DirEntry
	for p, body := range h {
		rest, ok := strings.CutPrefix(p, prefix)
		if !ok {
			continue
		}
		child, _, deeper := strings.Cut(rest, "/")
		if !seen[child] {
			seen[child] = true
			out = append(out, entry{name: child, dir: deeper, size: int64(len(body))})
		}
	}
	if len(out) == 0 {
		return nil, fs.ErrNotExist
	}
	slices.SortFunc(out, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return out, nil
}

func (h held) Stat(name string) (fs.FileInfo, error) {
	if body, ok := h[name]; ok {
		return entry{name: path.Base(name), size: int64(len(body))}, nil
	}
	if _, err := h.ReadDir(name); err != nil {
		return nil, err
	}
	return entry{name: path.Base(name), dir: true}, nil
}

type entry struct {
	name string
	dir  bool
	size int64
}

func (e entry) Name() string               { return e.name }
func (e entry) IsDir() bool                { return e.dir }
func (e entry) Info() (fs.FileInfo, error) { return e, nil }
func (e entry) Size() int64                { return e.size }
func (e entry) ModTime() time.Time         { return time.Time{} }
func (e entry) Sys() any                   { return nil }
func (e entry) Type() fs.FileMode          { return e.Mode().Type() }
func (e entry) Mode() fs.FileMode {
	if e.dir {
		return fs.ModeDir | 0o755
	}
	return 0o644
}

func (s Shared) Pull(context.Context) (contract.Pulled, error) {
	// invariant: as the real clone, the Mac's clone moves to the
	// repository's head only when the head descends from it.
	s.w.Lock()
	defer s.w.Unlock()
	at, err := s.clone()
	if err != nil {
		return contract.Pulled{}, err
	}
	if s.w.Fails("Source.Pull") {
		return contract.Pulled{Was: at, Now: at}, &contract.UnavailableError{Source: s.src.Name, Cause: errUnreachable}
	}
	repo := s.w.Sources[s.src.URL]
	if !repo.Descends(repo.Head, at) {
		return contract.Pulled{Was: at, Now: at}, &contract.NotForwardError{Source: s.src.Name, At: at, Newest: repo.Head, Clone: "~/.cache/station/sources/" + s.src.Name}
	}
	s.w.Mac(s.mac).Clones[s.src.URL] = repo.Head
	return contract.Pulled{Was: at, Now: repo.Head}, nil
}

func (s Shared) Back(_ context.Context, commit string) error {
	s.w.Lock()
	defer s.w.Unlock()
	if _, err := s.clone(); err != nil {
		return err
	}
	s.w.Mac(s.mac).Clones[s.src.URL] = commit
	return nil
}

func (s Shared) clone() (string, error) {
	repo := s.w.Sources[s.src.URL]
	if repo == nil || repo.Commits[repo.Head] == nil || s.w.Fails("Source.Read") {
		return "", &contract.UnavailableError{Source: s.src.Name, Cause: errNoRepository}
	}
	m := s.w.Mac(s.mac)
	if m.Clones == nil {
		m.Clones = map[string]string{}
	}
	if _, ok := m.Clones[s.src.URL]; !ok {
		m.Clones[s.src.URL] = repo.Head
	}
	return m.Clones[s.src.URL], nil
}

func tree(origin machinefile.OriginKind, root string, files map[string]string) machinefile.Tree {
	t := machinefile.Tree{Kind: origin, Root: root}
	for p, body := range files {
		if strings.HasPrefix(p, "catalog/") || strings.HasPrefix(p, "machinefiles/") {
			t.Files = append(t.Files, machinefile.File{Path: p, Body: []byte(body)})
		}
	}
	slices.SortFunc(t.Files, func(a, b machinefile.File) int { return strings.Compare(a.Path, b.Path) })
	return t
}
