package jsonfile_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func open(at string) *jsonfile.ReleaseCache {
	return jsonfile.NewReleaseCache(local.New(nil, nil, nil, nil).FS, at)
}

func TestTheReleaseFileKeepsTheContract(t *testing.T) {
	contracttest.ReleaseCache(t, func(t *testing.T) contract.ReleaseCache {
		return open(filepath.Join(t.TempDir(), "station", "release.json"))
	})
}

func TestTheReleaseFileIsVersionOne(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "release.json")
	read := time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC)
	if err := open(at).Write(t.Context(), engine.ReleaseKept{Read: read, Latest: &engine.Tagged{Tag: "v0.22.0", URL: "https://github.com/jterrazz/spwn/releases/tag/v0.22.0"}}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"v\": 1,\n  \"read\": \"2026-11-19T08:00:00Z\",\n  \"latest\": {\n    \"tag\": \"v0.22.0\",\n    \"url\": \"https://github.com/jterrazz/spwn/releases/tag/v0.22.0\"\n  }\n}\n"
	if string(got) != want {
		t.Errorf("wrote %s; want %s", got, want)
	}
	if err := os.WriteFile(at, []byte(`{"v":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := open(at).Read(t.Context()); err == nil {
		t.Error("read a file of a later version")
	}
}
