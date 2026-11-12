package bun_test

import (
	"context"
	"github.com/jterrazz/studio-station/src/apps"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/apps/adapter/bun"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

// value: bun's global packages as these tests play them.
type globals struct {
	packages map[string]string
	fails    bool
}

func (g *globals) play(c local.Cmd) (local.Output, error) {
	args := strings.Join(c.Args, " ")
	if g.fails && args != "pm ls -g" {
		return local.Output{ExitCode: 1, Stderr: "error: GET failed"}, nil
	}
	switch {
	case args == "pm ls -g":
		var out strings.Builder
		out.WriteString("/home/u/.bun/install/global node_modules\n")
		for name, version := range g.packages {
			out.WriteString("+-- " + name + "@" + version + "\n")
		}
		return local.Output{Stdout: out.String()}, nil
	case strings.HasPrefix(args, "install -g "):
		g.packages[strings.TrimPrefix(args, "install -g ")] = "1.0.0"
	case strings.HasPrefix(args, "update -g "):
	case strings.HasPrefix(args, "remove -g "):
		delete(g.packages, strings.TrimPrefix(args, "remove -g "))
	default:
		return local.Output{ExitCode: 97}, nil
	}
	return local.Output{}, nil
}

func TestInstaller(t *testing.T) {
	contracttest.Installer(t, catalog.BunInstall{Package: "skills"}, func(_ *testing.T, seed contracttest.Seed) contract.Installer {
		g := &globals{packages: map[string]string{}, fails: seed.Fails}
		for k, v := range seed.Installed {
			g.packages[k] = v
		}
		names := []string{"bun"}
		if seed.Absent {
			names = nil
		}
		return bun.New(local.Played(g.play, names...))
	})
}

func TestAScopedPackageIsListedWithItsScope(t *testing.T) {
	run := local.Played(func(local.Cmd) (local.Output, error) {
		return local.Output{Stdout: "/u/.bun/install/global node_modules (3)\n+-- @anthropic-ai/claude-code@2.0.1\n`-- qmd@0.4.0\n"}, nil
	}, "bun")

	got, err := bun.New(run).Inventory(context.Background())

	if err != nil || got["@anthropic-ai/claude-code"] != "2.0.1" || got["qmd"] != "0.4.0" {
		t.Fatalf("inventory %v %v", got, err)
	}
}

func TestAFailedInstallNamesTheCommandAndItsExit(t *testing.T) {
	run := local.Played(func(local.Cmd) (local.Output, error) { return local.Output{ExitCode: 1}, nil }, "bun")

	err := bun.New(run).Install(context.Background(), []catalog.Install{catalog.BunInstall{Package: "github:tobi/qmd", Name: "qmd"}}, nil)

	if err == nil || err.Error() != "bun install exited 1" {
		t.Errorf("error %v", err)
	}
}

func TestOutdatedIsWhatAGlobalUpdateWouldMove(t *testing.T) {
	// outside: a recorded `bun outdated -g` (bun 1.4.4): cowsay is installed
	// at a caret range, so its Update column moves; left-pad is installed at an
	// exact version, so only its Latest column differs and an update leaves it.
	table := `bun outdated v1.4.4 (38afee6c8)
|--------------------------------------|
| Package  | Current | Update | Latest |
|----------|---------|--------|--------|
| cowsay   | 1.4.0   | 1.6.0  | 1.6.0  |
|----------|---------|--------|--------|
| left-pad | 1.0.0   | 1.0.0  | 1.3.0  |
|--------------------------------------|
`
	var ran local.Cmd
	run := local.Played(func(c local.Cmd) (local.Output, error) {
		ran = c
		return local.Output{Stdout: table}, nil
	}, "bun")

	got, err := bun.New(run).Outdated(context.Background(), []string{"cowsay", "left-pad", "is-odd"})

	if err != nil || got["cowsay"] != (apps.Currency{Verdict: apps.Behind, Newest: "1.6.0"}) || got["left-pad"].Verdict != apps.Current || got["is-odd"].Verdict != apps.Current {
		t.Fatalf("currency %v %v", got, err)
	}
	if want := "outdated -g"; strings.Join(ran.Args, " ") != want || ran.Timeout != contract.CurrencyBudget {
		t.Errorf("bun ran %q within %v; want %q within %v", ran.Args, ran.Timeout, want, contract.CurrencyBudget)
	}
}

func TestABunWithNoOutdatedOrNoRegistryLeavesTheCurrencyUnknown(t *testing.T) {
	// outside: a bun before 1.2 answers `outdated` as a subcommand reserved
	// for future use, exit 1; one that cannot reach the registry prints only its
	// banner on stdout, its errors on stderr, exit 1.
	for name, out := range map[string]local.Output{
		"reserved":   {ExitCode: 1, Stdout: "Uh-oh. bun outdated is a subcommand reserved for future use by Bun."},
		"no network": {ExitCode: 1, Stdout: "bun outdated v1.4.4 (38afee6c8)\n", Stderr: "error: ConnectionRefused downloading package manifest cowsay"},
	} {
		run := local.Played(func(local.Cmd) (local.Output, error) { return out, nil }, "bun")
		got, err := bun.New(run).Outdated(context.Background(), []string{"cowsay"})
		if err != nil || got["cowsay"].Verdict != apps.Unknown {
			t.Errorf("%s: currency %v %v; want unknown", name, got, err)
		}
	}
}

func TestARangeIsInstalledAsTheSpecifierBunTakes(t *testing.T) {
	var ran []string
	run := local.Played(func(c local.Cmd) (local.Output, error) {
		ran = append(ran, strings.Join(c.Args, " "))
		return local.Output{}, nil
	}, "bun")
	for _, text := range []string{"", "1", "1.2", "^1.2.0", "1.2.3"} {
		r, _ := catalog.ParseRange(text)
		in, _ := catalog.Pinned(catalog.BunInstall{Package: "github:tobi/qmd", Name: "qmd"}, r)
		if err := bun.New(run).Install(context.Background(), []catalog.Install{in}, nil); err != nil {
			t.Fatal(err)
		}
	}
	want := "install -g github:tobi/qmd|install -g github:tobi/qmd@^1.0.0|install -g github:tobi/qmd@~1.2.0|install -g github:tobi/qmd@^1.2.0|install -g github:tobi/qmd@1.2.3"
	if got := strings.Join(ran, "|"); got != want {
		t.Errorf("ran %q; want %q", got, want)
	}
}
