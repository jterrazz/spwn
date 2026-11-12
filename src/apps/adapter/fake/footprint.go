package fake

import (
	"context"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/world"
)

type Footprint struct {
	// invariant: what a method that keeps no list left on a Mac of the world,
	// read as a real machine shows it: the commands on its search path and the
	// bundles in its Applications, by the catalog's apps whose install of that
	// method the world holds. It answers nothing a listed method installed.
	w    *world.World
	mac  string
	apps []catalog.App
}

func FootprintOf(w *world.World, mac string, c catalog.Catalog) Footprint {
	return Footprint{w: w, mac: mac, apps: c.Apps}
}

func (f Footprint) Command(name string) bool {
	return f.any(func(t catalog.App) bool {
		for _, p := range t.Provides {
			if p == name {
				return true
			}
		}
		return false
	})
}

func (f Footprint) Bundle(name string) bool {
	return f.any(func(t catalog.App) bool { return t.Bundle != "" && t.Bundle == name })
}

func (f Footprint) Path(pattern string) bool {
	return f.any(func(t catalog.App) bool { return t.Version.Path != "" && t.Version.Path == pattern })
}

func (f Footprint) Check(line string) (held bool, version string, commands []string, known bool) {
	// invariant: a check line some install of the catalog names answers as
	// the world holds that install, the copy it put in its own folder: exit
	// 0 and the version, or exit 1; commands are the app's.
	f.w.Lock()
	defer f.w.Unlock()
	for _, t := range f.apps {
		for _, p := range t.Installs {
			if catalog.CheckOf(p.Install) != line {
				continue
			}
			known, commands = true, t.Provides
			if v, there := manager(f.w, f.mac, p.Install.Method()).Installed[p.Install.Key()]; there {
				return true, said(t.Version.Pattern, v), commands, true
			}
		}
	}
	return false, "", commands, known
}

func (f Footprint) Reads(home, arg string) bool {
	return f.any(func(t catalog.App) bool {
		at, _, _ := strings.Cut(strings.Replace(t.Version.Path, "~", home, 1), "*")
		return t.Version.Path != "" && strings.HasPrefix(arg, at)
	})
}

func (f Footprint) VersionOf(home string, c local.Cmd) (string, bool) {
	// invariant: the program an app's version is read by answers what that app's
	// pattern reads the version out of: the version the world holds for it.
	f.w.Lock()
	defer f.w.Unlock()
	run := append([]string{path.Base(c.Name)}, c.Args...)
	for _, t := range f.apps {
		want := slices.Clone(t.Version.Args)
		for i := range want {
			want[i] = strings.Replace(want[i], "~", home, 1)
		}
		if len(want) == 0 || !slices.Equal(want, run) {
			continue
		}
		for _, p := range t.Installs {
			m := p.Install.Method()
			if version, there := manager(f.w, f.mac, m).Installed[p.Install.Key()]; there && !m.KeepsList() {
				return said(t.Version.Pattern, version), true
			}
		}
	}
	return "", false
}

func said(pattern, version string) string {
	if pattern == "" {
		return version
	}
	from, to := strings.Index(pattern, "("), strings.Index(pattern, ")")
	if from < 0 || to < from {
		return ""
	}
	out := strings.TrimPrefix(pattern[:from], "^") + version + pattern[to+1:]
	if m := regexp.MustCompile(pattern).FindStringSubmatch(out); len(m) < 2 || m[1] != version {
		return ""
	}
	return out
}

func (f Footprint) any(of func(catalog.App) bool) bool {
	f.w.Lock()
	defer f.w.Unlock()
	for _, t := range f.apps {
		if !of(t) {
			continue
		}
		for _, p := range t.Installs {
			if !p.Install.Method().KeepsList() {
				if _, there := manager(f.w, f.mac, p.Install.Method()).Installed[p.Install.Key()]; there {
					return true
				}
			}
		}
	}
	return false
}

