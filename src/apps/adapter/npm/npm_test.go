package npm_test

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/apps/adapter/npm"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

var line = regexp.MustCompile(`^npm "([^"]+)"$`)

// value: npm's global prefix as these tests play it, and the bundle that
// installs its lines.
type prefix struct {
	packages map[string]string
	fails    bool
}

func (p *prefix) play(c local.Cmd) (local.Output, error) {
	args := strings.Join(c.Args, " ")
	switch {
	case args == "ls -g --depth=0 --json":
		deps := map[string]map[string]string{}
		for name, version := range p.packages {
			deps[name] = map[string]string{"version": version}
		}
		body, err := json.Marshal(map[string]any{"dependencies": deps})
		return local.Output{Stdout: string(body)}, err
	case strings.HasPrefix(args, "uninstall -g "):
		delete(p.packages, strings.TrimPrefix(args, "uninstall -g "))
		return local.Output{}, nil
	}
	return local.Output{ExitCode: 97}, nil
}

func (p *prefix) Name() string { return "the bundle" }

func (p *prefix) Bundle(_ context.Context, lines []string, _ bool, _ contract.Logs) error {
	if p.fails {
		return errors.New("the bundle exited 1")
	}
	for _, l := range lines {
		if m := line.FindStringSubmatch(l); m != nil {
			p.packages[m[1]] = "1.0.0"
		}
	}
	return nil
}

func TestInstaller(t *testing.T) {
	contracttest.Installer(t, catalog.NpmInstall{Package: "@playwright/mcp"}, func(_ *testing.T, seed contracttest.Seed) contract.Installer {
		p := &prefix{packages: map[string]string{}, fails: seed.Fails}
		for k, v := range seed.Installed {
			p.packages[k] = v
		}
		names := []string{"npm"}
		if seed.Absent {
			names = nil
		}
		return npm.New(local.Played(p.play, names...), p)
	})
}

func TestAListingThatExitsOneStillListsEveryPackage(t *testing.T) {
	run := local.Played(func(local.Cmd) (local.Output, error) {
		return local.Output{ExitCode: 1, Stdout: `{"dependencies":{"@playwright/mcp":{"version":"0.0.41"}}}`, Stderr: "npm error missing: peer"}, nil
	}, "npm")

	got, err := npm.New(run, nil).Inventory(context.Background())

	if err != nil || got["@playwright/mcp"] != "0.0.41" {
		t.Fatalf("inventory %v %v", got, err)
	}
}
