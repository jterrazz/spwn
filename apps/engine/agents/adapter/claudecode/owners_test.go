package claudecode_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func ownersKept(t *testing.T, settings string) bool {
	t.Helper()
	data, err := os.ReadFile(settings)
	if err != nil {
		t.Fatal(err)
	}
	var got, want map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("settings.json is no longer JSON: %v\n%s", err, data)
	}
	_ = json.Unmarshal([]byte(owners), &want)
	for key, value := range want {
		if key == "hooks" {
			continue
		}
		if !reflect.DeepEqual(got[key], value) {
			return false
		}
	}
	hooks, _ := json.Marshal(got["hooks"])
	return strings.Contains(string(hooks), "afplay /System/Library/Sounds/Glass.aiff")
}
