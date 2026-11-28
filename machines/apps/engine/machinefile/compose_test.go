package machinefile_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
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
		{"the top file's empty phase clears every file's lines of it", map[string]string{
			"a":   kit("a", "run: { boot: echo a, upgrade: apt-get upgrade -y }\n"),
			"b":   kit("b", "run: { upgrade: echo b }\n"),
			"top": top + "extends: [a, b]\nrun: { upgrade: \"\" }\n",
		}, []scriptSaid{{"boot", "a", "echo a"}}},
		{"a kit's empty phase clears every file's before it, a sibling's included", map[string]string{
			"d":   kit("d", "run: { boot: echo d }\n"),
			"b":   kit("b", "extends: [d]\nrun: { boot: \"\" }\n"),
			"c":   kit("c", "run: { boot: echo c }\n"),
			"top": top + "extends: [c, b]\n",
		}, nil},
		{"a kit's empty phase leaves a sibling after it", map[string]string{
			"d":   kit("d", "run: { boot: echo d }\n"),
			"b":   kit("b", "extends: [d]\nrun: { boot: \"\" }\n"),
			"c":   kit("c", "run: { boot: echo c }\n"),
			"top": top + "extends: [b, c]\n",
		}, []scriptSaid{{"boot", "c", "echo c"}}},
		{"a file after the clearing gives its own lines", map[string]string{
			"d":   kit("d", "run: { boot: echo d }\n"),
			"b":   kit("b", "extends: [d]\nrun: { boot: \"\" }\n"),
			"top": top + "extends: [b]\nrun: { boot: echo top }\n",
		}, []scriptSaid{{"boot", "top", "echo top"}}},
		{"the top file's overlay clears every file's lines of a phase", map[string]string{
			"a":         kit("a", "run: { boot: echo a }\n"),
			"top":       top + "extends: [a]\nrun: { boot: echo top }\n",
			"top.local": "machinefile: station.machinefile/v1\noverrides: top\nrun: { boot: \"\" }\n",
		}, nil},
	} {
		t.Run(tc.how, func(t *testing.T) {
			user := files(tc.files)
			if o, ok := tc.files["top.local"]; ok {
				delete(user, "machinefiles/top.local.machine.yaml")
				user["machinefiles/top.local.yaml"] = o
			}
			r, err := machinefile.Resolve(library(t, user, nil), "top", machinefile.Machine{}, machinefile.Lock{})
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
		{"two kits, one signing in to a program the other grants", map[string]string{
			"a":   kit("a", "accounts: { signins: [claude] }\n"),
			"b":   kit("b", "accounts: { grants: [claude] }\n"),
			"top": top + "extends: [a, b]\n",
		}, "", nil, nil, "account claude is given two ways, by a (~/.config/station/machinefiles/a.machine.yaml) and b (~/.config/station/machinefiles/b.machine.yaml), neither extending the other: say it in top, which extends both"},
		{"the file extending both settles it", map[string]string{
			"a":   kit("a", "accounts: { signins: [claude] }\n"),
			"b":   kit("b", "accounts: { grants: [claude] }\n"),
			"top": top + "extends: [a, b]\naccounts: { grants: [claude] }\n",
		}, "", nil, []string{"claude=b"}, ""},
		{"its overlay settles it", map[string]string{
			"a":   kit("a", "accounts: { signins: [claude] }\n"),
			"b":   kit("b", "accounts: { grants: [claude] }\n"),
			"top": top + "extends: [a, b]\n",
		}, "overrides: top\naccounts: { signins: [claude] }\n", []string{"claude=top"}, nil, ""},
		{"a child moves what its parent signs in to", map[string]string{
			"a":   kit("a", "accounts: { signins: [gh, wrangler] }\n"),
			"top": top + "extends: [a]\naccounts: { grants: [wrangler] }\n",
		}, "", []string{"gh=a"}, []string{"wrangler=top"}, ""},
		{"an empty list clears what the files before it gave", map[string]string{
			"a":   kit("a", "accounts: { signins: [gh], grants: [claude] }\n"),
			"b":   kit("b", "extends: [a]\naccounts: { signins: [] }\n"),
			"top": top + "extends: [b]\naccounts: { signins: [wrangler] }\n",
		}, "", []string{"wrangler=top"}, []string{"claude=a"}, ""},
		{"an empty list clears a sibling's before it too", map[string]string{
			"a":   kit("a", "accounts: { signins: [gh] }\n"),
			"b":   kit("b", "accounts: { signins: [] }\n"),
			"top": top + "extends: [a, b]\n",
		}, "", nil, nil, ""},
	} {
		t.Run(tc.how, func(t *testing.T) {
			user := files(tc.files)
			if tc.overlay != "" {
				user["machinefiles/top.local.yaml"] = "machinefile: station.machinefile/v1\n" + tc.overlay
			}
			r, err := machinefile.Resolve(library(t, user, nil), "top", machinefile.Machine{}, machinefile.Lock{})
			if tc.refusing != "" {
				var conflict *machinefile.SiblingConflictError
				if !errors.As(err, &conflict) || err.Error() != tc.refusing {
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
			"a":   kit("a", "apps: [{ name: fd, version: '1' }]\n"),
			"b":   kit("b", "apps: [fd]\n"),
			"top": top + "extends: [a, b]\n",
		}, "app fd is given two ways, by a (~/.config/station/machinefiles/a.machine.yaml:3) and b (~/.config/station/machinefiles/b.machine.yaml:3), neither extending the other: say it in top, which extends both", ""},
		{"two kits ask two versions", map[string]string{
			"a":   kit("a", "apps: [{ name: fd, version: '1' }]\n"),
			"b":   kit("b", "apps: [{ name: fd, version: '2' }]\n"),
			"top": top + "extends: [a, b]\n",
		}, "app fd is given two ways, by a", ""},
		{"two kits give one setting two ways", map[string]string{
			"a":   kit("a", "settings: { hushlogin: { from: computer } }\n"),
			"b":   kit("b", "settings: { hushlogin: {} }\n"),
			"top": top + "extends: [a, b]\n",
		}, "setting hushlogin is given two ways, by a", ""},
		{"the same entry twice is one", map[string]string{
			"a":   kit("a", "apps: [fd]\n"),
			"b":   kit("b", "apps: [{ name: fd }]\n"),
			"top": top + "extends: [a, b]\n",
		}, "", "b"},
		{"latest is no version said", map[string]string{
			"a":   kit("a", "apps: [{ name: fd, version: latest }]\n"),
			"b":   kit("b", "apps: [fd]\n"),
			"top": top + "extends: [a, b]\n",
		}, "", "b"},
		{"optional is no other way", map[string]string{
			"a":   kit("a", "apps: [fd]\n"),
			"b":   kit("b", "apps: [{ name: fd, optional: true }]\n"),
			"top": top + "extends: [a, b]\n",
		}, "", "b"},
		{"the file that extends both settles it", map[string]string{
			"a":   kit("a", "apps: [{ name: fd, version: '1' }]\n"),
			"b":   kit("b", "apps: [fd]\n"),
			"top": machine("top", "{ os: ubuntu }", "[fd]") + "extends: [a, b]\n",
		}, "", "top"},
		{"a child overrides its parent", map[string]string{
			"a":   kit("a", "apps: [{ name: fd, version: '1' }]\n"),
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
			"d":   kit("d", "apps: [{ name: fd, version: '1' }]\n"),
			"b":   kit("b", "extends: [d]\n"),
			"c":   kit("c", "extends: [d]\napps: [{ name: fd, optional: true }]\n"),
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
		"b":   kit("b", "apps: [{ name: fd, version: '1' }]\n"),
		"top": machine("top", "{ os: ubuntu }", "[]") + "extends: [a, b]\n",
	})
	user["machinefiles/top.local.yaml"] = "machinefile: station.machinefile/v1\noverrides: top\napps: { add: [fd] }\n"
	if _, err := machinefile.Resolve(library(t, user, nil), "top", machinefile.Machine{}, machinefile.Lock{}); err != nil {
		t.Errorf("top's overlay saying fd: %v, want it settled as top's word", err)
	}
	delete(user, "machinefiles/top.local.yaml")
	user["machinefiles/a.local.yaml"] = "machinefile: station.machinefile/v1\noverrides: a\napps: { add: [{ name: fd, version: '1' }] }\n"
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

func TestARefusedChainIsNoTemplateAndExplainsNothing(t *testing.T) {
	lib := library(t, files(map[string]string{
		"a":    kit("a", "accounts: { signins: [claude] }\n"),
		"b":    kit("b", "accounts: { grants: [claude] }\n"),
		"both": kit("both", "extends: [a, b]\n"),
		"top":  machine("top", "{ os: ubuntu }", "[]") + "extends: [a, b]\nvm: { image: { from: base-ubuntu } }\n",
		"ok":   machine("ok", "{ os: ubuntu }", "[]") + "extends: [a]\nvm: { image: { from: base-ubuntu } }\n",
	}), nil)
	var conflict *machinefile.SiblingConflictError
	if _, err := machinefile.TemplateOf(lib, "top", nil); !errors.As(err, &conflict) {
		t.Errorf("the template of a refused chain: %v, want the siblings refused", err)
	}
	if got := lib.Templates(); !slices.Equal(got, []string{"ok"}) {
		t.Errorf("templates %v, want ok alone", got)
	}
	if _, err := machinefile.Own(lib, "both"); !errors.As(err, &conflict) {
		t.Errorf("a mixin's own values of a refused chain: %v, want the siblings refused", err)
	}
}

func TestSiblingsAreComparedByWhatTheirEntriesMean(t *testing.T) {
	base := machine("base-ubuntu", "{ os: ubuntu }", "[{ name: fd, optional: true }]")
	lib := library(t, files(map[string]string{
		"git-kit": kit("git-kit", "apps: [fd]\n"),
		"top":     machine("top", "{ os: ubuntu }", "[]") + "extends: [git-kit, base-ubuntu]\n",
	}), map[string]string{"machinefiles/base-ubuntu.machine.yaml": base})
	r, err := machinefile.Resolve(lib, "top", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatalf("a built-in base marking fd optional beside a kit requiring it: %v, want it resolved", err)
	}
	if i := slices.IndexFunc(r.Steps, func(s machinefile.Step) bool { return s.Package == "fd" }); i < 0 || r.Steps[i].Optional {
		t.Errorf("fd's step %+v, want it required, as the kit requires it", r.Steps)
	}
	lib = library(t, files(map[string]string{
		"a":   kit("a", "settings: { hushlogin: { optional: true } }\n"),
		"b":   kit("b", "settings: { hushlogin: {} }\n"),
		"top": machine("top", "{ os: ubuntu }", "[]") + "extends: [a, b]\n",
	}), nil)
	if _, err := machinefile.GivenOf(lib, "top", ""); err != nil {
		t.Errorf("a setting optional in one kit and not in another: %v, want it read", err)
	}
	top := machine("top", "{ os: macos, version: '26' }", "[]") + "extends: [a, b]\n"
	for _, tc := range []struct {
		how, a  string
		refused bool
	}{
		{"the manager fd takes by default there", "apps: [{ name: fd, manager: skill }]\n", false},
		{"another manager", "apps: [{ name: fd, manager: cask }]\n", true},
	} {
		lib := library(t, files(map[string]string{"a": kit("a", tc.a), "b": kit("b", "apps: [fd]\n"), "top": top}), nil)
		_, err := machinefile.Resolve(lib, "top", machinefile.Machine{}, machinefile.Lock{})
		var conflict *machinefile.SiblingConflictError
		if errors.As(err, &conflict) != tc.refused || !tc.refused && err != nil {
			t.Errorf("%s beside a bare fd: %v, want refused %v", tc.how, err, tc.refused)
		}
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
		{"thirty-three files", wide(32), "top's chain reads more than 32 Machinefiles: top is one past them"},
		{"a diamond's file counts once", func() map[string]string {
			named := wide(31)
			named["k1"] = kit("k1", "extends: [k2]\n")
			return named
		}(), ""},
		{"a file reached again deeper is judged at that depth", func() map[string]string {
			named := deep(6)
			named["top"], named["x"] = top+"extends: [k2, x]\n", kit("x", "extends: [k1]\n")
			return named
		}(), ""},
		{"a file reached again past the depth", func() map[string]string {
			named := deep(7)
			named["top"], named["x"] = top+"extends: [k2, x]\n", kit("x", "extends: [k1]\n")
			return named
		}(), "top's chain is more than 8 Machinefiles deep: top extends x extends k1 extends k2 extends k3 extends k4 extends k5 extends k6 extends k7"},
		{"a cycle longer than the depth is a cycle", func() map[string]string {
			named := deep(9)
			named["k9"] = kit("k9", "extends: [k1]\n")
			return named
		}(), "cycle"},
	} {
		t.Run(tc.how, func(t *testing.T) {
			_, err := machinefile.Resolve(library(t, files(tc.files), nil), "top", machinefile.Machine{}, machinefile.Lock{})
			var tooLong *machinefile.ChainTooLongError
			var cycle *machinefile.CycleError
			switch {
			case tc.want == "" && err != nil:
				t.Fatal(err)
			case tc.want == "cycle":
				if !errors.As(err, &cycle) || cycle.Said() != "k1 extends k2 extends k3 extends k4 extends k5 extends k6 extends k7 extends k8 extends k9 extends k1" {
					t.Fatalf("resolve: %v, want the cycle around its nine files", err)
				}
			case tc.want != "" && (!errors.As(err, &tooLong) || err.Error() != tc.want):
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

func TestAChainOfDiamondsIsWalkedOncePerFile(t *testing.T) {
	named := map[string]string{}
	level := func(i int) []string {
		out := make([]string, 0, 4)
		for _, side := range []string{"a", "b", "c", "d"} {
			out = append(out, "l"+strconv.Itoa(i)+side)
		}
		return out
	}
	for i := 1; i <= 7; i++ {
		for _, n := range level(i) {
			body := "ports: { open: [" + strconv.Itoa(1000+i) + "], to: private }\n"
			if i < 7 {
				body += "extends: [" + strings.Join(level(i+1), ", ") + "]\n"
			}
			named[n] = kit(n, body)
		}
	}
	named["top"] = machine("top", "{ os: ubuntu }", "[]") + "extends: [" + strings.Join(level(1), ", ") + "]\n"
	user := files(named)
	r, err := machinefile.Resolve(library(t, user, nil), "top", machinefile.Machine{}, machinefile.Lock{})
	if err != nil || len(r.Files) != 29 {
		t.Fatalf("eight deep, four wide, each extending the four below: %v, %d files, want 29", err, len(r.Files))
	}
	before := catalog.Decoded()
	rights, by, err := machinefile.RightsOf([]machinefile.Tree{tree(machinefile.User, user)}, "top", "")
	if n := catalog.Decoded() - before; err != nil || n != 29 || by != "l1d" || rights == nil {
		t.Errorf("rights of the same chain: %v by %s, %v, %d files decoded, want each of the 29 once and l1d's", rights, by, err, n)
	}
	named["top"] = machine("top", "{ os: ubuntu }", "[]") + "extends: [l1a, l1a]\n"
	if _, _, err := machinefile.RightsOf([]machinefile.Tree{tree(machinefile.User, files(named))}, "top", ""); err == nil || !strings.Contains(err.Error(), "extends l1a twice") {
		t.Errorf("a file extending one kit twice: %v, want it refused", err)
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

func TestAClearedPhaseIsSaidInExplainAndAnOverlayGivesNoLines(t *testing.T) {
	user := files(map[string]string{
		"a":   kit("a", "run: { upgrade: apt-get upgrade -y }\n"),
		"top": machine("top", "{ os: ubuntu }", "[]") + "extends: [a]\nrun: { boot: echo top }\n",
	})
	user["machinefiles/top.local.yaml"] = "machinefile: station.machinefile/v1\noverrides: top\nrun: { upgrade: \"\" }\n"
	r, err := machinefile.Resolve(library(t, user, nil), "top", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	var got []scriptSaid
	for _, v := range r.Values {
		if strings.HasPrefix(v.Key, "run.") {
			got = append(got, scriptSaid{v.Key, v.Origin.Name, v.Text})
		}
	}
	if want := []scriptSaid{{"run.boot", "top", "echo top"}, {"run.upgrade", "top", "cleared by top"}}; !slices.Equal(got, want) || len(r.Scripts) != 1 {
		t.Errorf("run values %+v and %d scripts, want %+v and top's boot alone to run", got, len(r.Scripts), want)
	}
	if _, err := catalog.DecodeMachinefile("top.local.yaml", []byte("machinefile: station.machinefile/v1\noverrides: top\nrun: { boot: echo mine }\n")); err == nil || !strings.Contains(err.Error(), "an overlay clears a phase's lines") {
		t.Errorf("an overlay giving run lines: %v, want it refused", err)
	}
	if _, err := catalog.DecodeMachinefile("a.machine.yaml", []byte(kit("a", "run: {}\n"))); err == nil || !strings.Contains(err.Error(), "run names the lines of a phase") {
		t.Errorf("a run naming no phase: %v, want it refused", err)
	}
}
