package jsonfile_test

import (
	"context"
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
	p := jsonfile.NewPreferences(disk(), dir, []string{"here", "hosts", "vm_user"})
	if err := p.Write("vm_user", "tester"); err != nil {
		t.Fatal(err)
	}
	if err := p.Write("here", "macbook"); err != nil {
		t.Fatal(err)
	}
	if err := p.Write("hosts", map[string]struct{}{"macbook": {}}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "preferences.json"))
	want := "{\n  \"here\": \"macbook\",\n  \"hosts\": {\n    \"macbook\": {}\n  },\n  \"vm_user\": \"tester\"\n}\n"
	if string(got) != want {
		t.Errorf("preferences.json =\n%s\nwant\n%s", got, want)
	}
}

func TestPreferencesMigration(t *testing.T) {
	t.Run("a retired key is dropped, every known group kept", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dir, "preferences.json"), []byte(`{"here":"x","tailnet":{"mode":"auto"}}`), 0o600)
		if err := jsonfile.NewPreferences(disk(), dir, []string{"here"}).Migrate(context.Background()); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(filepath.Join(dir, "preferences.json")); string(got) != "{\n  \"here\": \"x\"\n}\n" {
			t.Errorf("preferences.json = %s", got)
		}
	})
	t.Run("a group under its retired name is kept under its new one", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dir, "preferences.json"), []byte(`{"here":"x","hosts":{"mini":{}}}`), 0o600)
		p := jsonfile.NewPreferences(disk(), dir, []string{"here", "macs"}).Renaming("hosts", "macs")
		var macs map[string]struct{}
		if err := p.Read("macs", &macs); err != nil || len(macs) != 1 {
			t.Fatalf("Read(macs) = %v, %v; want the mini read under its retired name", macs, err)
		}
		if err := p.Migrate(context.Background()); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(filepath.Join(dir, "preferences.json")); string(got) != "{\n  \"here\": \"x\",\n  \"macs\": {\n    \"mini\": {}\n  }\n}\n" {
			t.Errorf("preferences.json = %s", got)
		}
	})
	t.Run("a file of a newer station is refused, never rewritten", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dir, "preferences.json"), []byte(`{"v":2,"here":"x"}`), 0o600)
		var newer *store.NewerError
		if err := jsonfile.NewPreferences(disk(), dir, []string{"here"}).Migrate(context.Background()); !errors.As(err, &newer) {
			t.Fatalf("Migrate = %v, want a newer version refused", err)
		}
	})
}
