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
	contracttest.Installer(t, catalog.SkillInstall{Repo: "anthropics/skills", Skill: "frontend-design"}, func(_ *testing.T, s contracttest.Seed) contract.Installer {
		return fake.InstallerOf(seeded(s, catalog.MethodSkill), "mbp", catalog.MethodSkill, false)
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
