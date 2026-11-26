package jsonfile_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/core/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

func disk() jsonfile.Files { return local.New(nil, nil, nil, nil).FS }

func TestPreferences(t *testing.T) {
	contracttest.Preferences(t, func(t *testing.T, groups []string) contract.Preferences {
		return jsonfile.NewPreferences(disk(), t.TempDir(), groups)
	})
}

func TestPreferencesFileReadsAsTodaysPreferences(t *testing.T) {
	dir := t.TempDir()
	p := jsonfile.NewPreferences(disk(), dir, []string{"name", "macs", "vm_user"})
	if err := p.Write("vm_user", "tester"); err != nil {
		t.Fatal(err)
	}
	if err := p.Write("name", "macbook"); err != nil {
		t.Fatal(err)
	}
	if err := p.Write("macs", map[string]struct{}{"macbook": {}}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "preferences.json"))
	want := "{\n  \"name\": \"macbook\",\n  \"macs\": {\n    \"macbook\": {}\n  },\n  \"vm_user\": \"tester\"\n}\n"
	if string(got) != want {
		t.Errorf("preferences.json =\n%s\nwant\n%s", got, want)
	}
}

func TestAKeyNoGroupNamesIsDroppedAtTheNextWrite(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "preferences.json"), []byte(`{"here":"x","stray":{"mode":"auto"}}`), 0o600)
	if err := jsonfile.NewPreferences(disk(), dir, []string{"here"}).Write("here", "y"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "preferences.json")); string(got) != "{\n  \"here\": \"y\"\n}\n" {
		t.Errorf("preferences.json = %s", got)
	}
}

func TestPreferencesOfANewerStationAreRefusedNeverRewritten(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "preferences.json")
	_ = os.WriteFile(at, []byte(`{"v":2,"here":"x"}`), 0o600)
	p := jsonfile.NewPreferences(disk(), dir, []string{"here"})
	var newer *store.NewerError
	var here string
	if err := p.Read("here", &here); !errors.As(err, &newer) {
		t.Fatalf("Read = %v, want a newer version refused", err)
	}
	if err := p.Write("here", "y"); !errors.As(err, &newer) {
		t.Fatalf("Write = %v, want a newer version refused", err)
	}
	if got, _ := os.ReadFile(at); string(got) != `{"v":2,"here":"x"}` {
		t.Errorf("preferences.json = %s, want it left as it was", got)
	}
}

type openedFiles struct {
	jsonfile.Files
	reads int
}

func (o *openedFiles) ReadFile(p string) ([]byte, error) {
	o.reads++
	return o.Files.ReadFile(p)
}

func TestPreferencesAreReadAgainOnlyWhenTheFileChanged(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "preferences.json")
	files := &openedFiles{Files: disk()}
	p := jsonfile.NewPreferences(files, dir, []string{"here"})
	var here string
	read := func() string {
		here = ""
		if err := p.Read("here", &here); err != nil {
			t.Fatal(err)
		}
		return here
	}

	for range 5 {
		if read() != "" {
			t.Fatalf("a file that is not there read as %q", here)
		}
	}
	if files.reads != 1 {
		t.Errorf("5 reads of an absent file opened it %d times; want once", files.reads)
	}

	_ = os.WriteFile(at, []byte(`{"here":"x"}`), 0o600)
	for range 5 {
		if read() != "x" {
			t.Fatalf("a file another process made read as %q; want x", here)
		}
	}
	if files.reads != 2 {
		t.Errorf("5 reads of a file that appeared opened it %d times in all; want 2, once for it", files.reads)
	}

	_ = os.WriteFile(at, []byte(`{"here":"yy"}`), 0o600)
	if read() != "yy" || files.reads != 3 {
		t.Errorf("after another process wrote it, read %q in %d opens; want yy in 3", here, files.reads)
	}

	if err := p.Write("here", "zz"); err != nil {
		t.Fatal(err)
	}
	if read() != "zz" {
		t.Errorf("after this store wrote it, read %q; want zz", here)
	}
}

func TestAWriteOfAnotherStoreOfTheProcessIsSeenWhateverTheDisksStamp(t *testing.T) {
	files := localtest.NewFS()
	groups := []string{"here"}
	mine := jsonfile.NewPreferences(files, "/prefs", groups)
	theirs := jsonfile.NewPreferences(files, "/prefs", groups)
	var here string
	for _, word := range []string{"a", "b", "c"} {
		if err := theirs.Write("here", word); err != nil {
			t.Fatal(err)
		}
		if err := mine.Read("here", &here); err != nil || here != word {
			t.Fatalf("after another store wrote %q, read %q (%v): a disk that keeps one stamp and size hid the write", word, here, err)
		}
	}
}
