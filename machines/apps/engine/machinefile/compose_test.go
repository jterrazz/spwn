package machinefile_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

// value: a mixin named name, its body after its header.
func kit(name, body string) string {
	return "machinefile: station.machinefile/v1\nname: " + name + "\n" + body
}

func files(named map[string]string) map[string]string {
	out := map[string]string{}
	for name, body := range named {
		out["machinefiles/"+name+".machine.yaml"] = body
	}
	return out
}

type scriptSaid struct{ phase, file, lines string }

func scriptsOf(r machinefile.Resolved) []scriptSaid {
	out := make([]scriptSaid, 0, len(r.Scripts))
	for _, s := range r.Scripts {
		out = append(out, scriptSaid{string(s.Phase), s.Origin.Name, s.Lines})
	}
	return out
}

func TestEachFilesRunLinesAreAStepOfTheirOwnInTheChainsOrder(t *testing.T) {
	top := machine("top", "{ os: ubuntu }", "[]")
	for _, tc := range []struct {
		how   string
		files map[string]string
		want  []scriptSaid
	}{
		{"two kits and the machine each say run.boot", map[string]string{
			"a":   kit("a", "run: { boot: echo a }\n"),
			"b":   kit("b", "run: { boot: echo b, image: make b }\n"),
			"top": top + "extends: [a, b]\nrun: { boot: echo top }\n",
		}, []scriptSaid{{"image", "b", "make b"}, {"boot", "a", "echo a"}, {"boot", "b", "echo b"}, {"boot", "top", "echo top"}}},
		{"a diamond reads the kit both extend once", map[string]string{
			"d":   kit("d", "run: { boot: echo d }\n"),
			"b":   kit("b", "extends: [d]\nrun: { boot: echo b }\n"),
			"c":   kit("c", "extends: [d]\nrun: { boot: echo c }\n"),
			"top": top + "extends: [b, c]\n",
		}, []scriptSaid{{"boot", "d", "echo d"}, {"boot", "b", "echo b"}, {"boot", "c", "echo c"}}},
		{"a file that says no phase adds none", map[string]string{
			"a":   kit("a", "apps: [fd]\n"),
			"top": top + "extends: [a]\nrun: { upgrade: echo up }\n",
		}, []scriptSaid{{"upgrade", "top", "echo up"}}},
	} {
		t.Run(tc.how, func(t *testing.T) {
			r, err := machinefile.Resolve(library(t, files(tc.files), nil), "top", machinefile.Machine{}, machinefile.Lock{})
			if err != nil {
				t.Fatal(err)
			}
			if got := scriptsOf(r); !slices.Equal(got, tc.want) {
				t.Errorf("scripts %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestTheChainsAccountsAreTheUnionOfItsFiles(t *testing.T) {
	top := machine("top", "{ os: ubuntu }", "[]")
	for _, tc := range []struct {
		how      string
		files    map[string]string
		overlay  string
		signins  []string
		grants   []string
		refusing string
	}{
		{"two kits each sign in to their own", map[string]string{
			"a":   kit("a", "accounts: { signins: [gh] }\n"),
			"b":   kit("b", "accounts: { signins: [wrangler, gh] }\n"),
			"top": top + "extends: [a, b]\naccounts: { grants: [claude] }\n",
		}, "", []string{"gh=a", "wrangler=b"}, []string{"claude=top"}, ""},
		{"a diamond's kit counts once", map[string]string{
			"d":   kit("d", "accounts: { signins: [gh] }\n"),
			"b":   kit("b", "extends: [d]\n"),
			"c":   kit("c", "extends: [d]\naccounts: { signins: [wrangler] }\n"),
			"top": top + "extends: [b, c]\n",
		}, "", []string{"gh=d", "wrangler=c"}, nil, ""},
		{"an overlay takes a program back", map[string]string{
			"a":   kit("a", "accounts: { signins: [gh, wrangler] }\n"),
			"top": top + "extends: [a]\n",
		}, "overrides: top\naccounts: { signins: [wrangler] }\n", []string{"wrangler=top"}, nil, ""},
		{"a union both signing in to and granting one program", map[string]string{
			"a":   kit("a", "accounts: { signins: [claude] }\n"),
			"b":   kit("b", "accounts: { grants: [claude] }\n"),
			"top": top + "extends: [a, b]\n",
		}, "", nil, nil, "b.machine.yaml: its accounts with those of the files before it in the chain: claude is both signed in and granted"},
	} {
		t.Run(tc.how, func(t *testing.T) {
			user := files(tc.files)
			if tc.overlay != "" {
				user["machinefiles/top.local.yaml"] = "machinefile: station.machinefile/v1\n" + tc.overlay
			}
			r, err := machinefile.Resolve(library(t, user, nil), "top", machinefile.Machine{}, machinefile.Lock{})
			if tc.refusing != "" {
				if err == nil || !strings.Contains(err.Error(), tc.refusing) {
					t.Fatalf("resolve: %v, want refused with %q", err, tc.refusing)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var signins, grants []string
			for _, v := range r.Values {
				if n, ok := strings.CutPrefix(v.Key, "accounts.signins."); ok {
					signins = append(signins, n+"="+v.Origin.Name)
				}
				if n, ok := strings.CutPrefix(v.Key, "accounts.grants."); ok {
					grants = append(grants, n+"="+v.Origin.Name)
				}
			}
			if !slices.Equal(signins, tc.signins) || !slices.Equal(grants, tc.grants) {
				t.Errorf("signins %v and grants %v, want %v and %v", signins, grants, tc.signins, tc.grants)
			}
		})
	}
}

func TestTwoFilesNeitherExtendingTheOtherGiveANameOneWayOrAreRefused(t *testing.T) {
	top := machine("top", "{ os: ubuntu }", "[]")
	for _, tc := range []struct {
		how   string
		files map[string]string
		want  string
		fd    string
	}{
		{"two kits list one app two ways", map[string]string{
			"a":   kit("a", "apps: [{ name: fd, optional: true }]\n"),
			"b":   kit("b", "apps: [fd]\n"),
			"top": top + "extends: [a, b]\n",
		}, "app fd is given two ways, by a (~/.config/station/machinefiles/a.machine.yaml:3) and b (~/.config/station/machinefiles/b.machine.yaml:3), neither extending the other: say it in top, which extends both", ""},
		{"two kits give one setting two ways", map[string]string{
			"a":   kit("a", "settings: { hushlogin: { optional: true } }\n"),
			"b":   kit("b", "settings: { hushlogin: {} }\n"),
			"top": top + "extends: [a, b]\n",
		}, "setting hushlogin is given two ways, by a", ""},
		{"the same entry twice is one", map[string]string{
			"a":   kit("a", "apps: [fd]\n"),
			"b":   kit("b", "apps: [{ name: fd }]\n"),
			"top": top + "extends: [a, b]\n",
		}, "", "b"},
		{"the file that extends both settles it", map[string]string{
			"a":   kit("a", "apps: [{ name: fd, optional: true }]\n"),
			"b":   kit("b", "apps: [fd]\n"),
			"top": machine("top", "{ os: ubuntu }", "[fd]") + "extends: [a, b]\n",
		}, "", "top"},
		{"a child overrides its parent", map[string]string{
			"a":   kit("a", "apps: [{ name: fd, optional: true }]\n"),
			"b":   kit("b", "extends: [a]\napps: [fd]\n"),
			"top": top + "extends: [b]\n",
		}, "", "b"},
		{"a diamond's kit is no conflict with itself", map[string]string{
			"d":   kit("d", "apps: [{ name: fd, optional: true }]\n"),
			"b":   kit("b", "extends: [d]\n"),
			"c":   kit("c", "extends: [d]\n"),
			"top": top + "extends: [b, c]\n",
		}, "", "d"},
		{"a diamond's side overrides the kit both extend", map[string]string{
			"d":   kit("d", "apps: [{ name: fd, optional: true }]\n"),
			"b":   kit("b", "extends: [d]\n"),
			"c":   kit("c", "extends: [d]\napps: [fd]\n"),
			"top": top + "extends: [b, c]\n",
		}, "", "c"},
	} {
		t.Run(tc.how, func(t *testing.T) {
			r, err := machinefile.Resolve(library(t, files(tc.files), nil), "top", machinefile.Machine{}, machinefile.Lock{})
			if tc.want != "" {
				var conflict *machinefile.SiblingConflictError
				if !errors.As(err, &conflict) || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("resolve: %v, want the siblings refused: %s", err, tc.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, v := range r.Values {
				if v.Key == "apps.fd" && v.Origin.Name != tc.fd {
					t.Errorf("fd said by %s, want %s", v.Origin.Name, tc.fd)
				}
			}
		})
	}
}

func TestAnOverlayIsTheWordOfTheFileItChanges(t *testing.T) {
	user := files(map[string]string{
		"a":   kit("a", "apps: [fd]\n"),
		"b":   kit("b", "apps: [{ name: fd, optional: true }]\n"),
		"top": machine("top", "{ os: ubuntu }", "[]") + "extends: [a, b]\n",
	})
	user["machinefiles/top.local.yaml"] = "machinefile: station.machinefile/v1\noverrides: top\napps: { add: [fd] }\n"
	if _, err := machinefile.Resolve(library(t, user, nil), "top", machinefile.Machine{}, machinefile.Lock{}); err != nil {
		t.Errorf("top's overlay saying fd: %v, want it settled as top's word", err)
	}
	delete(user, "machinefiles/top.local.yaml")
	user["machinefiles/a.local.yaml"] = "machinefile: station.machinefile/v1\noverrides: a\napps: { add: [{ name: fd, optional: true }] }\n"
	user["machinefiles/b.machine.yaml"] = kit("b", "apps: [fd]\n")
	var conflict *machinefile.SiblingConflictError
	if _, err := machinefile.Resolve(library(t, user, nil), "top", machinefile.Machine{}, machinefile.Lock{}); !errors.As(err, &conflict) || conflict.First.Name != "a" || conflict.Second.Name != "b" {
		t.Errorf("a's overlay against b: %v, want a and b refused", err)
	}
	user["machinefiles/a.local.yaml"] = "machinefile: station.machinefile/v1\noverrides: a\napps: { remove: [fd] }\n"
	if _, err := machinefile.Resolve(library(t, user, nil), "top", machinefile.Machine{}, machinefile.Lock{}); !errors.As(err, &conflict) || conflict.First.At() != "~/.config/station/machinefiles/a.local.yaml" {
		t.Errorf("a's overlay removing what b lists: %v, want refused at the overlay, which names no line", err)
	}
}

func TestAChainIsAtMostEightDeepAndThirtyTwoFiles(t *testing.T) {
	top := machine("top", "{ os: ubuntu }", "[]")
	deep := func(n int) map[string]string {
		named := map[string]string{"top": top + "extends: [k1]\n"}
		for i := 1; i < n; i++ {
			named["k"+strconv.Itoa(i)] = kit("k"+strconv.Itoa(i), "extends: [k"+strconv.Itoa(i+1)+"]\n")
		}
		named["k"+strconv.Itoa(n)] = kit("k"+strconv.Itoa(n), "")
		return named
	}
	wide := func(n int) map[string]string {
		named := map[string]string{}
		var kits []string
		for i := 1; i <= n; i++ {
			named["k"+strconv.Itoa(i)] = kit("k"+strconv.Itoa(i), "")
			kits = append(kits, "k"+strconv.Itoa(i))
		}
		named["top"] = top + "extends: [" + strings.Join(kits, ", ") + "]\n"
		return named
	}
	for _, tc := range []struct {
		how   string
		files map[string]string
		want  string
	}{
		{"eight deep, itself counted", deep(7), ""},
		{"nine deep", deep(8), "top's chain is more than 8 Machinefiles deep: top extends k1 extends k2 extends k3 extends k4 extends k5 extends k6 extends k7 extends k8"},
		{"thirty-two files", wide(31), ""},
		{"thirty-three files", wide(32), "top's chain reads more than 32 Machinefiles"},
		{"a diamond's file counts once", func() map[string]string {
			named := wide(31)
			named["k1"] = kit("k1", "extends: [k2]\n")
			return named
		}(), ""},
	} {
		t.Run(tc.how, func(t *testing.T) {
			_, err := machinefile.Resolve(library(t, files(tc.files), nil), "top", machinefile.Machine{}, machinefile.Lock{})
			var tooLong *machinefile.ChainTooLongError
			if tc.want == "" && err != nil {
				t.Fatal(err)
			}
			if tc.want != "" && (!errors.As(err, &tooLong) || err.Error() != tc.want) {
				t.Fatalf("resolve: %v, want %s", err, tc.want)
			}
		})
	}
	named := wide(31)
	user := files(named)
	user["machinefiles/k1.local.yaml"] = "machinefile: station.machinefile/v1\noverrides: k1\n"
	if _, err := machinefile.Resolve(library(t, user, nil), "top", machinefile.Machine{}, machinefile.Lock{}); err != nil {
		t.Errorf("thirty-two files and an overlay: %v, want an overlay not counted", err)
	}
}

func TestEachFilesRunLinesAreAValueOfTheirOwnInExplain(t *testing.T) {
	lib := library(t, files(map[string]string{
		"a":   kit("a", "run: { boot: echo a }\n"),
		"top": machine("top", "{ os: ubuntu }", "[]") + "extends: [a]\nrun: { boot: echo top }\n",
	}), nil)
	r, err := machinefile.Resolve(lib, "top", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	var got []scriptSaid
	for _, v := range r.Values {
		if v.Key == "run.boot" {
			got = append(got, scriptSaid{v.Key, v.Origin.Name, v.Text})
		}
	}
	if want := []scriptSaid{{"run.boot", "a", "echo a"}, {"run.boot", "top", "echo top"}}; !slices.Equal(got, want) {
		t.Errorf("run.boot values %+v, want %+v", got, want)
	}
}
