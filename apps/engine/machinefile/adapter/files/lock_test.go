package files_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/files"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func TestTheLocksLiveBesideTheOwnersMachinefiles(t *testing.T) {
	contracttest.Locks(t, func(t *testing.T) (contract.Locks, func(name, body string)) {
		dir := filepath.Join(t.TempDir(), "machinefiles")
		put := func(name, body string) {
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, name+".lock"), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return files.NewLocks(local.New(nil, nil, nil, nil).FS, dir, "~/.config/station/machinefiles"), put
	})
}
