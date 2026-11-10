package apfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/src/disk/contract"
	"github.com/jterrazz/studio-station/src/disk/contract/contracttest"
	"github.com/jterrazz/studio-station/src/platform/local"
)

type pgrep struct{}

func (pgrep) Output(context.Context, local.Cmd) (local.Output, error) {
	return local.Output{ExitCode: 1}, nil
}

func lay(t *testing.T, files map[string]int) string {
	root := t.TempDir()
	for p, n := range files {
		path := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, make([]byte, n), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestTheAPFSDiskUsageKeepsTheContract(t *testing.T) {
	contracttest.DiskUsage(t, func(t *testing.T, files map[string]int) (contract.DiskUsage, string) {
		return New(pgrep{}), lay(t, files)
	})
}
