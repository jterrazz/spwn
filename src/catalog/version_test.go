package catalog_test

import (
	"reflect"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
)

// value: every form a version takes, as a Machinefile writes it.
var forms = []string{"", "latest", "24", "24.11", "24.11.3", "^24.2.0", "~24.11.0", "^0.2.3", "~24", "=1:2.43.0-1ubuntu7"}

func TestAVersionIsReadIntoItsRange(t *testing.T) {
	for text, want := range map[string]struct {
		in, out []string
		head    string
		npm     string
	}{
		"":                   {in: []string{"1.0", "lts"}, head: "", npm: ""},
		"latest":             {in: []string{"25.1.0"}, head: "", npm: ""},
		"24":                 {in: []string{"24.0.0", "24.14.2", "v24.1.0"}, out: []string{"25.0.0", "23.9.9", "240.1", "lts"}, head: "24", npm: "^24.0.0"},
		"24.11":              {in: []string{"24.11.0", "24.11.9"}, out: []string{"24.12.0", "24.10.9"}, head: "24.11", npm: "~24.11.0"},
		"24.11.3":            {in: []string{"24.11.3", "v24.11.3"}, out: []string{"24.11.4", "24.11"}, head: "24.11.3", npm: "24.11.3"},
		"^24.2.0":            {in: []string{"24.2.0", "24.9.1"}, out: []string{"24.1.9", "25.0.0"}, head: "24", npm: "^24.2.0"},
		"~24.11.0":           {in: []string{"24.11.5"}, out: []string{"24.12.0"}, head: "24.11", npm: "~24.11.0"},
		"^0.2.3":             {in: []string{"0.2.9"}, out: []string{"0.3.0", "0.2.2"}, head: "0.2", npm: "^0.2.3"},
		"^0.0.3":             {in: []string{"0.0.3"}, out: []string{"0.0.4"}, head: "0.0.3", npm: "^0.0.3"},
		"~24":                {in: []string{"24.9.0"}, out: []string{"25.0.0"}, head: "24", npm: "~24.0.0"},
		"=1:2.43.0-1ubuntu7": {in: []string{"1:2.43.0-1ubuntu7"}, out: []string{"2.43.0"}, head: "1:2.43.0-1ubuntu7", npm: "1:2.43.0-1ubuntu7"},
	} {
		r, err := catalog.ParseRange(text)
		if err != nil {
			t.Fatalf("ParseRange(%q): %v", text, err)
		}
		for _, v := range want.in {
			if !r.Contains(v) {
				t.Errorf("%q holds no %q; want it held", text, v)
			}
		}
		for _, v := range want.out {
			if r.Contains(v) {
				t.Errorf("%q holds %q; want it out", text, v)
			}
		}
		if r.Head() != want.head || r.Npm() != want.npm {
			t.Errorf("%q: head %q, npm %q; want %q, %q", text, r.Head(), r.Npm(), want.head, want.npm)
		}
		back, err := catalog.ParseRange(r.Spec())
		if err != nil || back.Spec() != r.Spec() {
			t.Errorf("%q: its spec %q reads back as %q (%v)", text, r.Spec(), back.Spec(), err)
		}
	}
}

func TestAWordThatIsNoVersionIsRefusedNamingTheForms(t *testing.T) {
	for _, text := range []string{"lts", "24.x", ">=24", "^", "24.11.3.1", "1 2", "=", "=lts"} {
		if _, err := catalog.ParseRange(text); err == nil || err.Error() != `version "`+text+`" is no version: `+catalog.RangeForms {
			t.Errorf("ParseRange(%q) = %v; want refused naming the forms", text, err)
		}
	}
}

func TestANewerVersionPastTheRangeWaitsWithTheLineMoved(t *testing.T) {
	for _, c := range []struct{ asked, newest, moved string }{
		{"24", "25.1.0", "25"},
		{"24.11", "24.14.0", "24.14"},
		{"^24.2.0", "25.1.0", "^25.1.0"},
		{"24.11.3", "24.11.4", "24.11.4"},
	} {
		r, _ := catalog.ParseRange(c.asked)
		if !r.Past(c.newest) || r.Moved(c.newest) != c.moved {
			t.Errorf("%q past %q: %v, moved %q; want past, %q", c.asked, c.newest, r.Past(c.newest), r.Moved(c.newest), c.moved)
		}
	}
	for _, c := range [][2]string{{"", "99.0.0"}, {"24", "24.14.0"}, {"24", "lts"}} {
		r, _ := catalog.ParseRange(c[0])
		if r.Past(c[1]) {
			t.Errorf("%q: %q is past it; want within, or unsaid", c[0], c[1])
		}
	}
}

