package architecture_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type seededMac struct {
	VMs         map[string]json.RawMessage `json:"vms"`
	Preferences struct {
		Macs map[string]json.RawMessage `json:"macs"`
	} `json:"preferences"`
}

// value: each world of specs/usecases/worlds/ by its file's name, its Macs
// and VMs as the seed writes them.
func seededWorlds(t *testing.T, root string) map[string]map[string]seededMac {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(root, "specs", "usecases", "worlds", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string]seededMac{}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var w struct {
			Macs map[string]seededMac `json:"macs"`
		}
		if err := json.Unmarshal(data, &w); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		out[strings.TrimSuffix(filepath.Base(path), ".json")] = w.Macs
	}
	return out
}

// invariant: rule worlds-macs-know-each-other: in a world, each Mac a Mac
// knows knows the same Macs back, itself and every other one, as the
// owner's Macs do once each has added the others; so a name any of them
// holds is held at each (machines.create's claims), and a world never
// stands for a Mac that would hand out a name another Mac holds.
func ruleWorldsMacsKnowEachOther(worlds map[string]map[string]seededMac) []finding {
	var out []finding
	for _, name := range slices.Sorted(maps.Keys(worlds)) {
		macs := worlds[name]
		vms := map[string]bool{}
		for _, m := range macs {
			for vm := range m.VMs {
				vms[vm] = true
			}
		}
		knows := func(mac string) []string {
			return slices.Sorted(maps.Keys(macs[mac].Preferences.Macs))
		}
		for _, mac := range slices.Sorted(maps.Keys(macs)) {
			if vms[mac] {
				continue
			}
			circle := slices.Sorted(slices.Values(append(knows(mac), mac)))
			for _, peer := range knows(mac) {
				if _, ok := macs[peer]; !ok || vms[peer] {
					continue
				}
				if back := slices.Sorted(slices.Values(append(knows(peer), peer))); !slices.Equal(back, circle) {
					out = append(out, finding{"worlds-macs-know-each-other", "specs/usecases/worlds/" + name + ".json",
						fmt.Sprintf("%s knows %v, %s knows %v", mac, circle, peer, back)})
				}
			}
		}
	}
	return out
}

func TestWorldsMacsKnowEachOther(t *testing.T) {
	worlds := seededWorlds(t, repoRoot(t))
	if len(worlds["two-macs"]["mac-mini"].Preferences.Macs) == 0 {
		t.Fatal("two-macs's mac-mini knows no Mac: the rule proves nothing")
	}
	for _, f := range ruleWorldsMacsKnowEachOther(worlds) {
		t.Errorf("%s: %s", f.where, f.what)
	}
	t.Run("bites", func(t *testing.T) {
		mac := func(vms []string, peers ...string) seededMac {
			var m seededMac
			m.VMs, m.Preferences.Macs = map[string]json.RawMessage{}, map[string]json.RawMessage{}
			for _, vm := range vms {
				m.VMs[vm] = json.RawMessage(`{}`)
			}
			for _, p := range peers {
				m.Preferences.Macs[p] = json.RawMessage(`{}`)
			}
			return m
		}
		oneWay := map[string]map[string]seededMac{"mutant": {
			"macbook-pro": mac([]string{"linux-1"}, "mac-mini"),
			"mac-mini":    mac(nil),
			"linux-1":     mac(nil),
		}}
		if len(ruleWorldsMacsKnowEachOther(oneWay)) == 0 {
			t.Error("a Mac its peer does not know back passed the rule")
		}
		partly := map[string]map[string]seededMac{"mutant": {
			"macbook-pro": mac(nil, "mac-mini", "macbook-air"),
			"mac-mini":    mac(nil, "macbook-pro"),
			"macbook-air": mac(nil, "macbook-pro", "mac-mini"),
		}}
		if len(ruleWorldsMacsKnowEachOther(partly)) == 0 {
			t.Error("a Mac that knows This Mac but not a third Mac passed the rule")
		}
		alone := map[string]map[string]seededMac{"mutant": {"macbook-pro": mac(nil), "mac-mini": mac(nil)}}
		if f := ruleWorldsMacsKnowEachOther(alone); len(f) != 0 {
			t.Errorf("two Macs that know no Mac were refused: %v", f)
		}
	})
}
