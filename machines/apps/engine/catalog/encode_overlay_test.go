package catalog_test

import (
	"bytes"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"go.yaml.in/yaml/v3"
)

// value: n distinct words of the pool, in the pool's order shuffled.
func pick(r *rand.Rand, pool []string, n int) []string {
	p := slices.Clone(pool)
	r.Shuffle(len(p), func(i, j int) { p[i], p[j] = p[j], p[i] })
	return p[:min(n, len(p))]
}

func overlayOf(r *rand.Rand) catalog.Machinefile {
	values := []string{"36", "true", "dark", "yes", "0755", "on", "a b", "vault:token", "-1", "1.5", "null", "~"}
	mf := catalog.Machinefile{Machinefile: catalog.MachinefileSchema, Overrides: pick(r, []string{"web", "linux-dev"}, 1)[0]}
	if r.IntN(3) > 0 {
		names := pick(r, []string{"jq", "fd", "gh", "ripgrep", "bat", "node"}, 1+r.IntN(5))
		cut := r.IntN(len(names) + 1)
		apps := &catalog.AppList{Changes: true}
		for _, n := range names[:cut] {
			e := catalog.AppEntry{Name: n}
			if r.IntN(2) == 0 {
				var v catalog.Range
				if err := yaml.Unmarshal([]byte("'"+pick(r, []string{"1", "24", "1.7", "22.11.3"}, 1)[0]+"'"), &v); err != nil {
					panic(err)
				}
				e.Version, e.Optional = v, r.IntN(2) == 0
			}
			if r.IntN(3) == 0 {
				e.Options = map[string]catalog.OptionValue{"flavor": catalog.OptionValue(pick(r, values, 1)[0])}
			}
			apps.Add = append(apps.Add, e)
		}
		if cut < len(names) {
			apps.Remove = names[cut:]
		}
		mf.Apps = apps
	}
	if r.IntN(3) > 0 {
		names := pick(r, []string{"hushlogin", "macos-dock", "git-config", "dock-spacer"}, 1+r.IntN(4))
		cut := r.IntN(len(names) + 1)
		settings := &catalog.SettingList{Changes: true}
		if cut > 0 {
			settings.Set = map[string]catalog.SettingEntry{}
			for _, n := range names[:cut] {
				e := catalog.SettingEntry{Optional: r.IntN(2) == 0}
				if r.IntN(2) == 0 {
					e.From = catalog.FromComputer
				}
				if r.IntN(2) == 0 {
					e.Values = map[string]catalog.OptionValue{"size": catalog.OptionValue(pick(r, values, 1)[0])}
				}
				settings.Set[n] = e
			}
		}
		if cut < len(names) {
			settings.Remove = names[cut:]
		}
		mf.Settings = settings
	}
	if r.IntN(3) > 0 {
		programs := pick(r, catalog.ProgramName("").Enum(), r.IntN(5))
		cut := r.IntN(len(programs) + 1)
		accounts := &catalog.AccountsSpec{}
		for _, p := range programs[:cut] {
			accounts.SignIns = append(accounts.SignIns, catalog.ProgramName(p))
		}
		for _, p := range programs[cut:] {
			accounts.Grants = append(accounts.Grants, catalog.ProgramName(p))
		}
		mf.Accounts = accounts
	}
	if r.IntN(3) > 0 {
		ports := &catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3999}, {From: 5173, To: 5173}}, To: catalog.AudiencePrivate}
		if r.IntN(2) == 0 {
			ports.Except = []int{3306}
		}
		if r.IntN(2) == 0 {
			ports.Audiences = catalog.PortAudiences{5432: catalog.AudienceLocal, 8080: catalog.AudiencePublic}
		}
		mf.Ports = ports
	}
	return mf
}

// value: a decoded value with the lines its entries were read at left out,
// which an encoding moves.
func unlined(mf catalog.Machinefile) catalog.Machinefile {
	if a := mf.Apps; a != nil {
		c := *a
		c.Add = slices.Clone(a.Add)
		for i := range c.Add {
			c.Add[i].Line = 0
		}
		mf.Apps = &c
	}
	if s := mf.Settings; s != nil && s.Set != nil {
		c := *s
		c.Set = map[string]catalog.SettingEntry{}
		for k, e := range s.Set {
			e.Line = 0
			c.Set[k] = e
		}
		mf.Settings = &c
	}
	return mf
}

// invariant: an overlay station encodes decodes back to the same value,
// and encodes again to the same bytes, over generated overlays.
func TestAnEncodedOverlayDecodesToItself(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 42)) //nolint:gosec // value: a seeded generator, so a failing overlay is found again by its number.
	for i := range 500 {
		mf := overlayOf(r)
		body, err := catalog.EncodeOverlay(mf)
		if err != nil {
			t.Fatalf("overlay %d: %v", i, err)
		}
		back, err := catalog.DecodeMachinefile("~/.config/station/machinefiles/web.local.yaml", body)
		if err != nil {
			t.Fatalf("overlay %d does not decode: %v\n%s", i, err, body)
		}
		if !reflect.DeepEqual(unlined(back), mf) {
			t.Fatalf("overlay %d decodes to\n%+v\nwant\n%+v\n%s", i, unlined(back), mf, body)
		}
		again, err := catalog.EncodeOverlay(back)
		if err != nil || !bytes.Equal(again, body) {
			t.Fatalf("overlay %d encodes again to\n%s\nwant\n%s (%v)", i, again, body, err)
		}
	}
	want := "machinefile: station.machinefile/v1\noverrides: web\napps: {add: [lazygit], remove: [jq]}\nports: {open: [3000-3999], to: private, except: [3306], audiences: {5432: local}}\n"
	got, err := catalog.EncodeOverlay(catalog.Machinefile{
		Overrides: "web", Apps: &catalog.AppList{Add: []catalog.AppEntry{{Name: "lazygit"}}, Remove: []string{"jq"}, Changes: true},
		Ports: &catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3999}}, To: catalog.AudiencePrivate, Except: []int{3306}, Audiences: catalog.PortAudiences{5432: catalog.AudienceLocal}},
	})
	if err != nil || string(got) != want {
		t.Errorf("encoded\n%s\nwant\n%s (%v)", got, want, err)
	}
	if _, err := catalog.EncodeOverlay(catalog.Machinefile{Name: "web"}); err == nil {
		t.Error("a Machinefile that is no overlay encoded")
	}
}
