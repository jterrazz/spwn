package npm_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jterrazz/studio-station/apps/engine/apps"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/npm"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
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
		return npm.New(local.Played(p.play, names...), p, nil)
	})
}

func TestAListingThatExitsOneStillListsEveryPackage(t *testing.T) {
	run := local.Played(func(local.Cmd) (local.Output, error) {
		return local.Output{ExitCode: 1, Stdout: `{"dependencies":{"@playwright/mcp":{"version":"0.0.41"}}}`, Stderr: "npm error missing: peer"}, nil
	}, "npm")

	got, err := npm.New(run, nil, nil).Inventory(context.Background())

	if err != nil || got["@playwright/mcp"] != "0.0.41" {
		t.Fatalf("inventory %v %v", got, err)
	}
}

func TestTheListingLeavesOutWhatNodeItselfLays(t *testing.T) {
	run := local.Played(func(local.Cmd) (local.Output, error) {
		return local.Output{Stdout: `{"dependencies":{"corepack":{"version":"0.34.0"},"npm":{"version":"11.6.2"},"typescript":{"version":"5.9.3"}}}`}, nil
	}, "npm")

	got, err := npm.New(run, nil, nil).Inventory(context.Background())

	if err != nil || len(got) != 1 || got["typescript"] != "5.9.3" {
		t.Fatalf("inventory %v %v; want typescript alone", got, err)
	}
}

func TestTheReadAndTheRemovalAddressTheFolderOfTheGlobalNode(t *testing.T) {
	// invariant: a node first on PATH, or a prefix the environment moves,
	// would make npm list and remove another node's packages than those of
	// the node its manager takes back.
	var ran []string
	run := local.Played(func(c local.Cmd) (local.Output, error) {
		ran = append(ran, strings.Join(c.Args, " "))
		return local.Output{Stdout: `{"dependencies":{"typescript":{"version":"5.9.3"}}}`}, nil
	}, "npm")
	where := func(_ context.Context, runtime string) ([]string, error) {
		if runtime != "node" {
			return nil, nil
		}
		return []string{"/home/u/node/24.22.2"}, nil
	}
	n := npm.New(run, nil, where)

	_, _ = n.Inventory(context.Background())
	_ = n.Remove(context.Background(), []catalog.Install{catalog.NpmInstall{Package: "typescript"}}, nil)

	want := []string{"ls -g --depth=0 --json --prefix /home/u/node/24.22.2", "uninstall -g --prefix /home/u/node/24.22.2 typescript"}
	if !slices.Equal(ran, want) {
		t.Errorf("npm ran %q; want %q", ran, want)
	}
}

func TestAGlobalNodeThatDoesNotReadLeavesThePackagesUnread(t *testing.T) {
	run := local.Played(func(local.Cmd) (local.Output, error) {
		return local.Output{Stdout: `{"dependencies":{}}`}, nil
	}, "npm")
	unread := func(context.Context, string) ([]string, error) { return nil, errors.New("the manager did not answer") }

	if got, err := npm.New(run, nil, unread).Inventory(context.Background()); err == nil {
		t.Fatalf("inventory %v; want the global node's read to fail it", got)
	}
}

func TestEveryGlobalNodeIsReadAndRemovedFrom(t *testing.T) {
	// invariant: the manager's removal takes back every version it holds
	// globally, so a package of the owner's on any of them is held, and
	// station's own package is gone from each.
	held := map[string]string{
		"/home/u/node/22.1.0":  `{"dependencies":{"typescript":{"version":"5.9.3"}}}`,
		"/home/u/node/24.22.2": `{"dependencies":{"typescript":{"version":"5.9.3"},"owner-pkg":{"version":"1.0.0"}}}`,
	}
	var removed []string
	run := local.Played(func(c local.Cmd) (local.Output, error) {
		args := strings.Join(c.Args, " ")
		for folder, doc := range held {
			switch args {
			case "ls -g --depth=0 --json --prefix " + folder:
				return local.Output{Stdout: doc}, nil
			case "uninstall -g --prefix " + folder + " typescript":
				removed = append(removed, folder)
				return local.Output{}, nil
			}
		}
		return local.Output{ExitCode: 97}, nil
	}, "npm")
	where := func(context.Context, string) ([]string, error) {
		return []string{"/home/u/node/22.1.0", "/home/u/node/24.22.2"}, nil
	}
	n := npm.New(run, nil, where)

	got, err := n.Inventory(context.Background())
	if err != nil || got["owner-pkg"] != "1.0.0" || got["typescript"] != "5.9.3" || len(got) != 2 {
		t.Errorf("inventory %v %v; want the packages of both nodes", got, err)
	}
	if err := n.Remove(context.Background(), []catalog.Install{catalog.NpmInstall{Package: "typescript"}}, nil); err != nil {
		t.Fatal(err)
	}
	if want := []string{"/home/u/node/22.1.0", "/home/u/node/24.22.2"}; !slices.Equal(removed, want) {
		t.Errorf("typescript removed from %q; want %q", removed, want)
	}
}

