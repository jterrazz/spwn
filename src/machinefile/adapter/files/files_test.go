package files_test

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/adapter/files"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/src/platform/local"
)

func TestTheOwnersFoldersKeepTheContract(t *testing.T) {
	contracttest.Source(t, func(t *testing.T, held map[string]string) contract.Source {
		root := t.TempDir()
		for p, body := range held {
			if err := os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, p), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return files.New(local.New(nil, nil, nil, nil).FS, filepath.Join(root, "catalog"), filepath.Join(root, "machinefiles"), machinefile.User, "~/.config/station")
	})
}

func TestAnEmbeddedTreeKeepsTheContract(t *testing.T) {
	contracttest.Source(t, func(_ *testing.T, held map[string]string) contract.Source {
		fsys := fstest.MapFS{}
		for p, body := range held {
			fsys[p] = &fstest.MapFile{Data: []byte(body)}
		}
		return files.New(files.Embedded{FS: fsys}, "catalog", "machinefiles", machinefile.Builtin, "built-in")
	})
}
