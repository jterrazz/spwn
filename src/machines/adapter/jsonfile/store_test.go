package jsonfile_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/machines/adapter/jsonfile"
	"github.com/jterrazz/studio-station/src/machines/contract"
	"github.com/jterrazz/studio-station/src/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/src/platform/local"
)

func disk() jsonfile.Files { return local.New(nil, nil, nil, nil).FS }

func TestMachineStore(t *testing.T) {
	contracttest.MachineStore(t, func(t *testing.T) contract.MachineStore {
		return jsonfile.New(disk(), filepath.Join(t.TempDir(), "vms"))
	})
}

func TestAPutKeepsTheV2SharesUntilTheyMoveOut(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "api.json")
	_ = os.WriteFile(at, []byte(`{"name":"api","image":"linux","lent":[{"name":"session:a","at":"2026-11-01T09:00:00Z","until":"2026-11-01T17:00:00Z"}]}`), 0o600)
	s := jsonfile.New(disk(), dir)
	r, err := s.Get(context.Background(), "api")
	if err != nil {
		t.Fatal(err)
	}
	r.Want = "stopped"
	if err := s.Put(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(at); !strings.Contains(string(data), `"until": "2026-11-01T17:00:00Z"`) {
		t.Errorf("the share was lost:\n%s", data)
	}
}

func TestAReadableListingSkipsADamagedRecordButNotANewerOne(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "api.json"), []byte(`{"name":"api","image":"linux","version":"2026.11.01"}`), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "x.json"), []byte(`{"name": "x", "ima`), 0o600)
	s := jsonfile.New(disk(), dir)
	if _, err := s.List(context.Background()); err == nil {
		t.Errorf("List passed a damaged record; want it named")
	}
	list, err := s.ListReadable(context.Background())
	if err != nil || len(list) != 1 || list[0].Name != "api" {
		t.Fatalf("ListReadable = %+v, %v; want api alone", list, err)
	}
	_ = os.WriteFile(filepath.Join(dir, "new.json"), []byte(`{"v":3,"name":"new","image":"linux"}`), 0o600)
	if _, err := s.ListReadable(context.Background()); err == nil {
		t.Errorf("ListReadable passed a record a newer station wrote; want it refused")
	}
}

func TestMigrateRewritesAnOlderRecordInTodaysLifetimeWordsAndLeavesTheRest(t *testing.T) {
	dir := t.TempDir()
	old := map[string]string{
		"api":   `{"name":"api","image":"linux","lifetime":"permanent","lent":[{"name":"session:a","at":"2026-11-01T09:00:00Z","until":"2026-11-01T17:00:00Z"}]}`,
		"v2":    `{"name":"v2","image":"linux","lifetime":"lasting"}`,
		"temp":  `{"name":"temp","image":"linux","lifetime":"throwaway"}`,
		"today": "{\n  \"name\": \"today\",\n  \"lifetime\": \"ephemeral\",\n  \"v\": 2\n}\n",
	}
	for name, body := range old {
		_ = os.WriteFile(filepath.Join(dir, name+".json"), []byte(body), 0o600)
	}
	s := jsonfile.New(disk(), dir)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"api": "persistent", "v2": "persistent", "temp": "ephemeral"} {
		data, _ := os.ReadFile(filepath.Join(dir, name+".json"))
		if !strings.Contains(string(data), `"lifetime": "`+want+`"`) || !strings.Contains(string(data), `"v": 2`) {
			t.Errorf("%s was not rewritten at version 2 with %s:\n%s", name, want, data)
		}
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "api.json")); !strings.Contains(string(data), `"until": "2026-11-01T17:00:00Z"`) {
		t.Errorf("the share was lost:\n%s", data)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "today.json")); string(data) != old["today"] {
		t.Errorf("a record already at version 2 changed:\n%s", data)
	}
}

func TestMigrateStopsOnARecordANewerStationWrote(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "new.json"), []byte(`{"v":3,"name":"new","image":"linux"}`), 0o600)
	if err := jsonfile.New(disk(), dir).Migrate(context.Background()); err == nil {
		t.Errorf("Migrate passed a record a newer station wrote; want it stopped")
	}
}
