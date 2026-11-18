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
