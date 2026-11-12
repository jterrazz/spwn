package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/src/apps/adapter/fake"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/world"
)

func seeded(s contracttest.Seed, m catalog.Method) *world.World {
	w := &world.World{ThisMac: "mbp"}
	w.Mac("mbp").Apps = map[catalog.Method]*world.Manager{m: {Absent: s.Absent, Installed: s.Installed}}
	if s.Fails {
		w.Script = []world.Fault{{Call: "Installer.Install"}, {Call: "Installer.Update"}, {Call: "Installer.Remove"}, {Call: "Bundler.Bundle"}}
	}
	return w
}

func TestInstaller(t *testing.T) {
	samples := []catalog.Install{
		catalog.SkillInstall{Repo: "anthropics/skills", Skill: "frontend-design"},
		catalog.ScriptInstall{URL: "https://claude.ai/install.sh"},
		catalog.ScriptInstall{URL: "https://claude.ai/install.sh", Undo: "rm ~/.local/bin/tool"},
		catalog.ShellInstall{Line: "npx playwright install chromium"},
		catalog.ShellInstall{Line: "make tool", Undo: "make untool"},
		catalog.AppInstall{Hint: "download it from its website"},
		catalog.ProvidedByInstall{Item: "desk-app", Commands: []string{"dk"}},
		catalog.GitHubReleaseInstall{Repo: "acme/tool", Asset: "tool-*-{os}-{arch}.tar.gz", Commands: []string{"tool"}},
	}
	for _, sample := range samples {
		t.Run(string(sample.Method()), func(t *testing.T) {
			contracttest.Installer(t, sample, func(_ *testing.T, s contracttest.Seed) contract.Installer {
				return fake.InstallerOf(seeded(s, sample.Method()), "mbp", sample.Method(), false)
			})
		})
	}
}

func TestNewest(t *testing.T) {
	sample := catalog.GitHubReleaseInstall{Repo: "acme/tool", Asset: "tool-*-{os}-{arch}.tar.gz", Commands: []string{"tool"}}
	contracttest.Newest(t, sample, func(_ *testing.T, s contracttest.Seed) contract.Installer {
		w := seeded(s, sample.Method())
		w.Mac("mbp").Apps[sample.Method()].ReadsNewest = true
		return fake.InstallerOf(w, "mbp", sample.Method(), false)
	})
}

func TestHoldingInstaller(t *testing.T) {
	build := func(_ *testing.T, s contracttest.Seed) contract.Installer {
		return fake.InstallerOf(seeded(s, catalog.MethodApt), "mbp", catalog.MethodApt, false)
	}
	contracttest.Installer(t, catalog.AptInstall{Packages: []string{"fd-find"}}, build)
	contracttest.Holder(t, catalog.AptInstall{Packages: []string{"fd-find"}}, build)
}

func TestLinedInstaller(t *testing.T) {
	contracttest.Installer(t, catalog.CaskInstall{Token: "ghostty"}, func(_ *testing.T, s contracttest.Seed) contract.Installer {
		return fake.InstallerOf(seeded(s, catalog.MethodCask), "mbp", catalog.MethodCask, true)
	})
}

func TestBundler(t *testing.T) {
	contracttest.Bundler(t, catalog.CaskInstall{Token: "ghostty"}, func(_ *testing.T, s contracttest.Seed) contracttest.Bundled {
		w := seeded(s, catalog.MethodCask)
		return contracttest.Bundled{Bundler: fake.BundlerOf(w, "mbp"), Installer: fake.InstallerOf(w, "mbp", catalog.MethodCask, true)}
	})
}

// invariant: the fake resolves a runtime's word as its real manager does
// (gate probe, mise 2026.11.5), for the installs a Machinefile line makes
// of the catalog's items: node's channel holds an even major, never a newer
// odd one, latest every major, a vendor its own versions alone, and an
// exact version it does not offer is held as it is written.
func TestAVersionWordIsResolvedAsItsManagerResolvesIt(t *testing.T) {
	cat, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ app, version, want string }{
		{"node", "", "24.14.0"}, {"node", "latest", "25.1.0"}, {"node", "22", "22.14.0"},
		{"java", "latest", "temurin-27.0.1+0"}, {"java", "21", "temurin-21.0.13+9.0.LTS"}, {"java", "9.8.7", "temurin-9.8.7"},
	} {
		a, _ := cat.App(c.app)
		in, ok := a.InstallOn(catalog.Mac)
		if !ok {
			t.Fatalf("%s installs on no Mac", c.app)
		}
		if c.version != "" {
			r, err := catalog.ParseRange(c.version)
			if err != nil {
				t.Fatal(err)
			}
			if in, ok = catalog.Pinned(in, r); !ok {
				t.Fatalf("%s holds no %s", c.app, c.version)
			}
		}
		w := &world.World{ThisMac: "mbp"}
		w.Mac("mbp").Apps = map[catalog.Method]*world.Manager{in.Method(): {Installed: map[string]string{}, Versions: map[string][]string{
			"node": {"22.14.0", "24.14.0", "25.1.0"}, "java": {"21.0.2+13", "temurin-21.0.13+9.0.LTS", "temurin-27.0.1+0", "27.0.1+0"},
		}}}
		f := fake.InstallerOf(w, "mbp", in.Method(), false)
		if err := f.Install(t.Context(), []catalog.Install{in}, nil); err != nil {
			t.Fatal(err)
		}
		if got := w.Mac("mbp").Apps[in.Method()].Installed[in.Key()]; got != c.want {
			t.Errorf("%s at %q holds %q; want %q", c.app, c.version, got, c.want)
		}
	}
}
