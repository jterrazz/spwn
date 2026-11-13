package files_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/files"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func TestTheOwnersFolderKeepsWhatAPathNames(t *testing.T) {
	contracttest.Owned(t, func(t *testing.T, path, body string) (contract.Owned, contract.Source) {
		root := t.TempDir()
		if path != "" {
			if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		fsys := local.New(nil, nil, nil, nil).FS
		dir := filepath.Join(root, "config")
		owned := files.NewOwned(fsys, filepath.Join(dir, "machinefiles"))
		return pathed{owned, root}, files.New(fsys, contract.FileLinks, filepath.Join(dir, "catalog"), filepath.Join(dir, "machinefiles"), machinefile.User, "~/.config/station")
	})
}

// value: the folder's adapter with a path read under the test's root.
type pathed struct {
	files.Owned
	root string
}

func (p pathed) Read(ctx context.Context, path string) ([]byte, error) {
	return p.Owned.Read(ctx, filepath.Join(p.root, path))
}
