package automation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func receiptsProject(t *testing.T) (root, outside string) {
	t.Helper()
	dir := t.TempDir()
	root, outside = filepath.Join(dir, "site"), filepath.Join(dir, "outside")
	for _, d := range []string{filepath.Join(root, ".spwn"), outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "rc"), []byte("secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, outside
}

func TestLogsFollowReadsNoReceiptsThroughALinkOut(t *testing.T) {
	refusals := map[bool]string{}
	for _, exists := range []bool{true, false} {
		root, outside := receiptsProject(t)
		target := filepath.Join(outside, "rc")
		if !exists {
			target = filepath.Join(outside, "missing")
		}
		if err := os.Symlink(target, receiptsPath(root)); err != nil {
			t.Fatal(err)
		}
		f, err := openReceipts(root)
		if err == nil {
			f.Close()
			t.Fatal("the receipts were opened through a link out")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if f, werr := waitForFile(root, ctx); werr == nil || werr == context.DeadlineExceeded {
			if f != nil {
				f.Close()
			}
			t.Errorf("--follow waited on or opened a link out: %v", werr)
		}
		cancel()
		refusals[exists] = strings.ReplaceAll(err.Error(), root, "<root>")
	}
	if refusals[true] != refusals[false] {
		t.Errorf("the refusal tells what is outside: %q vs %q", refusals[true], refusals[false])
	}
}

func TestLogsFollowDoesNotBlockOnAFIFO(t *testing.T) {
	root, _ := receiptsProject(t)
	if err := syscall.Mkfifo(receiptsPath(root), 0o644); err != nil {
		t.Skipf("mkfifo: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		f, err := openReceipts(root)
		if f != nil {
			f.Close()
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("a FIFO was opened as the receipts")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("a FIFO receipts file blocked logs --follow")
	}
}
