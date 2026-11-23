package local_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func TestAMkdirThatFailsPartWayTakesItsFolderAway(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root gives a folder to anyone")
	}
	base := t.TempDir()
	if err := (local.Beneath{}).Mkdir(base, "made", 0o700, local.Owner{UID: os.Geteuid() + 1, GID: os.Getegid()}); err == nil {
		t.Fatal("Mkdir gave a folder away without the right to")
	}
	if _, err := os.Lstat(filepath.Join(base, "made")); !os.IsNotExist(err) {
		t.Fatalf("Mkdir that failed left its folder: %v", err)
	}
}
