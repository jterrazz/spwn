package wire

import (
	"encoding/json"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type journal struct {
	local.FS
	writes  []string
	renames [][2]string
}

func (j *journal) WriteFile(path string, data []byte, perm fs.FileMode) error {
	j.writes = append(j.writes, path)
	return j.FS.WriteFile(path, data, perm)
}

func (j *journal) Rename(from, to string) error {
	j.renames = append(j.renames, [2]string{from, to})
	return j.FS.Rename(from, to)
}

func TestTheEngineRecordIsReplacedWholeAtTheStateFolderWithItsModes(t *testing.T) {
	disk := local.New(nil, nil, nil, nil).FS
	state := filepath.Join(t.TempDir(), "state", "station")
	file := filepath.Join(state, "engine.json")
	if err := disk.MkdirAll(state, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := disk.WriteFile(file, []byte(`{"v":1,"executable":"/old/station"}`), 0o640); err != nil {
		t.Fatal(err)
	}
	j := &journal{FS: disk}
	want := engine.Record{V: engine.RecordVersion, Executable: "/opt/bin/station", Version: "v0.21.0", At: time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC)}
	if err := (fileRecord{fs: j, file: file}).Write(want); err != nil {
		t.Fatal(err)
	}
	body, err := disk.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var got engine.Record
	if err = json.Unmarshal(body, &got); err != nil || got != want {
		t.Fatalf("%s holds %q (%v), want %+v", file, body, err, want)
	}
	for _, w := range j.writes {
		if w == file {
			t.Errorf("the record was written into %s, where a reader may hold it half written", file)
		}
	}
	if len(j.renames) != 1 || j.renames[0][1] != file {
		t.Errorf("renames %v, want one over %s", j.renames, file)
	}
	info, err := disk.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode %v, want 0600", info.Mode().Perm())
	}
	entries, err := disk.ReadDir(state)
	if err != nil || len(entries) != 1 || entries[0].Name() != "engine.json" {
		t.Errorf("%s holds %v (%v), want engine.json alone", state, entries, err)
	}
}

func TestTheEngineRecordMakesItsFolderPrivate(t *testing.T) {
	disk := local.New(nil, nil, nil, nil).FS
	state := filepath.Join(t.TempDir(), "state", "station")
	if err := (fileRecord{fs: disk, file: filepath.Join(state, "engine.json")}).Write(engine.Record{V: engine.RecordVersion}); err != nil {
		t.Fatal(err)
	}
	info, err := disk.Stat(state)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Errorf("folder mode %v, want 0700", info.Mode().Perm())
	}
}
