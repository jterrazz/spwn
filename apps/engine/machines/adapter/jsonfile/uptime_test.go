package jsonfile_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func uptimeDisk() jsonfile.UptimeFiles { return local.New(nil, nil, nil, nil).FS }

func TestUptimeStore(t *testing.T) {
	contracttest.UptimeStore(t, func(t *testing.T) contract.UptimeStore {
		return jsonfile.NewUptime(uptimeDisk(), filepath.Join(t.TempDir(), "uptime.json"))
	})
}

func TestTheUptimeFileIsVersionedAndKeptPrivate(t *testing.T) {
	file := filepath.Join(t.TempDir(), "uptime.json")
	s := jsonfile.NewUptime(uptimeDisk(), file)
	record, _ := s.Get(context.Background(), "infra")
	record.Days = nil
	if err := s.Put(context.Background(), "infra", record); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(file)
	if want := "{\n  \"schema\": \"station.uptime/v1\",\n  \"servers\": {\n    \"infra\": {}\n  }\n}\n"; string(data) != want {
		t.Errorf("uptime.json =\n%s\nwant\n%s", data, want)
	}
	if info, err := local.New(nil, nil, nil, nil).FS.Stat(file); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("uptime.json mode = %v; want 0600", err)
	}
}

func TestSetupMigratesTheUptimeFileAndRefusesANewerOne(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "uptime.json")
	s := jsonfile.NewUptime(uptimeDisk(), file)
	if err := s.Migrate(context.Background()); err != nil {
		t.Errorf("Migrate with no file = %v; want nothing to do", err)
	}
	unversioned := `{"servers":{"infra":{"since":"2026-10-20T09:00:00Z","days":[{"date":"2026-11-05","down_s":420,"restarts":1}]}}}`
	_ = os.WriteFile(file, []byte(unversioned), 0o600)
	if err := s.Migrate(context.Background()); err != nil {
		t.Errorf("Migrate of a file with no schema = %v; want it read as v1", err)
	}
	if data, _ := os.ReadFile(file); string(data) != unversioned {
		t.Errorf("Migrate rewrote a v1 file:\n%s", data)
	}
	if got, err := s.Get(context.Background(), "infra"); err != nil || len(got.Days) != 1 || got.Days[0].DownS != 420 {
		t.Errorf("Get = %+v, %v; want the day it kept", got, err)
	}
	_ = os.WriteFile(file, []byte(`{"schema":"station.uptime/v2","servers":{}}`), 0o600)
	var newer *store.NewerError
	if err := s.Migrate(context.Background()); !errors.As(err, &newer) {
		t.Errorf("Migrate of a file a newer station wrote = %v; want it refused", err)
	}
	if _, err := s.Get(context.Background(), "infra"); !errors.As(err, &newer) {
		t.Errorf("Get of a file a newer station wrote = %v; want it refused", err)
	}
}
