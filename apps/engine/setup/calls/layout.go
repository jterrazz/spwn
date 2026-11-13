package calls

import (
	"bytes"
	"context"
	"errors"
	"path"
	"reflect"
	"slices"
	"strings"

	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"

	"github.com/jterrazz/studio-station/apps/engine/setup"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

const legacyDir = ".studio"

const legacyDNS = "dns"

// invariant: the files of the old ~/.studio that station wrote, by name: the
// layout step moves or removes exactly these and never anything else.
func legacyBin() []string { return []string{"station", "workstation"} }

func (s *Provisioner) legacyPath(parts ...string) string {
	return path.Join(append([]string{s.m.Home(), legacyDir}, parts...)...)
}

func (s *Provisioner) v1Path() string       { return s.legacyPath("config.json") }
func (s *Provisioner) settingsPath() string { return s.o.PreferencesFile }
func (s *Provisioner) logPath() string      { return path.Join(s.o.State, "logs", "setup.log") }

func (s *Provisioner) exists(p string) bool {
	return s.m.Exists(p)
}

// invariant: config.json is removed only after the new settings.json is
// written, so a rerun after any failure finds its source.
func (s *Provisioner) migrateSettings() error {
	data, found, err := s.m.ReadFile(s.v1Path())
	if err != nil || !found {
		return err
	}
	converted, err := setup.ConvertV1(data, s.m.User())
	if err != nil {
		return err
	}
	if s.exists(s.settingsPath()) {
		existing, same, loadErr := s.current()
		if loadErr != nil {
			return loadErr
		}
		if !same || !reflect.DeepEqual(existing, converted) {
			return &setup.DifferentContentError{Path: s.settingsPath()}
		}
	}
	if !s.exists(s.settingsPath()) {
		if writeErr := s.write(converted); writeErr != nil {
			return writeErr
		}
	}
	_, err = s.m.Remove(s.v1Path())
	return err
}

// invariant: same is false when the preferences hold a group a v1
// config.json never converts to (the share list, the services' policy).
func (s *Provisioner) current() (v setup.V1Preferences, same bool, err error) {
	if v.ThisMachine, err = s.p.Name.Get(); err != nil {
		return v, false, err
	}
	if v.Macs, err = s.p.Macs.Get(); err != nil {
		return v, false, err
	}
	if v.VMUser, err = s.p.VMUser.Get(); err != nil {
		return v, false, err
	}
	id, err := s.p.Identity.Get()
	if err != nil {
		return v, false, err
	}
	v.UserEmail, v.UserName = id.UserEmail, id.UserName
	list, err := s.p.ShareList.Get()
	if err != nil {
		return v, false, err
	}
	policy, err := s.p.Ports.Get()
	if err != nil {
		return v, false, err
	}
	return v, len(list) == 0 && policy == nil, nil
}

func (s *Provisioner) write(v setup.V1Preferences) error {
	if err := s.p.Name.Set(v.ThisMachine); err != nil {
		return err
	}
	if err := machinescalls.AdoptV1(s.p.Macs, s.p.VMUser, v.Macs, v.VMUser); err != nil {
		return err
	}
	return s.p.Identity.Set(setup.Identity{UserEmail: v.UserEmail, UserName: v.UserName})
}

func (s *Provisioner) layout(ctx context.Context, migrated error) ([]setup.Step, error) {
	if migrated != nil {
		step, err := failed(stepLayout, migrated)
		return []setup.Step{step}, err
	}
	retired, background, retireErr := s.retire(ctx)
	if retireErr != nil {
		step, err := failed(stepLayout, retireErr)
		return []setup.Step{step}, err
	}
	if background {
		return retired, nil
	}
	if !s.exists(s.legacyPath()) {
		if len(retired) > 0 {
			return retired, nil
		}
		return []setup.Step{{Name: stepLayout, State: catalog.StateOK, Detail: "nothing to retire"}}, nil
	}
	moved, moveErr := s.moveDNS()
	if moveErr != nil {
		step, err := failed(stepLayout, moveErr)
		return []setup.Step{step}, err
	}
	removed, leftover := s.tidy()
	step := setup.Step{Name: stepLayout, State: catalog.StateOK, Detail: "nothing to move", Changed: moved || removed}
	if step.Changed {
		step.Detail = "moved to the XDG folders"
	}
	if leftover {
		step.Detail += " (~/.studio keeps files that are not station's)"
	}
	return append([]setup.Step{step}, retired...), nil
}

// invariant: the folder station's zsh block once read its functions from
// goes the way ~/.studio did. Each file station wrote there, by its
// setting's name, moves into ~/.zshrc as that setting's block, its bytes as
// they are (an edit of the owner's included), so every shell keeps its
// functions and the setting's record still matches what is there; a file
// whose block ~/.zshrc holds already with exactly its bytes is what an
// earlier move could not delete, and goes now; one whose block holds other
// text, or whose bytes hold a marker line, stays where it is. A file of the
// owner's in the folder stays, and so does the folder; it goes once it is
// empty.
func retiredZsh() []string { return []string{"docker-shortcuts"} }

func (s *Provisioner) retireZshDir(ctx context.Context) ([]setup.Step, error) {
	dir := path.Join(path.Dir(s.o.PreferencesFile), "zsh")
	entries, found, err := s.m.Entries(dir)
	switch {
	case err != nil:
		return failedSteps(stepZsh, err)
	case !found:
		return nil, nil
	}
	var moved, leftover []string
	err = s.o.Zshrc.Edit(ctx, func(zshrc string) (string, error) {
		moved, leftover = nil, nil
		for _, e := range entries {
			name, ok := strings.CutSuffix(e.Name, ".zsh")
			if e.Dir || !ok || !slices.Contains(retiredZsh(), name) {
				continue
			}
			file := path.Join(dir, e.Name)
			data, found, readErr := s.m.ReadFile(file)
			id := "station:" + name
			body := string(data)
			if !strings.HasSuffix(body, "\n") {
				body += "\n"
			}
			have, state := settings.PackageBlock(zshrc, id)
			switch {
			case readErr != nil || !found || holdsMarker(body):
			case state == settings.BlockHere && have == body:
				leftover = append(leftover, file)
			case state == settings.BlockAbsent:
				zshrc, _ = settings.UpsertPackageBlock(zshrc, id, body)
				moved = append(moved, file)
			}
		}
		return zshrc, nil
	})
	if err != nil {
		return failedSteps(stepZsh, err)
	}
	dropped := false
	for _, file := range append(moved, leftover...) {
		dropped = s.drop(file) || dropped
	}
	if left, _, err := s.m.Entries(dir); err == nil && len(left) == 0 {
		dropped = s.drop(dir) || dropped
	}
	if !dropped {
		return nil, nil
	}
	return []setup.Step{{Name: stepZsh, State: catalog.StateOK, Detail: "retired ~/.config/station/zsh: shell blocks live in ~/.zshrc now", Changed: true}}, nil
}

func holdsMarker(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "# >>> station") || strings.HasPrefix(t, "# <<< station") {
			return true
		}
	}
	return false
}

