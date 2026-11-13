package calls

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
)

type prefs map[string][]byte

func (p prefs) Read(group string, v any) error {
	if data, ok := p[group]; ok {
		return json.Unmarshal(data, v)
	}
	return nil
}

func (p prefs) Write(group string, v any) error {
	data, err := json.Marshal(v)
	p[group] = data
	return err
}

func TestASettingPutsANameOnTheShareListOnceAndTakesItOff(t *testing.T) {
	list := NewShareList(contract.ShareList(prefs{}))
	for range 2 {
		if err := list.Add("operator:claude:oauth-token"); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := list.Get()
	if n := len(slices.DeleteFunc(slices.Clone(got), func(s string) bool { return s != "operator:claude:oauth-token" })); n != 1 {
		t.Fatalf("the share list holds the token %d times after two applies: %v", n, got)
	}
	if err := list.Remove("operator:claude:oauth-token"); err != nil {
		t.Fatal(err)
	}
	if got, _ = list.Get(); slices.Contains(got, "operator:claude:oauth-token") {
		t.Fatalf("the share list still holds the token after the setting is off: %v", got)
	}
}
