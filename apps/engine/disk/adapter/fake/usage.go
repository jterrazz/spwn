package fake

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/world"
)

var ErrFault = errors.New("the world failed this erase")

type Usage struct {
	// invariant: the volume of one Mac of the world and the files on it; a
	// file named ~/… is under home.
	w    *world.World
	mac  string
	home string
}

func UsageOf(w *world.World, mac string) *Usage { return &Usage{w: w, mac: mac} }

func (u *Usage) At(home string) *Usage { return &Usage{w: u.w, mac: u.mac, home: home} }

func NewUsage(files map[string]int64, total, free int64, running ...string) *Usage {
	// value: a disk holding files by their absolute path and length, on a
	// volume of total bytes with free left.
	w := &world.World{ThisMac: "this-mac"}
	w.Mac("this-mac").Disk = &world.Disk{Total: total, Free: free, Files: maps.Clone(files), Running: running}
	return UsageOf(w, "this-mac")
}

func (u *Usage) disk() *world.Disk {
	m := u.w.Mac(u.mac)
	if m.Disk == nil {
		m.Disk = &world.Disk{}
	}
	if m.Disk.Files == nil {
		m.Disk.Files = map[string]int64{}
	}
	return m.Disk
}

func (u *Usage) abs(key string) string {
	if rest, ok := strings.CutPrefix(key, "~/"); ok && u.home != "" {
		return path.Join(u.home, rest)
	}
	return key
}

func (u *Usage) key(abs string) string {
	if rest, ok := strings.CutPrefix(abs, u.home+"/"); ok && u.home != "" {
		return "~/" + rest
	}
	return abs
}

func under(p, root string) bool { return p == root || strings.HasPrefix(p, root+"/") }

func (u *Usage) Sizes(ctx context.Context, roots []string, each func(int, int64)) {
	// invariant: each root is told with the world unlocked, since its caller
	// may wait on a reader that needs the world.
	totals := make([]int64, len(roots))
	u.w.Lock()
	late := u.disk().Late
	for i, root := range roots {
		for p, n := range u.disk().Files {
			if under(u.abs(p), root) {
				totals[i] += n
			}
		}
	}
	u.w.Unlock()
	if late > 0 {
		waited, cancel := context.WithTimeout(ctx, late)
		<-waited.Done()
		cancel()
	}
	for i := range roots {
		if ctx.Err() != nil {
			each(i, -1)
			continue
		}
		each(i, totals[i])
	}
}

func (u *Usage) Volume(context.Context) (total, free int64, err error) {
	u.w.Lock()
	defer u.w.Unlock()
	d := u.disk()
	return d.Total, d.Free, nil
}

func (u *Usage) Move(_ context.Context, from, to string) error {
	u.w.Lock()
	defer u.w.Unlock()
	d := u.disk()
	for p := range d.Files {
		if under(u.abs(p), to) {
			return &fs.PathError{Op: "move", Path: to, Err: fs.ErrExist}
		}
	}
	for p, n := range d.Files {
		if abs := u.abs(p); under(abs, from) {
			delete(d.Files, p)
			d.Files[u.key(to+strings.TrimPrefix(abs, from))] = n
		}
	}
	return nil
}

func (u *Usage) Erase(_ context.Context, paths []string) error {
	u.w.Lock()
	defer u.w.Unlock()
	if u.w.Fails("DiskUsage.Erase") {
		return ErrFault
	}
	u.drop(paths)
	return nil
}

func (u *Usage) Drop(paths []string) {
	// value: an erase the owner's own tool did: never a scripted failure.
	u.w.Lock()
	defer u.w.Unlock()
	u.drop(paths)
}

func (u *Usage) drop(paths []string) {
	d := u.disk()
	for _, root := range paths {
		for p, n := range d.Files {
			if under(u.abs(p), root) {
				delete(d.Files, p)
				d.Free += n
			}
		}
	}
}

func (u *Usage) Running(_ context.Context, process string) bool {
	u.w.Lock()
	defer u.w.Unlock()
	return slices.Contains(u.disk().Running, process)
}

// value: every path the world's files make: each file and each folder
// above it, by absolute path.
func (u *Usage) paths() map[string]bool {
	u.w.Lock()
	defer u.w.Unlock()
	out := map[string]bool{}
	for p := range u.disk().Files {
		for abs := u.abs(p); abs != "/" && abs != "." && abs != ""; abs = path.Dir(abs) {
			out[abs] = true
		}
	}
	return out
}

func (u *Usage) Glob(pattern string) []string {
	var out []string
	for p := range u.paths() {
		if ok, _ := path.Match(pattern, p); ok {
			out = append(out, p)
		}
	}
	slices.Sort(out)
	return out
}

func (u *Usage) Has(p string) bool { return u.paths()[p] }

func (u *Usage) Children(dir string) []string {
	var out []string
	for p := range u.paths() {
		if path.Dir(p) == dir {
			out = append(out, path.Base(p))
		}
	}
	slices.Sort(out)
	return out
}

func (u *Usage) Folder(p string) bool { return len(u.Children(p)) > 0 }