// value: where the world finds a command a no-list install left.
const seenAt = "/opt/world/bin/"

type Seen struct {
	// invariant: a machine's runner that also finds the commands its
	// Footprint holds, after its own search.
	local.Runner
	Foot Footprint
	Home string
}

func (s Seen) LookPath(name string) (string, bool) {
	if where, ok := s.Runner.LookPath(name); ok {
		return where, true
	}
	if s.Foot.Command(name) {
		return seenAt + name, true
	}
	return "", false
}

func (s Seen) Output(ctx context.Context, c local.Cmd) (local.Output, error) {
	// value: a program the world only finds, or one that reads what a no-list
	// install left, answers every run with exit 0 and no output: no version to
	// read, none to fail.
	if len(c.Args) == 2 && path.Base(c.Name) == "bash" && c.Args[0] == "-c" {
		if held, version, commands, known := s.Foot.Check(c.Args[1]); known {
			if out, ok := s.ahead(ctx, c, commands); ok {
				return out, nil
			}
			if !held {
				return local.Output{ExitCode: 1}, nil
			}
			return local.Output{Stdout: version + "\n"}, nil
		}
	}
	if out, ok := s.Foot.VersionOf(s.Home, c); ok {
		return local.Output{Stdout: out + "\n"}, nil
	}
	if strings.HasPrefix(c.Name, seenAt) || slices.ContainsFunc(c.Args, func(a string) bool { return s.Foot.Reads(s.Home, a) }) {
		return local.Output{}, nil
	}
	return s.Runner.Output(ctx, c)
}

// invariant: a check run on a PATH it does not seal answers by the first
// copy of its command that PATH finds, as bash would: one of the owner's
// ahead of station's (a shim, a copy on PATH) answers for it.
func (s Seen) ahead(ctx context.Context, c local.Cmd, commands []string) (local.Output, bool) {
	words := strings.Fields(c.Args[1])
	if c.Sealed || len(words) == 0 || !slices.Contains(commands, words[0]) {
		return local.Output{}, false
	}
	where, found := s.Runner.LookPath(words[0])
	if !found {
		return local.Output{}, false
	}
	out, err := s.Runner.Output(ctx, local.Cmd{Name: where, Args: words[1:]})
	return out, err == nil
}

func (s Seen) Within(env []string) local.Runner {
	return Seen{Runner: s.Runner.Within(env), Foot: s.Foot, Home: s.Home}
}

type SeenFS struct {
	// invariant: a machine's files that also hold what its Footprint says a
	// no-list install left: the bundle in /Applications, and the path an app's
	// version is read at, which a glob finds as one match.
	local.FS
	Foot Footprint
	Home string
}

func (f SeenFS) Stat(p string) (fs.FileInfo, error) {
	info, err := f.FS.Stat(p)
	if err != nil && path.Base(path.Dir(p)) == "Applications" && strings.HasSuffix(p, ".app") && f.Foot.Bundle(strings.TrimSuffix(path.Base(p), ".app")) {
		return seenInfo{name: path.Base(p)}, nil
	}
	return info, err
}

func (f SeenFS) Glob(pattern string) ([]string, error) {
	found, err := f.FS.Glob(pattern)
	if err == nil && len(found) == 0 && f.Foot.Path(f.unhome(pattern)) {
		return []string{strings.ReplaceAll(pattern, "*", "world")}, nil
	}
	return found, err
}

func (f SeenFS) unhome(p string) string {
	if rest, ok := strings.CutPrefix(p, f.Home+"/"); ok {
		return "~/" + rest
	}
	return p
}

type seenInfo struct{ name string }

func (i seenInfo) Name() string     { return i.name }
func (seenInfo) Size() int64        { return 0 }
func (seenInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o755 }
func (seenInfo) ModTime() time.Time { return time.Time{} }
func (seenInfo) IsDir() bool        { return true }
func (seenInfo) Sys() any           { return nil }
