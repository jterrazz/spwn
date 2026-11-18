package wire

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

func zshrcOnDisk(t *testing.T) zshrcFile {
	t.Helper()
	dir := t.TempDir()
	at := filepath.Join(dir, ".zshrc")
	if err := os.WriteFile(at, []byte("mine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return zshrcFile{fs: local.New(nil, nil, io.Discard, io.Discard).FS, path: at, lock: filepath.Join(dir, "state", "zshrc.lock")}
}

func TestEveryBlockOfWritersAtOnceSurvives(t *testing.T) {
	z := zshrcOnDisk(t)
	const writers = 24
	var all sync.WaitGroup
	failed := make(chan error, writers)
	for i := range writers {
		all.Go(func() {
			id := fmt.Sprintf("station:w%d", i)
			failed <- z.Edit(t.Context(), func(text string) (string, error) {
				text, _ = settings.UpsertPackageBlock(text, id, "echo "+id+"\n")
				return text, nil
			})
		})
	}
	all.Wait()
	close(failed)
	for err := range failed {
		if err != nil {
			t.Fatal(err)
		}
	}
	text, err := z.Read(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got := settings.PackageBlockIDs(text); len(got) != writers {
		t.Errorf("%d of %d blocks survive: %v", len(got), writers, got)
	}
}

func TestAWaitForTheZshrcLockEndsWithItsContextAndReadsInterrupted(t *testing.T) {
	z := zshrcOnDisk(t)
	if err := os.MkdirAll(filepath.Dir(z.lock), 0o700); err != nil {
		t.Fatal(err)
	}
	release, err := local.Exclusive(t.Context(), z.lock)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	ended := make(chan error, 1)
	go func() {
		ended <- z.Edit(ctx, func(text string) (string, error) { return text + "late\n", nil })
	}()
	bound, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	select {
	case err := <-ended:
		if err == nil || err.Error() != "interrupted" || !errors.Is(err, context.Canceled) {
			t.Errorf("Edit = %v, want interrupted", err)
		}
	case <-bound.Done():
		t.Fatal("the wait for the lock outlived its context")
	}
	if text, _ := z.Read(t.Context()); text != "mine\n" {
		t.Errorf("an interrupted edit wrote %q", text)
	}
}