func failedSteps(name string, err error) ([]setup.Step, error) {
	step, err := failed(name, err)
	return []setup.Step{step}, err
}

func (s *Provisioner) moveDNS() (bool, error) {
	entries, found, err := s.m.Entries(s.legacyPath(legacyDNS))
	if err != nil || !found {
		return false, err
	}
	moved := false
	for _, e := range entries {
		if e.Dir {
			continue
		}
		if err := s.move(s.legacyPath(legacyDNS, e.Name), path.Join(s.o.State, "dns", e.Name)); err != nil {
			return moved, err
		}
		moved = true
	}
	return moved, nil
}

// hazard: a run over the tailnet loses its session while the retired node stops, so
// the retirement runs in a process of its own, logging to setup.log.
func (s *Provisioner) detach() error {
	if err := s.m.MkdirAll(path.Dir(s.logPath()), 0o700); err != nil {
		return err
	}
	return s.m.Start(s.o.Executable, []string{"init"}, []string{"STATION_XDG_DETACHED=1"}, s.logPath())
}

// invariant: a destination holding the same bytes is a move already done, one
// holding other bytes is refused with both files untouched, and the source goes
// only after the copy is read back identical (a node state file IS the node's identity).
func (s *Provisioner) move(src, dst string) error {
	data, found, err := s.m.ReadFile(src)
	if err != nil {
		return err
	}
	if !found {
		return errors.New(src + " is gone")
	}
	if existing, there, readErr := s.m.ReadFile(dst); readErr == nil && there {
		if !bytes.Equal(existing, data) {
			return &setup.DifferentContentError{Path: dst}
		}
		_, err = s.m.Remove(src)
		return err
	}
	if copyErr := s.copyVerified(src, dst, data); copyErr != nil {
		return copyErr
	}
	_, err = s.m.Remove(src)
	return err
}

func (s *Provisioner) copyVerified(src, dst string, data []byte) error {
	perm := s.m.Mode(src, 0o600)
	if err := s.m.MkdirAll(path.Dir(dst), 0o700); err != nil {
		return err
	}
	if err := s.m.WriteFile(dst, data, perm); err != nil {
		return err
	}
	copied, _, err := s.m.ReadFile(dst)
	if err != nil {
		return err
	}
	if !bytes.Equal(copied, data) {
		return &setup.DifferentContentError{Path: dst}
	}
	return nil
}

func (s *Provisioner) tidy() (removed, leftover bool) {
	for _, name := range legacyBin() {
		removed = s.drop(s.legacyPath("bin", name)) || removed
	}
	for _, dir := range []string{catalog.RetiredLegacyDir, legacyDNS, "bin"} {
		removed = s.drop(s.legacyPath(dir)) || removed
	}
	removed = s.drop(s.legacyPath()) || removed
	return removed, s.exists(s.legacyPath())
}

func (s *Provisioner) drop(p string) bool {
	removed, err := s.m.Remove(p)
	return removed && err == nil
}

func (s *Provisioner) holdsStationFiles() bool {
	if s.exists(s.v1Path()) {
		return true
	}
	for _, rel := range [][]string{legacyNodeState(), {"bin", legacyBin()[0]}, {"bin", legacyBin()[1]}} {
		if s.exists(s.legacyPath(rel...)) {
			return true
		}
	}
	entries, _, err := s.m.Entries(s.legacyPath(legacyDNS))
	return err == nil && len(entries) > 0
}
