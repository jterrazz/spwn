package catalog_test

import (
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestEachKindOfNameIsCheckedByOneRule(t *testing.T) {
	for _, c := range []struct {
		check      func(string) error
		good, evil []string
	}{
		{catalog.CheckPackage, []string{"jq", "fd-find", "g++", "libc6:arm64", "python3.12"},
			[]string{"", "j", "-oDPkg::Pre-Invoke::=id", "--allow-unauthenticated", "jq curl", "Jq", "../jq", "jq/x", ".jq", "jq=1.0"}},
		{catalog.CheckPackageVersion, []string{"1.7.1-1ubuntu1", "2:9.1.0016-1", "1.0~rc1+dfsg"},
			[]string{"", "-o", "1.0 jq", "1.0\njq", "=1.0", "*"}},
		{catalog.CheckCommand, []string{"tool", "rust-analyzer", "fd", "g++", "python3.12", ".tool"},
			[]string{"", ".", "..", "../../.profile", "../tool", "/etc/passwd", "a/b", `a\b`, "-tool", "tool\x00"}},
		{catalog.CheckRepo, []string{"acme/tool", "junegunn/fzf", "BurntSushi/ripgrep", "a/b.c_d-e"},
			[]string{"", "gh", "a/../../../../../tmp/x", "../x", "a/..", "a/.", "-a/b", "a/b/c", "a//b", "a/b c", "/a/b", "a/b?x=1", "a/b#x"}},
		{catalog.CheckTag, []string{"v1.2.3", "1.4.4", "release-2024.01", "v1.0.0-rc.1", "1.0+build5", "nightly_2"},
			[]string{"", "../../../../repos/evil/x/releases/latest", "..", "v1..2", "a/b", "/v1", "v1?x=1", "v1#x", "v1 -x", " v1", "-v1", ".v1", "v1\n", "v1%2f"}},
		{catalog.CheckSkill, []string{"frontend-design", "last30days", "vercel-react-best-practices", "a.b_c"},
			[]string{"", "-x", "--help", "a b", "a/b", "../x", "a..b", ".x", "x\n", "x;id"}},
	} {
		for _, n := range c.good {
			if err := c.check(n); err != nil {
				t.Errorf("%q refused: %v", n, err)
			}
		}
		for _, n := range c.evil {
			if err := c.check(n); err == nil {
				t.Errorf("%q passes", n)
			}
		}
	}
}

func TestAPackageWhoseNameEscapesItsRuleIsRefused(t *testing.T) {
	release := strings.Replace(gh, "brew: gh", "github-release: {repo: acme/tool, asset: 'tool-{os}-{arch}'}", 1)
	apt := strings.Replace(gh, "    - os: macos\n      brew: gh", "    - os: ubuntu\n      needs: [sudo]\n      apt: {packages: [gh]}", 1)
	if err := decoded(t, release); err != nil {
		t.Fatalf("a release refused: %v", err)
	}
	if err := decoded(t, apt); err != nil {
		t.Fatalf("an apt method refused: %v", err)
	}
	for body, want := range map[string]string{
		strings.Replace(release, "provides: [gh]", "provides: ['../../.profile']", 1):                                                        `"../../.profile" is no command name`,
		strings.Replace(release, "github-release:", "provides: ['-gh']\n      github-release:", 1):                                           `"-gh" is no command name`,
		strings.Replace(release, "acme/tool", "'../../../x'", 1):                                                                             `"../../../x" is no repo`,
		strings.Replace(release, "acme/tool", "'a/../../../../../tmp/x'", 1):                                                                 `"a/../../../../../tmp/x" is no repo`,
		strings.Replace(release, "provides: [gh]\n", "", 1):                                                                                  "a release lands the commands its app names in provides",
		strings.Replace(apt, "packages: [gh]", "packages: ['-oAPT::Update::Pre-Invoke::=id>/tmp/pwn', gh]", 1):                               `"-oAPT::Update::Pre-Invoke::=id>/tmp/pwn" is no package name`,
		strings.Replace(apt, "packages: [gh]", "packages: ['jq curl']", 1):                                                                   `"jq curl" is no package name`,
		strings.Replace(release, "github-release: {repo: acme/tool, asset: 'tool-{os}-{arch}'}", "skill: {repo: '--help', name: gh}", 1):     `"--help" is no repo`,
		strings.Replace(release, "github-release: {repo: acme/tool, asset: 'tool-{os}-{arch}'}", "skill: {repo: acme/tool, name: '-gh'}", 1): `"-gh" is no skill name`,
	} {
		refusedFor(t, body, want)
	}
}
