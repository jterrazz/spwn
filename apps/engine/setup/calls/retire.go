package calls

import (
	"context"
	"errors"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/setup"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func legacyNodeState() []string { return []string{catalog.RetiredLegacyDir, catalog.RetiredNodeState} }

func retiredAccounts() []string { return []string{"vm-join-id", "vm-join-secret", "tailnet-authkey"} }

func (s *Provisioner) retiredDirs() []string {
	return []string{path.Join(s.o.State, catalog.RetiredNodeDir), s.legacyPath(catalog.RetiredLegacyDir)}
}

func (s *Provisioner) livePIDs() []int {
	var pids []int
	for _, dir := range s.retiredDirs() {
		for _, name := range []string{catalog.RetiredNodePID, catalog.RetiredAwakePID} {
			data, found, err := s.m.ReadFile(path.Join(dir, name))
			if err != nil || !found {
				continue
			}
			if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && pid > 0 && s.m.Alive(pid) {
				pids = append(pids, pid)
			}
		}
	}
	return pids
}

// invariant: the userspace era ends in three acts a rerun skips when done: its
// running daemon stops (detached, since a run arriving over that very node would
// cut its own session), its named files go, and its keychain accounts go; the old
// node stays in the admin console until the owner removes it, which the row says.
func (s *Provisioner) retire(ctx context.Context) (rows []setup.Step, background bool, err error) {
	if pids := s.livePIDs(); len(pids) > 0 {
		if !s.o.Detached {
			if err := s.detach(); err != nil {
				return nil, false, err
			}
			return []setup.Step{{Name: stepLayout, State: catalog.StateDegraded, Detail: catalog.RetiredNodeNote, Log: s.logPath(), Changed: true}}, true, nil
		}
		for _, pid := range pids {
			if err := s.stop(ctx, pid); err != nil {
				return nil, false, err
			}
		}
	}
	removed := false
	for _, dir := range s.retiredDirs() {
		for _, name := range catalog.RetiredNodeFiles() {
			removed = s.drop(path.Join(dir, name)) || removed
		}
		removed = s.drop(dir) || removed
	}
	if removed {
		thisMac, _ := s.reg.ThisMac()
		rows = append(rows, setup.Step{Name: stepLayout, State: catalog.StateOK, Changed: true,
			Detail: catalog.RetiredNodeDetail(thisMac)})
	}
	left := false
	for _, account := range retiredAccounts() {
		switch err := s.vault.Forget(ctx, account); {
		case err == nil:
			left = true
		case errors.Is(err, accounts.ErrNotKept):
		default:
			return nil, false, err
		}
	}
	if left {
		rows = append(rows, setup.Step{Name: stepLayout, State: catalog.StateOK, Changed: true,
			Detail: "the VM-join client and the tailnet auth key left the keychain; revoke the OAuth client at " + catalog.OAuthConsole})
	}
	return rows, false, s.migrateStores(ctx)
}

func (s *Provisioner) migrateStores(ctx context.Context) error {
	for _, m := range s.p.Stores {
		if err := m.Migrate(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Provisioner) stop(ctx context.Context, pid int) error {
	if err := s.m.Terminate(pid); err != nil {
		return err
	}
	for range 20 {
		if !s.m.Alive(pid) {
			return nil
		}
		if err := s.m.Sleep(ctx, 250*time.Millisecond); err != nil {
			return err
		}
	}
	return s.m.Kill(pid)
}
