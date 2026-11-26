package files_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/wiki/adapter/files"
	"github.com/jterrazz/studio-station/apps/engine/wiki/contract"
	"github.com/jterrazz/studio-station/apps/engine/wiki/contract/contracttest"
)

func put(t *testing.T, root, path, body string) {
	t.Helper()
	at := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCorpus(t *testing.T) {
	contracttest.Corpus(t, func(t *testing.T) contracttest.Subject {
		root := t.TempDir()
		return contracttest.Subject{Corpus: files.New(root), Put: func(path, body string) { put(t, root, path, body) }}
	}, func(*testing.T) contract.Corpus { return files.New("") })
}

func linkedOut(t *testing.T, target string) error {
	t.Helper()
	root := t.TempDir()
	put(t, root, "wiki.yaml", "wiki: spwn.wiki/v1\n")
	put(t, root, "blueprint/philosophy.md", "# Philosophy\n")
	if err := local.New(nil, nil, nil, nil).FS.Symlink(target, filepath.Join(root, "blueprint", "secret.md")); err != nil {
		t.Fatal(err)
	}
	got, err := files.New(root).Files(context.Background())
	if err == nil {
		t.Fatalf("Files = %d files, read through a link out of the corpus; want a refusal", len(got))
	}
	return err
}

func TestAPageLinkedOutOfTheCorpusIsNeverRead(t *testing.T) {
	outside := t.TempDir()
	put(t, outside, "secret.md", "# The owner's notes\n")
	there := linkedOut(t, filepath.Join(outside, "secret.md"))
	missing := linkedOut(t, filepath.Join(outside, "absent.md"))
	if errors.Is(there, fs.ErrNotExist) || errors.Is(missing, fs.ErrNotExist) {
		t.Fatalf("refusals %v and %v say whether the outside target exists", there, missing)
	}
	if there.Error() != missing.Error() {
		t.Fatalf("refusal %q differs from %q: it tells whether the outside target exists", there, missing)
	}
}
