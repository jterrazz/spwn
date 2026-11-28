package dotfile_test

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the bench's disk whose swap lands the file, then fails as a
// sync of its folder would.
type unsynced struct{ disk }

var errUnsynced = errors.New("the folder's sync failed")

func (u unsynced) Swap(p string, seen local.Seen, data []byte, perm fs.FileMode) error {
	if err := u.disk.Swap(p, seen, data, perm); err != nil {
		return err
	}
	return errUnsynced
}

func TestAFileThatLandedBeforeItsSwapFailedIsRecordedSoOffTakesItBack(t *testing.T) {
	for _, mode := range []catalog.WriteMode{catalog.WriteMerge, catalog.WriteBlock} {
		t.Run(string(mode), func(t *testing.T) {
			frag, to := `{"a": 1}`, ".config/new/x.json"
			if mode == catalog.WriteBlock {
				frag, to = "set a 1\n", ".config/new/x.conf"
			}
			b := newBench(t, mode, frag, to)
			b.over(unsynced{disk: b.files}, b.ours)
			if _, err := b.applier.Apply(t.Context(), b.setting, nil); !errors.Is(err, errUnsynced) || !strings.Contains(err.Error(), "recorded") {
				t.Errorf("Apply: %v; want the failed sync said, with the file written and recorded", err)
			}
			b.over(b.files, b.ours)
			b.off()
			if _, err := b.files.Stat(filepath.Join(b.home, ".config", "new")); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("off left the file Machines made, or its folder: %v", err)
			}
		})
	}
}
