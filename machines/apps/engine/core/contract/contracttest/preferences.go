package contracttest

import (
	"reflect"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/contract"
)

type example struct {
	Range    string `json:"range,omitempty"`
	MaxPerVM int    `json:"max_per_vm,omitempty"`
}

func Preferences(t *testing.T, open func(t *testing.T, groups []string) contract.Preferences) {
	groups := []string{"name", "ports", "share_list"}

	t.Run("a group never written reads as nothing, the value handed left as it was", func(t *testing.T) {
		got := example{Range: "kept"}
		if err := open(t, groups).Read("ports", &got); err != nil {
			t.Fatalf("Read: %v", err)
		}
		if got.Range != "kept" {
			t.Errorf("Read changed the value to %+v", got)
		}
	})

	t.Run("a group written reads back as it was written", func(t *testing.T) {
		p := open(t, groups)
		want := example{Range: "41000-41999", MaxPerVM: 5}
		if err := p.Write("ports", want); err != nil {
			t.Fatalf("Write: %v", err)
		}
		var got example
		if err := p.Read("ports", &got); err != nil || got != want {
			t.Errorf("Read = %+v, %v; want %+v", got, err, want)
		}
	})

	t.Run("writing one group keeps every other as it was", func(t *testing.T) {
		p := open(t, groups)
		if err := p.Write("name", "mac-mini"); err != nil {
			t.Fatalf("Write name: %v", err)
		}
		if err := p.Write("share_list", []string{"session:a"}); err != nil {
			t.Fatalf("Write share_list: %v", err)
		}
		if err := p.Write("share_list", []string{"session:a", "session:b"}); err != nil {
			t.Fatalf("Write share_list again: %v", err)
		}
		var name string
		var list []string
		if err := p.Read("name", &name); err != nil || name != "mac-mini" {
			t.Errorf("name = %q, %v; want mac-mini", name, err)
		}
		if err := p.Read("share_list", &list); err != nil || !reflect.DeepEqual(list, []string{"session:a", "session:b"}) {
			t.Errorf("share_list = %v, %v; want the second list", list, err)
		}
	})

	t.Run("an empty value removes its group", func(t *testing.T) {
		p := open(t, groups)
		if err := p.Write("name", "mac-mini"); err != nil {
			t.Fatalf("Write: %v", err)
		}
		if err := p.Write("name", ""); err != nil {
			t.Fatalf("Write empty: %v", err)
		}
		name := "untouched"
		if err := p.Read("name", &name); err != nil || name != "untouched" {
			t.Errorf("name = %q, %v; want the group gone", name, err)
		}
	})
}
