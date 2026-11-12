package contract_test

import (
	"encoding/json"
	"testing"

	"github.com/jterrazz/studio-station/src/machinefile/contract"
)

// value: preferences as a file holds them, one JSON document per group.
type file map[string]string

func (f file) Read(group string, v any) error {
	if raw, ok := f[group]; ok {
		return json.Unmarshal([]byte(raw), v)
	}
	return nil
}

func (f file) Write(group string, v any) error {
	raw, err := json.Marshal(v)
	f[group] = string(raw)
	return err
}

func TestAnOlderSourceThatNamesACommitIsReadByItsNameAndURL(t *testing.T) {
	prefs := file{contract.GroupSource: `{"source": {"name": "studio", "url": "https://example.com/studio.git", "commit": "1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d"}}`}
	src, ok, err := contract.NewSourcePrefs(prefs).Get()
	if err != nil || !ok {
		t.Fatalf("read %v, %v", ok, err)
	}
	if src != (contract.SharedSource{Name: "studio", URL: "https://example.com/studio.git"}) {
		t.Errorf("read %+v", src)
	}
}
