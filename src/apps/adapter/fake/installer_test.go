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
