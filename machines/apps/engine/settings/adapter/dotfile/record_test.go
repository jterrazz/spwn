package dotfile_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/dotfile"
)

// value: the bench's applier anew over other files and other embedded
// fragments, the home and the record unchanged.
func (b *bench) over(files dotfile.Files, ours fs.FS) {
	b.applier = dotfile.New(files, silent{}, dotfile.Options{Home: b.home, Ours: ours, Record: b.record})
}

// value: the bench's disk, counting each file read whole.
type counted struct {
	disk
	reads map[string]int
}

func (c counted) ReadFile(p string) ([]byte, error) {
	c.reads[p]++
	return c.disk.ReadFile(p)
}

// value: embedded fragments, counting each read whole.
type countedFS struct {
	fstest.MapFS
	reads map[string]int
}

func (c countedFS) ReadFile(name string) ([]byte, error) {
	c.reads[name]++
	return c.MapFS.ReadFile(name)
}

// value: the record as JSON, edited by hand, then written back.
func (b *bench) editRecord(edit func(file map[string]any)) {
	b.t.Helper()
	data, err := os.ReadFile(b.record)
	if err != nil {
		b.t.Fatal(err)
	}
	var r map[string]any
	if err := json.Unmarshal(data, &r); err != nil {
		b.t.Fatal(err)
	}
	files, _ := r["files"].([]any)
	file, _ := files[0].(map[string]any)
	edit(file)
	out, _ := json.Marshal(r)
	if err := os.WriteFile(b.record, out, 0o600); err != nil {
		b.t.Fatal(err)
	}
}

func entryOf(file map[string]any, i int) map[string]any {
	values, _ := file["values"].([]any)
	e, _ := values[i].(map[string]any)
	return e
}

func TestARecordWithAnEntryMachinesNeverWritesRefusesTheWholeTakeBack(t *testing.T) {
	for name, edit := range map[string]func(file map[string]any){
		"a kind it never writes":     func(file map[string]any) { entryOf(file, 1)["kind"] = "wholefile" },
		"a pointer with no slash":    func(file map[string]any) { entryOf(file, 1)["pointer"] = "b" },
		"a pointer with a bad tilde": func(file map[string]any) { entryOf(file, 1)["pointer"] = "/b~2" },
		"the top of the file":        func(file map[string]any) { entryOf(file, 1)["pointer"] = "" },
		"a digest of another form":   func(file map[string]any) { entryOf(file, 1)["digest"] = "md5:00" },
		"a container with a digest":  func(file map[string]any) { entryOf(file, 1)["kind"] = "object" },
		"a path out of the home":     func(file map[string]any) { file["path"] = "/etc/x.json" },
		"no setting":                 func(file map[string]any) { file["setting"] = "" },
		"a folder that is no path":   func(file map[string]any) { file["folders"] = []any{"relative/dir"} },
	} {
		t.Run(name, func(t *testing.T) {
			b := newBench(t, catalog.WriteMerge, `{"a": 1, "b": 2}`, ".x.json")
			b.write("{\n  \"keep\": 1\n}\n", 0o600)
			b.apply()
			merged := b.read()
			b.editRecord(edit)
			for _, call := range []string{"off", "apply"} {
				var err error
				if call == "off" {
					_, err = b.applier.Undo(t.Context(), b.setting)
				} else {
					_, err = b.applier.Apply(t.Context(), b.setting, nil)
				}
				if err == nil || !strings.Contains(err.Error(), "merged.json") || !strings.Contains(err.Error(), "Machines takes nothing back by a record it cannot trust") {
					t.Errorf("%s: %v; want the record refused whole", call, err)
				}
				if b.read() != merged {
					t.Errorf("%s took something back by a record it cannot trust: %q", call, b.read())
				}
			}
		})
	}
}

func TestAFolderTheRecordNamesGoesOnlyIfItHoldsTheFileInTheHome(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, `{"a": 1}`, ".config/new/x.json")
	if err := os.MkdirAll(filepath.Join(b.home, ".config"), 0o750); err != nil {
		t.Fatal(err)
	}
	b.apply()
	mine, outside := filepath.Join(b.home, "Projects"), t.TempDir()
	if err := os.MkdirAll(mine, 0o750); err != nil {
		t.Fatal(err)
	}
	kept := []string{mine, outside, b.home, filepath.Dir(b.home)}
	b.editRecord(func(file map[string]any) {
		folders, _ := file["folders"].([]any)
		for _, k := range kept {
			folders = append(folders, k)
		}
		file["folders"] = folders
	})
	if got := b.off(); !reflect.DeepEqual(got, []string{"~/.config/new/x.json: removed"}) {
		t.Errorf("off said %q", got)
	}
	for _, k := range kept {
		if _, err := b.files.Stat(k); err != nil {
			t.Errorf("a folder the record named and Machines did not make for the file went: %s: %v", k, err)
		}
	}
	if _, err := b.files.Stat(filepath.Join(b.home, ".config", "new")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the folder Machines made stayed: %v", err)
	}
}

func TestAMadeFileTheRecordClaimsThroughALinkIsNeverRemoved(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, `{"a": 1}`, ".x.json")
	theirs := b.put("dotfiles/x.json", "{}\n")
	b.link("dotfiles/x.json", ".x.json")
	b.apply()
	b.editRecord(func(file map[string]any) { file["made"] = true })
	b.off()
	if data, err := os.ReadFile(theirs); err != nil || string(data) != "{}\n" {
		t.Errorf("the owner's linked file holds %q, %v", data, err)
	}
	if to, err := b.files.Readlink(b.file); err != nil || to != "dotfiles/x.json" {
		t.Errorf("the link is now %q, %v", to, err)
	}
}

func TestTheBoundsAreHeldBeforeAnyRead(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, `{"a": 1}`, ".x.json")
	b.write(`{"b": "`+strings.Repeat("x", 1<<20)+`"}`, 0o600)
	reads, embedded := map[string]int{}, map[string]int{}
	ours := countedFS{MapFS: fstest.MapFS{"pkg/fragment.json": {Data: []byte(`{"b": "` + strings.Repeat("x", 256<<10) + `"}`)}}, reads: embedded}
	b.over(counted{disk: b.files, reads: reads}, ours)
	if _, err := b.applier.Apply(t.Context(), b.setting, nil); err == nil || !strings.Contains(err.Error(), "past 256 KiB") || embedded["pkg/fragment.json"] > 0 {
		t.Errorf("Apply: %v, the fragment read %d times; want it refused unread", err, embedded["pkg/fragment.json"])
	}
	ours.MapFS["pkg/fragment.json"].Data = []byte(`{"a": 1}`)
	if _, err := b.applier.Apply(t.Context(), b.setting, nil); err == nil || !strings.Contains(err.Error(), "past 1 MiB") || reads[b.file] > 0 {
		t.Errorf("Apply: %v, the file read %d times; want it refused unread", err, reads[b.file])
	}
	b.write("{}\n", 0o600)
	b.put(b.record, "{\"v\": 1, \"files\": [], \"pad\": \""+strings.Repeat("x", 1<<20)+"\"}\n")
	if _, err := b.applier.Undo(t.Context(), b.setting); err == nil || !strings.Contains(err.Error(), "past 1 MiB") || reads[b.record] > 0 {
		t.Errorf("Undo: %v, the record read %d times; want it refused unread", err, reads[b.record])
	}
}