func TestAGlobalNodeWhoseListDoesNotReadLeavesThePackagesUnread(t *testing.T) {
	run := local.Played(func(c local.Cmd) (local.Output, error) {
		if strings.HasSuffix(strings.Join(c.Args, " "), "/home/u/node/24.22.2") {
			return local.Output{ExitCode: 1, Stdout: "npm error"}, nil
		}
		return local.Output{Stdout: `{"dependencies":{}}`}, nil
	}, "npm")
	where := func(context.Context, string) ([]string, error) {
		return []string{"/home/u/node/22.1.0", "/home/u/node/24.22.2"}, nil
	}

	if got, err := npm.New(run, nil, where).Inventory(context.Background()); err == nil {
		t.Fatalf("inventory %v; want the second node's list to fail it", got)
	}
}

func TestOutdatedReadsTheRegistrysLatestOfEachGlobalNode(t *testing.T) {
	// outside: a recorded `npm outdated -g --json`: it exits 1 when it lists a
	// package, and lists the npm and corepack node itself lays too.
	listing := `{
  "cowsay": {"current": "1.4.0", "wanted": "1.6.0", "latest": "1.6.0", "dependent": "global", "location": "/n/lib/node_modules/cowsay"},
  "npm": {"current": "10.7.0", "wanted": "12.2.2", "latest": "12.2.2", "dependent": "global", "location": "/n/lib/node_modules/npm"}
}`
	var ran []local.Cmd
	run := local.Played(func(c local.Cmd) (local.Output, error) {
		ran = append(ran, c)
		return local.Output{ExitCode: 1, Stdout: listing}, nil
	}, "npm")
	where := func(context.Context, string) ([]string, error) { return []string{"/home/u/node/22.1.0"}, nil }

	got, err := npm.New(run, nil, where).Outdated(context.Background(), []string{"cowsay", "left-pad"})

	if err != nil || got["cowsay"] != (apps.Currency{Verdict: apps.Behind, Newest: "1.6.0"}) || got["left-pad"].Verdict != apps.Current {
		t.Fatalf("currency %v %v", got, err)
	}
	if want := "outdated -g --json --prefix /home/u/node/22.1.0"; len(ran) != 1 || strings.Join(ran[0].Args, " ") != want || ran[0].Timeout != contract.CurrencyBudget {
		t.Errorf("npm ran %v; want %q within %v", ran, want, contract.CurrencyBudget)
	}
}

func TestOutdatedIsCurrentWhenNpmListsNothingAndUnknownWhenItDidNotAnswer(t *testing.T) {
	for name, tc := range map[string]struct {
		out  local.Output
		want apps.Verdict
	}{
		"nothing behind":       {local.Output{}, apps.Current},
		"the registry is down": {local.Output{ExitCode: 1}, apps.Unknown},
		"npm's own error":      {local.Output{ExitCode: 254, Stdout: `{"error":{"code":"ENOENT","summary":"no such file"}}`}, apps.Unknown},
		"an error document":    {local.Output{ExitCode: 1, Stdout: `{"error":{"code":"E","summary":"s"}}`}, apps.Unknown},
	} {
		run := local.Played(func(local.Cmd) (local.Output, error) { return tc.out, nil }, "npm")
		got, err := npm.New(run, nil, nil).Outdated(context.Background(), []string{"cowsay"})
		if err != nil || got["cowsay"].Verdict != tc.want {
			t.Errorf("%s: currency %v %v; want %s", name, got, err, tc.want)
		}
	}
}

func TestOutdatedIsBehindWhereAnyGlobalNodeHoldsAnOldCopy(t *testing.T) {
	run := local.Played(func(c local.Cmd) (local.Output, error) {
		if strings.HasSuffix(strings.Join(c.Args, " "), "/22.1.0") {
			return local.Output{ExitCode: 1, Stdout: `{"cowsay":{"current":"1.4.0","latest":"1.6.0"}}`}, nil
		}
		return local.Output{}, nil
	}, "npm")
	where := func(context.Context, string) ([]string, error) {
		return []string{"/home/u/node/22.1.0", "/home/u/node/24.22.2"}, nil
	}

	got, _ := npm.New(run, nil, where).Outdated(context.Background(), []string{"cowsay"})

	if got["cowsay"].Verdict != apps.Behind {
		t.Errorf("currency %v; want behind", got)
	}
}