// invariant: every method meets every form of version: it holds the
// range (HoldOf) and its install carries it in its manager's own word, or
// it holds none and Pinned refuses it; no pair is left unjudged.
func TestEveryMethodMeetsEveryVersionForm(t *testing.T) {
	installs := map[catalog.Method]catalog.Install{
		catalog.MethodBrew:          catalog.BrewInstall{Formula: "jq"},
		catalog.MethodCask:          catalog.CaskInstall{Token: "ghostty"},
		catalog.MethodMas:           catalog.MasInstall{Name: "Xcode", ID: 497799835},
		catalog.MethodNpm:           catalog.NpmInstall{Package: "pnpm"},
		catalog.MethodUv:            catalog.UvInstall{Package: "ruff"},
		catalog.MethodBun:           catalog.BunInstall{Package: "qmd"},
		catalog.MethodMise:          catalog.MiseInstall{App: "node", Version: "lts"},
		catalog.MethodApt:           catalog.AptInstall{Packages: []string{"git"}},
		catalog.MethodGitHubRelease: catalog.GitHubReleaseInstall{Repo: "o/r", Asset: "r.tar.gz"},
		catalog.MethodScript:        catalog.ScriptInstall{URL: "https://example.com/i.sh"},
		catalog.MethodShell:         catalog.ShellInstall{Line: "true"},
		catalog.MethodApp:           catalog.AppInstall{Hint: "the App Store"},
		catalog.MethodSkill:         catalog.SkillInstall{Repo: "o/r", Skill: "s"},
		catalog.MethodProvidedBy:    catalog.ProvidedByInstall{Item: "node"},
	}
	c, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range c.Items {
		if it.Item.Kind == catalog.KindSetting {
			continue
		}
		for _, m := range it.Item.Methods {
			if _, ok := installs[catalog.Method(m.Method())]; !ok {
				t.Errorf("%s's method %s is not enumerated here", it.Item.Name, m.Method())
			}
		}
	}
	for method, in := range installs {
		for _, text := range forms {
			r, err := catalog.ParseRange(text)
			if err != nil {
				t.Fatal(err)
			}
			pinned, ok := catalog.Pinned(in, r)
			holds := catalog.HoldOf(in).Holds(r)
			switch {
			case ok != holds:
				t.Errorf("%s at %q: pinned %v, holds %v; want the same", method, text, ok, holds)
			case !ok:
			case r.Any() && method == catalog.MethodMise && catalog.VersionOf(pinned) != "latest":
				t.Errorf("mise at latest carries %q; want latest, never the catalog's lts", catalog.VersionOf(pinned))
			case r.Any() && method != catalog.MethodMise && !reflect.DeepEqual(pinned, in):
				t.Errorf("%s at latest carries %v; want the catalog's own install", method, pinned)
			case !r.Any() && catalog.VersionOf(pinned) != r.Head() && catalog.VersionOf(pinned) != r.Npm():
				t.Errorf("%s at %q carries %q to its manager; want its prefix %q or its range %q", method, text, catalog.VersionOf(pinned), r.Head(), r.Npm())
			}
		}
	}
}

func TestAPrereleaseIsNeitherHeldByARangeNorWaiting(t *testing.T) {
	r, _ := catalog.ParseRange("25")
	if r.Contains("25.1.0-rc.1") || !r.Contains("25.1.0+build.2") {
		t.Errorf("25 holds 25.1.0-rc.1: %v, 25.1.0+build.2: %v; want neither the prerelease, the build", r.Contains("25.1.0-rc.1"), r.Contains("25.1.0+build.2"))
	}
	if r24, _ := catalog.ParseRange("24"); r24.Past("26.0.0-rc.1") {
		t.Error("26.0.0-rc.1 waits past 24; want a prerelease never waiting")
	}
	if got := catalog.Newest("24.1.0-rc.1", "24.1.0", "24.0.9"); got != "24.1.0" {
		t.Errorf("newest %q; want the release before its prerelease", got)
	}
}

func TestAVendorBeforeTheNumbersIsReadThrough(t *testing.T) {
	r, _ := catalog.ParseRange("21")
	if !r.Contains("temurin-21.0.13+9.0.LTS") || r.Contains("temurin-27.0.1+0") || !r.Past("temurin-27.0.1+0") {
		t.Error("21 does not read temurin's versions by their numbers")
	}
	for _, c := range []struct {
		exact, version string
		holds          bool
	}{
		{"9.8.7", "temurin-9.8.7", true}, {"v9.8.7", "temurin-v9.8.7", true}, {"9.8.7", "temurin-9.8.8", false},
		{"temurin-9.8.7", "temurin-9.8.7", true}, {"temurin-9.8.7", "zulu-9.8.7", false}, {"temurin-9.8.7", "9.8.7", false},
	} {
		if got := catalog.Exactly(c.exact).Contains(c.version); got != c.holds {
			t.Errorf("exactly %s holds %s: %v; want %v", c.exact, c.version, got, c.holds)
		}
	}
	for _, c := range []struct{ catalog, text, want string }{
		{"lts", "", "latest"}, {"lts", "latest", "latest"}, {"stable", "latest", "stable"}, {"latest", "24", "24"},
		{"temurin-21", "latest", "temurin"}, {"temurin-21", "21", "temurin-21"}, {"temurin-21", "21.0", "temurin-21.0"},
		{"temurin-21", "=temurin-21.0.13+9.0.LTS", "temurin-21.0.13+9.0.LTS"},
	} {
		r, _ := catalog.ParseRange(c.text)
		if got := catalog.MiseAt(c.catalog, r); got != c.want {
			t.Errorf("mise at %q of %q: %q; want %q", c.text, c.catalog, got, c.want)
		}
	}
}
