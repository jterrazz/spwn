package calls

import (
	"context"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/setup"

	"github.com/jterrazz/studio-station/apps/engine/disk"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

const (
	stepRelocate = "relocate"
	busyBudget   = 2 * time.Second
)

type Configurer interface {
	Configure(ctx context.Context, pkg string, interactive bool) error
}

func (s *Provisioner) placedApps() []catalog.App {
	var out []catalog.App
	for _, t := range s.o.Apps {
		if slices.Contains(t.LivesOn(), s.o.Kind) {
			out = append(out, t)
		}
	}
	return out
}

func (s *Provisioner) relocate(ctx context.Context, interactive bool) (setup.Step, error) {
	written, notes, err := s.writeAppFiles(s.placedApps())
	if err != nil {
		return failed(stepRelocate, err)
	}
	var moved []string
	if s.onMac() {
		var moveNotes []string
		moved, moveNotes, err = s.moveMisplaced(ctx, interactive)
		notes = append(notes, moveNotes...)
		if err != nil {
			return failed(stepRelocate, err)
		}
	}
	switch {
	case len(notes) > 0:
		return setup.Step{Name: stepRelocate, State: catalog.StateDegraded, Detail: strings.Join(notes, "; "), Changed: len(written)+len(moved) > 0}, nil
	case len(written)+len(moved) == 0:
		return setup.Step{Name: stepRelocate, State: catalog.StateOK, Detail: "nothing to do"}, nil
	}
	parts := slices.Clone(written)
	if len(moved) > 0 {
		parts = append(parts, "moved "+strings.Join(moved, ", "))
	}
	return setup.Step{Name: stepRelocate, State: catalog.StateOK, Detail: strings.Join(parts, ", "), Changed: true}, nil
}

func (s *Provisioner) Relocate(_ context.Context, app string) error {
	// invariant: one placed app's relocation, written as init writes it,
	// never over a value set by hand.
	apps := slices.DeleteFunc(s.placedApps(), func(t catalog.App) bool { return t.Name != app })
	_, _, err := s.writeAppFiles(apps)
	return err
}

func (s *Provisioner) writeAppFiles(apps []catalog.App) (written, notes []string, err error) {
	byFile := map[catalog.AppFile][]catalog.Relocation{}
	var order []catalog.AppFile
	for _, t := range apps {
		for _, r := range t.Disk.Relocate {
			if r.Pending != "" {
				continue
			}
			if _, seen := byFile[r.File]; !seen {
				order = append(order, r.File)
			}
			byFile[r.File] = append(byFile[r.File], r)
		}
	}
	for _, file := range order {
		target := s.o.Files[file]
		if target == "" {
			notes = append(notes, file.Label()+" is off (GOENV=off): Machines writes nothing")
			continue
		}
		current, _, readErr := s.m.ReadFile(target)
		if readErr != nil {
			return written, notes, readErr
		}
		text := string(current)
		changed := false
		if file == catalog.FileBunfig {
			values := map[string]string{}
			for _, r := range byFile[file] {
				values[r.Key] = s.o.Expand.Expand(r.Value)
			}
			updated, did, conflict := catalog.UpsertBunfig(text, values["install.globalDir"], values["install.globalBinDir"], values["install.cache.dir"])
			if conflict != "" {
				notes = append(notes, conflict)
			}
			text, changed = updated, did
		}
		for _, r := range byFile[file] {
			if file == catalog.FileBunfig {
				continue
			}
			value := s.o.Expand.Expand(r.Value)
			upsert := UpsertEnvLine
			if file.Tabled() {
				upsert = UpsertTableKey
			}
			updated, did, hand := upsert(text, r.Key, value)
			if hand != "" {
				notes = append(notes, file.KeyLabel(r.Key)+" set by hand to "+hand+"; Machines leaves it")
				continue
			}
			text, changed = updated, changed || did
		}
		if !changed {
			continue
		}
		if err := s.m.MkdirAll(path.Dir(target), 0o755); err != nil {
			return written, notes, err
		}
		if err := s.m.WriteFile(target, []byte(text), s.m.Mode(target, 0o644)); err != nil {
			return written, notes, err
		}
		written = append(written, file.Label())
	}
	return written, notes, nil
}

func UpsertEnvLine(text, key, value string) (updated string, changed bool, handSet string) {
	if current, ok := disk.LineValue(text, key); ok {
		if current == value {
			return text, false, ""
		}
		return text, false, current
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return text + key + "=" + value + "\n", true, ""
}

func (s *Provisioner) moveMisplaced(ctx context.Context, interactive bool) (moved, notes []string, err error) {
	for _, t := range s.placedApps() {
		for _, st := range t.Disk.Misplaced {
			if !s.pending(st) {
				continue
			}
			if st.Busy != "" && s.busy(ctx, st.Busy) {
				return moved, notes, &setup.BusyError{Process: st.Busy}
			}
			did, conflicts, moveErr := s.moveOne(st)
			notes = append(notes, conflicts...)
			if moveErr != nil {
				return moved, notes, moveErr
			}
			for _, p := range s.drops(st) {
				if err := s.m.RemoveAll(p); err != nil {
					return moved, notes, err
				}
				did = true
			}
			for _, dir := range st.Prune {
				_, _ = s.m.Remove(s.o.Expand.Expand(dir))
			}
			if did {
				moved = append(moved, st.Path)
			}
			for _, pkg := range st.Then {
				if s.o.Configurer == nil {
					continue
				}
				if err := s.o.Configurer.Configure(ctx, pkg, interactive); err != nil {
					return moved, notes, err
				}
			}
		}
	}
	return moved, notes, nil
}

func (s *Provisioner) pending(st catalog.Misplaced) bool {
	for _, m := range st.Move {
		if len(s.m.Glob(s.o.Expand.Expand(m.From))) > 0 {
			return true
		}
	}
	return len(s.drops(st)) > 0
}

func (s *Provisioner) Moves(ctx context.Context) []setup.Move {
	// invariant: what relocate takes out of the home root on this Mac, with
	// its bytes, before anything moves, as relocate then does it: each
	// source whose destination is empty, one row per move of the catalogue,
	// and each file it deletes; a source refused or kept is not in it.
	if !s.onMac() || s.o.Server {
		return nil
	}
	var out []setup.Move
	for _, t := range s.placedApps() {
		for _, st := range t.Disk.Misplaced {
			for _, m := range st.Move {
				from, to := s.o.Expand.Expand(m.From), s.o.Expand.Expand(m.To)
				entries := strings.HasSuffix(m.From, "/*")
				moving, bytes := false, int64(0)
				for _, src := range s.m.Glob(from) {
					dst := to
					if entries {
						dst = path.Join(to, path.Base(src))
					}
					if !s.empty(dst) {
						continue
					}
					moving, bytes = true, bytes+s.size(ctx, src)
				}
				if moving {
					out = append(out, setup.Move{From: s.o.Expand.Display(strings.TrimSuffix(from, "/*")), To: s.o.Expand.Display(to), Bytes: bytes})
				}
			}
			for _, p := range s.drops(st) {
				out = append(out, setup.Move{From: s.o.Expand.Display(p), Bytes: s.size(ctx, p)})
			}
		}
	}
	return out
}

// value: what a misplaced row's free deletes by path that no move of it
// takes, as found now: a file its app writes again where its settings point.
func (s *Provisioner) drops(st catalog.Misplaced) []string {
	sources := make([]string, 0, len(st.Move))
	for _, m := range st.Move {
		sources = append(sources, strings.TrimSuffix(s.o.Expand.Expand(m.From), "/*"))
	}
	moves := func(p string) bool {
		return slices.ContainsFunc(sources, func(src string) bool { return p == src || strings.HasPrefix(p, src+"/") })
	}
	var out []string
	for _, a := range st.Free {
		if len(a.Run) > 0 || !a.PlacedOn(s.o.Kind) {
			continue
		}
		for _, pattern := range a.Delete {
			for _, p := range s.m.Glob(s.o.Expand.Expand(pattern)) {
				if !moves(p) {
					out = append(out, p)
				}
			}
		}
	}
	return out
}

func (s *Provisioner) size(ctx context.Context, p string) int64 {
	if s.o.Size == nil {
		return 0
	}
	return max(s.o.Size(ctx, p), 0)
}

func (s *Provisioner) busy(ctx context.Context, process string) bool {
	_, code, err := s.m.Output(ctx, "pgrep", []string{"-x", process}, busyBudget)
	return err == nil && code == 0
}

// invariant: every move is one rename(2), never a copy: an entry glob moves each
// match into the destination folder and keeps the owner's file where one of the same
// name is already there; a whole folder refuses a destination that holds anything.
func (s *Provisioner) moveOne(st catalog.Misplaced) (did bool, conflicts []string, err error) {
	for _, m := range st.Move {
		from, to := s.o.Expand.Expand(m.From), s.o.Expand.Expand(m.To)
		entries := strings.HasSuffix(m.From, "/*")
		matches := s.m.Glob(from)
		for _, src := range matches {
			dst := to
			if entries {
				dst = path.Join(to, path.Base(src))
			}
			if !s.empty(dst) {
				if entries {
					conflicts = append(conflicts, s.o.Expand.Display(src)+" already in "+s.o.Expand.Display(to))
					continue
				}
				return did, conflicts, &setup.DestinationExistsError{Path: s.o.Expand.Display(dst)}
			}
			if err := s.rename(src, dst); err != nil {
				return did, conflicts, err
			}
			did = true
		}
	}
	return did, conflicts, nil
}

func (s *Provisioner) empty(p string) bool {
	dir, found := s.m.IsDir(p)
	if !found {
		return true
	}
	if !dir {
		return false
	}
	entries, _, err := s.m.Entries(p)
	return err == nil && len(entries) == 0
}

func (s *Provisioner) rename(from, to string) error {
	if err := s.m.MkdirAll(path.Dir(to), 0o755); err != nil {
		return err
	}
	if s.exists(to) {
		if _, err := s.m.Remove(to); err != nil {
			return err
		}
	}
	crossDevice, err := s.m.Rename(from, to)
	if crossDevice {
		return &setup.CrossDeviceError{From: s.o.Expand.Display(from), To: s.o.Expand.Display(to)}
	}
	return err
}
