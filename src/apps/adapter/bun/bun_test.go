package bun_test

import (
	"context"
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
