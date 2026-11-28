package machinefile_test

import (
	"slices"
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
