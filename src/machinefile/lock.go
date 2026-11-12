package machinefile

import (
	"cmp"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
)

// value: the lock a resolution reads: what the machine's last apply or
// upgrade left, and the file the owner reads it at.

type Lock struct {
	catalog.LockFile
	File string
}

func (l Lock) App(name string) (catalog.LockedApp, bool) {
	// value: the method and version the lock holds an app by.
	i := slices.IndexFunc(l.Apps, func(a catalog.LockedApp) bool { return a.Name == name })
	if i < 0 {
		return catalog.LockedApp{}, false
	}
	return l.Apps[i], true
}

func (s Step) InstallVersion() string {
	// value: the range apply hands the install, in its one unambiguous
	// word: the lock's version exactly, where the step's method pins one;
	// else the range the Machinefile asks, every version said `latest` to a
	// method that holds any range, so mise installs its newest of every
	// major, never the catalog's narrower channel (node's `lts`).
	if s.Locked != "" {
		return catalog.Exactly(s.Locked).Spec()
	}
	if s.Version.Any() && s.Spec.Holds() == catalog.HoldAny {
		return "latest"
	}
	return s.Version.Spec()
}

func Relocked(p Plan, done []Done, source *catalog.LockSource) catalog.LockFile {
	// invariant: the lock a run leaves names, for every app the Machinefile
	// lists that reads there by a method once the run ended, that method
	// and the version it reads; an app the run did not bring there (it
	// failed, was interrupted, was not run, or does not read) keeps what
	// the lock held of it, so the next run installs it as locked; an app
	// the Machinefile lists no longer leaves the lock, and so does a copy
	// no method holds (not managed), the owner's.
	out := catalog.LockFile{Machinefile: p.Name, Source: source}
	ended := map[string]Done{}
	for _, d := range done {
		ended[d.Line.Step.Item] = d
	}
	for _, l := range p.Lines {
		if l.Step.Kind == catalog.KindSetting {
			continue
		}
		locked, ok := p.Lock.App(l.Step.Item)
		if d, ran := ended[l.Step.Item]; ran && (l.Acts() || d.Updated) {
			if d.State == catalog.StateOK && d.Reason == "" {
				locked, ok = catalog.LockedApp{Name: l.Step.Item, Method: cmp.Or(d.Method, l.Step.Method), Version: d.Version}, true
			}
		} else if (l.Seen.Condition == Present || l.Seen.Condition == Outdated) && l.Seen.Method != "" {
			locked, ok = catalog.LockedApp{Name: l.Step.Item, Method: l.Seen.Method, Version: l.Seen.Version}, true
		} else if l.Seen.Condition == NotManaged {
			ok = false
		}
		if ok {
			out.Apps = append(out.Apps, locked)
		}
	}
	slices.SortFunc(out.Apps, func(a, b catalog.LockedApp) int { return strings.Compare(a.Name, b.Name) })
	return out
}

// value: how one app's line of the lock moved: added (Was empty), gone
// (Now empty), or another version or method.

type LockChange struct {
	App      string
	Was, Now *catalog.LockedApp
}

// value: what moved between two locks: each app whose line changed, by
// name, and the shared source's commit when it moved.

type LockDiff struct {
	Apps   []LockChange
	Source *[2]string
}

func Diff(was, now catalog.LockFile) LockDiff {
	var d LockDiff
	names := map[string]bool{}
	for _, a := range slices.Concat(was.Apps, now.Apps) {
		names[a.Name] = true
	}
	find := func(l catalog.LockFile, name string) *catalog.LockedApp {
		i := slices.IndexFunc(l.Apps, func(a catalog.LockedApp) bool { return a.Name == name })
		if i < 0 {
			return nil
		}
		return &l.Apps[i]
	}
	for _, name := range sortedKeys(names) {
		w, n := find(was, name), find(now, name)
		if w != nil && n != nil && *w == *n {
			continue
		}
		d.Apps = append(d.Apps, LockChange{App: name, Was: w, Now: n})
	}
	if commit(was.Source) != commit(now.Source) {
		d.Source = &[2]string{commit(was.Source), commit(now.Source)}
	}
	return d
}

func Merged(current, was, now catalog.LockFile) catalog.LockFile {
	// invariant: a run moves only what it changed: the lock on disk now
	// takes each app and the source that differ between the lock the run
	// planned from and the one it ends with, and keeps every other line as
	// it stands, so two runs at once never lose each other's installs.
	out := now
	out.Apps = slices.Clone(current.Apps)
	d := Diff(was, now)
	for _, c := range d.Apps {
		out.Apps = slices.DeleteFunc(out.Apps, func(a catalog.LockedApp) bool { return a.Name == c.App })
		if c.Now != nil {
			out.Apps = append(out.Apps, *c.Now)
		}
	}
	slices.SortFunc(out.Apps, func(a, b catalog.LockedApp) int { return strings.Compare(a.Name, b.Name) })
	if d.Source == nil {
		out.Source = current.Source
	}
	return out
}

func commit(s *catalog.LockSource) string {
	if s == nil {
		return ""
	}
	return s.Name + "@" + s.Commit
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func (d LockDiff) Empty() bool { return len(d.Apps) == 0 && d.Source == nil }
