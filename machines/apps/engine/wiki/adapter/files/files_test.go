package files_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/wiki/adapter/files"
	"github.com/jterrazz/studio-station/apps/engine/wiki/contract"
	"github.com/jterrazz/studio-station/apps/engine/wiki/contract/contracttest"
)

func TestCorpus(t *testing.T) {
	disk := local.New(nil, nil, nil, nil).FS
	contracttest.Corpus(t, func(t *testing.T) contracttest.Subject {
		root := t.TempDir()
		return contracttest.Subject{Corpus: files.New(disk, root), Put: func(path, body string) {
			at := filepath.Join(root, filepath.FromSlash(path))
			if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(at, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}}
	}, func(*testing.T) contract.Corpus { return files.New(disk, "") })
}
