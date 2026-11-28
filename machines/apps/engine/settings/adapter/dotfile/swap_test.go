package dotfile_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the bench's disk with another writer (an editor, the app itself)
// that writes the file between Machines' read and its write, the first
// times times.
type racing struct {
	disk
	times  int
	writes []string
}

func (r *racing) Swap(p string, seen local.Seen, data []byte, perm fs.FileMode) error {
	if len(r.writes) < r.times {
		theirs := "{\n  \"editor\": " + strconv.Itoa(len(r.writes)+1) + "\n}\n"
		r.writes = append(r.writes, theirs)
		if err := os.WriteFile(p, []byte(theirs), 0o600); err != nil {
			return err
		}
	}
	return r.disk.Swap(p, seen, data, perm)
}

func (r *racing) Replace(p string, data []byte, perm fs.FileMode) error {
	if strings.HasSuffix(p, "merged.json") {
		return r.disk.Replace(p, data, perm)
	}
	panic("a fragment's file is written by Replace, which never checks what it overwrites: " + p)
}

func TestAFileAnotherWriterChangesBetweenTheReadAndTheWriteIsMergedAgain(t *testing.T) {
	for _, mode := range []catalog.WriteMode{catalog.WriteMerge, catalog.WriteBlock} {
		t.Run(string(mode), func(t *testing.T) {
			frag, to, ours := `{"a": 1}`, ".x.json", `"a": 1`
			if mode == catalog.WriteBlock {
				frag, to, ours = "set a 1\n", ".x.conf", "set a 1"
			}
			b := newBench(t, mode, frag, to)
			b.write("{\n  \"mine\": 0\n}\n", 0o600)
			race := &racing{disk: b.files, times: 2}
			b.over(race, b.ours)
			b.apply()
			if got := b.read(); !strings.HasPrefix(got, "{\n  \"editor\": 2") || !strings.Contains(got, ours) {
				t.Errorf("the file holds %q; want the other writer's last bytes with Machines' fragment in them", got)
			}
			race.times = 3
			b.off()
			if got := b.read(); got != race.writes[2] {
				t.Errorf("off left %q; want the other writer's bytes, which hold nothing of Machines'", got)
			}
		})
	}
}

func TestAFileAnotherWriterKeepsChangingIsRefusedAfterThreeTries(t *testing.T) {
	for _, mode := range []catalog.WriteMode{catalog.WriteMerge, catalog.WriteBlock} {
		t.Run(string(mode), func(t *testing.T) {
			frag, to := `{"a": 1}`, ".x.json"
			if mode == catalog.WriteBlock {
				frag, to = "set a 1\n", ".x.conf"
			}
			b := newBench(t, mode, frag, to)
			b.write("{\n  \"mine\": 0\n}\n", 0o600)
			race := &racing{disk: b.files, times: 3}
			b.over(race, b.ours)
			_, err := b.applier.Apply(t.Context(), b.setting, nil)
			if err == nil || !strings.Contains(err.Error(), "~/"+to+" changed under Machines 3 times") {
				t.Errorf("Apply: %v; want it refused after three tries, naming the file", err)
			}
			if got := b.read(); got != race.writes[2] {
				t.Errorf("the file holds %q; want the other writer's last bytes, untouched", got)
			}
			if _, err := b.files.Stat(b.record); err == nil {
				t.Error("a record was written for a fragment never written")
			}
		})
	}
}

// value: the bench's disk with another writer that writes the file once,
// right after Machines read it.
type writerAfterRead struct {
	disk
	file, theirs string
	wrote        *bool
}

func (w writerAfterRead) Open(p string) (fs.File, error) {
	if p != w.file || *w.wrote {
		return w.disk.Open(p)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	*w.wrote = true
	if werr := os.WriteFile(p, []byte(w.theirs), 0o600); werr != nil {
		return nil, werr
	}
	return fstest.MapFS{"read": {Data: data}}.Open("read")
}

func TestOffNeverRemovesAFileItMadeThatAnotherWriterChangedAfterItsRead(t *testing.T) {
	for _, mode := range []catalog.WriteMode{catalog.WriteMerge, catalog.WriteBlock} {
		t.Run(string(mode), func(t *testing.T) {
			frag, to, theirs, want := `{"a": 1}`, ".x.json", "{\n  \"a\": 1,\n  \"editor\": 1\n}\n", "{\n  \"editor\": 1\n}\n"
			if mode == catalog.WriteBlock {
				frag, to = "set a 1\n", ".x.conf"
				theirs = "# >>> station setting:fmt >>>\nset a 1\n# <<< station setting:fmt <<<\nset editor 1\n"
				want = "set editor 1\n"
			}
			b := newBench(t, mode, frag, to)
			b.apply()
			wrote := false
			b.over(writerAfterRead{disk: b.files, file: b.file, theirs: theirs, wrote: &wrote}, b.ours)
			b.off()
			if got := b.read(); !wrote || got != want {
				t.Errorf("off left %q; want the other writer's key, with nothing of Machines' left in it, %q", got, want)
			}
		})
	}
}

// value: the bench's disk where the first swap finds the file changed,
// and writes nothing.
type changedOnce struct {
	disk
	tried *bool
}

func (c changedOnce) Swap(p string, seen local.Seen, data []byte, perm fs.FileMode) error {
	if !*c.tried {
		*c.tried = true
		return local.ErrChanged
	}
	return c.disk.Swap(p, seen, data, perm)
}

func TestTheFoldersATryThatDidNotLandMadeAreTakenBackByOff(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, `{"a": 1}`, ".config/new/deep/x.json")
	if err := os.MkdirAll(filepath.Join(b.home, ".config"), 0o750); err != nil {
		t.Fatal(err)
	}
	tried := false
	b.over(changedOnce{disk: b.files, tried: &tried}, b.ours)
	b.apply()
	b.off()
	if _, err := b.files.Stat(filepath.Join(b.home, ".config", "new")); !tried || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a folder Machines made on a try that did not land stayed after off: %v", err)
	}
}

// value: the bench's disk whose lock another run holds: each wait ends as
// a wait that ran out, and says when it was to end.
type heldLock struct {
	disk
	until *time.Time
}

func (h heldLock) Hold(ctx context.Context, _ string) (func(), error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil, context.Canceled
	}
	*h.until = deadline
	return nil, context.DeadlineExceeded
}

func TestAWaitForTheFragmentsLockIsBounded(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, `{"a": 1}`, ".x.json")
	run, cancel := context.WithTimeout(t.Context(), time.Hour)
	defer cancel()
	ends, _ := run.Deadline()
	var until time.Time
	b.over(heldLock{disk: b.files, until: &until}, b.ours)
	_, err := b.applier.Apply(run, b.setting, nil)
	if err == nil || !strings.Contains(err.Error(), "merged.json.lock is held by another run") {
		t.Errorf("Apply: %v; want the held lock refused", err)
	}
	if given := time.Hour - ends.Sub(until); until.IsZero() || given > 11*time.Second {
		t.Errorf("the wait was given about %v; want at most 10s", given)
	}
	if _, err := b.files.Stat(b.file); err == nil {
		t.Error("a file was written with the lock not held")
	}
}
