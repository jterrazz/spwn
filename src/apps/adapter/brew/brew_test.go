package brew_test

import (
	"context"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/apps/adapter/brew"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

var brewfileLine = regexp.MustCompile(`^(brew|cask) "(?:[^"]*/)?([^"/]+)"`)

// value: Homebrew as these tests play it: what it lists, what a bundle and
// an uninstall change, and every command it was given.
type cellar struct {
	formulae, casks map[string]string
	fails           bool
	calls           []string
}

func newCellar(s contracttest.Seed, cask bool) *cellar {
	c := &cellar{formulae: map[string]string{}, casks: map[string]string{}, fails: s.Fails}
	target := c.formulae
	if cask {
		target = c.casks
	}
	for k, v := range s.Installed {
		target[k] = v
	}
	return c
}

func (c *cellar) play(cmd local.Cmd) (local.Output, error) {
	args := strings.Join(cmd.Args, " ")
	c.calls = append(c.calls, args+"\n"+cmd.Stdin)
	listing := func(m map[string]string) string {
		lines := make([]string, 0, len(m))
		for _, k := range slices.Sorted(maps.Keys(m)) {
			lines = append(lines, k+" "+m[k])
		}
		return strings.Join(lines, "\n") + "\n"
	}
	switch {
	case args == "list --formula --versions":
		return local.Output{Stdout: listing(c.formulae)}, nil
	case args == "list --cask --versions":
		return local.Output{Stdout: listing(c.casks)}, nil
	case c.fails:
		return local.Output{ExitCode: 1, Stderr: "==> Downloading\nError: 404\n"}, nil
	case strings.HasPrefix(args, "bundle install "):
		for _, line := range strings.Split(cmd.Stdin, "\n") {
			if m := brewfileLine.FindStringSubmatch(line); m != nil {
				map[string]map[string]string{"brew": c.formulae, "cask": c.casks}[m[1]][m[2]] = "2.0.0"
			}
		}
	case strings.HasPrefix(args, "uninstall --cask "):
		delete(c.casks, strings.TrimPrefix(args, "uninstall --cask "))
	case strings.HasPrefix(args, "uninstall "):
		delete(c.formulae, path.Base(strings.TrimPrefix(args, "uninstall ")))
	default:
		return local.Output{ExitCode: 97, Stderr: "unplayed: " + args}, nil
	}
	return local.Output{}, nil
}

func build(c *cellar, absent bool) *brew.Brew {
	var names []string
	if !absent {
		names = []string{"brew"}
	}
	return brew.New(local.Played(c.play, names...), nil)
}

func TestFormulae(t *testing.T) {
	contracttest.Installer(t, catalog.BrewInstall{Formula: "hashicorp/tap/terraform", Trusted: true}, func(_ *testing.T, s contracttest.Seed) contract.Installer {
		return build(newCellar(s, false), s.Absent).Formulae()
	})
}

func TestCasks(t *testing.T) {
	contracttest.Installer(t, catalog.CaskInstall{Token: "ghostty"}, func(_ *testing.T, s contracttest.Seed) contract.Installer {
		return build(newCellar(s, true), s.Absent).Casks()
	})
}

func TestBundle(t *testing.T) {
	contracttest.Bundler(t, catalog.BrewInstall{Formula: "hashicorp/tap/terraform", Trusted: true}, func(_ *testing.T, s contracttest.Seed) contracttest.Bundled {
		b := build(newCellar(s, false), s.Absent)
		return contracttest.Bundled{Bundler: b, Installer: b.Formulae()}
	})
}

func TestAFormulaListedWithSeveralVersionsReadsItsLatest(t *testing.T) {
	run := local.Played(func(local.Cmd) (local.Output, error) {
		return local.Output{Stdout: "bat 0.24.0 0.25.0\njq\n"}, nil
	}, "brew")

	got, err := brew.New(run, nil).Formulae().Inventory(context.Background())

	if err != nil || got["bat"] != "0.25.0" || got["jq"] != "" || len(got) != 2 {
		t.Fatalf("inventory %v %v", got, err)
	}
}

func TestATappedFormulaIsWrittenWholeAndTrusted(t *testing.T) {
	lined, ok := build(newCellar(contracttest.Seed{}, false), false).Formulae().(contract.Lined)
	if !ok {
		t.Fatal("formulae ride no bundle")
	}
	if got := lined.Line(catalog.BrewInstall{Formula: "hashicorp/tap/terraform", Trusted: true}); got != `brew "hashicorp/tap/terraform", trusted: true` {
		t.Errorf("line %q", got)
	}
}

func TestOneBundleCarriesEveryLineAndNamesItsFailure(t *testing.T) {
	c := newCellar(contracttest.Seed{Fails: true}, false)

	err := build(c, false).Bundle(context.Background(), []string{`brew "bat"`, `cask "ghostty"`}, false, nil)

	if err == nil || err.Error() != "brew bundle exited 1: Error: 404" {
		t.Errorf("error %v", err)
	}
	if len(c.calls) != 1 || c.calls[0] != "bundle install --no-upgrade --file=-\nbrew \"bat\"\ncask \"ghostty\"\n" {
		t.Errorf("calls %q", c.calls)
	}
}
