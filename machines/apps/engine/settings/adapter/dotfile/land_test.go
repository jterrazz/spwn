package dotfile_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// value: a record of another setting's, padded by one folder name of
// width bytes, so its size moves byte for byte with width.
func padded(width int) string {
	return "{\"v\": 1, \"files\": [{\"setting\": \"other\", \"path\": \"~/.y.conf\", \"block\": \"sha256:" + strings.Repeat("0", 64) + "\", \"folders\": [\"/" + strings.Repeat("x", width) + "\"]}]}\n"
}

func TestAFragmentThatMakesAFileInNewFoldersIsMeasuredWithThemBeforeAnyWrite(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, `{"a": 1}`, ".config/new/deep/x.json")
	b.write("{}\n", 0o600)
	b.put(b.record, padded(1000))
	b.apply()
	info, err := b.files.Stat(b.record)
	if err != nil {
		t.Fatal(err)
	}
	room := (1 << 20) - int(info.Size()) - 8
	b.off()
	if err := b.files.RemoveAll(filepath.Join(b.home, ".config", "new")); err != nil {
		t.Fatal(err)
	}
	before := padded(1000 + room)
	b.put(b.record, before)
	if _, err := b.applier.Apply(t.Context(), b.setting, nil); err == nil || !strings.Contains(err.Error(), "merged.json would pass 1 MiB") {
		t.Errorf("Apply: %v; want the apply refused by the record its new file and folders would write", err)
	}
	if _, err := b.files.Stat(filepath.Join(b.home, ".config", "new")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a folder or the file was made: %v", err)
	}
	if data, _ := os.ReadFile(b.record); string(data) != before {
		t.Errorf("the record moved: %d bytes", len(data))
	}
}
