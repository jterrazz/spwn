package jsonfile_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func TestShareStore(t *testing.T) {
	contracttest.ShareStore(t, func(t *testing.T, vms ...string) contract.ShareStore {
		dir := t.TempDir()
		for _, vm := range vms {
			if err := os.WriteFile(filepath.Join(dir, vm+".json"), []byte(`{"name":"`+vm+`"}`), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return jsonfile.NewShares(local.New(nil, nil, nil, nil).FS, filepath.Join(t.TempDir(), "shares.json"), dir)
	})
}

func TestAV2RecordsSharesMoveOutOnce(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "api.json")
	record := `{"name":"api","image":"linux","cpus":4,"lent":[{"name":"session:a","at":"2026-11-01T09:00:00Z","until":"2026-11-01T21:00:00Z"}]}`
	if err := os.WriteFile(at, []byte(record), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "shares.json")
	s := jsonfile.NewShares(local.New(nil, nil, nil, nil).FS, file, dir)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	held, err := s.Get(context.Background(), "api")
	if err != nil || len(held) != 1 || held[0].Name != "session:a" || held[0].Until.Hour() != 21 {
		t.Fatalf("Get = %+v, %v; want session:a until 21:00, moved out of the record", held, err)
	}
	data, _ := os.ReadFile(at)
	var got map[string]any
	_ = json.Unmarshal(data, &got)
	if want := map[string]any{"name": "api", "image": "linux", "cpus": 4.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("the record keeps more than its own keys:\n%s", data)
	}
	first, _ := os.ReadFile(file)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if again, _ := os.ReadFile(file); !bytes.Equal(again, first) {
		t.Errorf("a second migration rewrote shares.json:\n%s", again)
	}
}

type writtenAt struct {
	jsonfile.Files
	at time.Time
}

func (w writtenAt) Stat(p string) (fs.FileInfo, error) {
	info, err := w.Files.Stat(p)
	if err != nil {
		return nil, err
	}
	return modTime{FileInfo: info, at: w.at}, nil
}

type modTime struct {
	fs.FileInfo
	at time.Time
}

func (m modTime) ModTime() time.Time { return m.at }

func TestAV2RecordsPlainNamesMoveOutAsSharedAtTheRecordsTimeForTheDefaultTerm(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "api.json")
	if err := os.WriteFile(at, []byte(`{"name":"api","lent":["session:a","operator:claude:oauth-token"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	written := time.Date(2026, 10, 30, 8, 0, 0, 0, time.UTC)
	s := jsonfile.NewShares(writtenAt{Files: local.New(nil, nil, nil, nil).FS, at: written}, filepath.Join(t.TempDir(), "shares.json"), dir)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	held, err := s.Get(context.Background(), "api")
	want := []accounts.Share{
		{Name: "session:a", At: written, Until: written.Add(accounts.DefaultShareTerm)},
		{Name: "operator:claude:oauth-token", At: written, Until: written.Add(accounts.DefaultShareTerm)},
	}
	if err != nil || !reflect.DeepEqual(held, want) {
		t.Fatalf("Get = %+v, %v; want %+v", held, err, want)
	}
	if data, _ := os.ReadFile(at); bytes.Contains(data, []byte("lent")) {
		t.Errorf("the record still holds its shares:\n%s", data)
	}
}
