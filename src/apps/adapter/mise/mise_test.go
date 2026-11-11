package mise_test

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/apps/adapter/mise"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

// value: mise's global runtimes as these tests play them.
type runtimes struct {
	pinned map[string]string
	fails  bool
}

func (r *runtimes) play(c local.Cmd) (local.Output, error) {
	args := strings.Join(c.Args, " ")
	if r.fails && args != "ls --global --json" {
		return local.Output{ExitCode: 1}, nil
	}
	switch {
	case args == "ls --global --json":
		doc := map[string][]map[string]any{}
		for tool, version := range r.pinned {
			doc[tool] = []map[string]any{{"version": version, "installed": true}}
		}
		body, err := json.Marshal(doc)
		return local.Output{Stdout: string(body)}, err
	case strings.HasPrefix(args, "use --global "):
		tool, version, _ := strings.Cut(strings.TrimPrefix(args, "use --global "), "@")
		r.pinned[tool] = version
	case strings.HasPrefix(args, "upgrade "):
	case strings.HasPrefix(args, "unuse --global --no-prune "):
		delete(r.pinned, strings.TrimPrefix(args, "unuse --global --no-prune "))
	default:
		return local.Output{ExitCode: 97}, nil
	}
	return local.Output{}, nil
}

func TestInstaller(t *testing.T) {
	contracttest.Installer(t, catalog.MiseInstall{App: "node", Version: "lts"}, func(_ *testing.T, seed contracttest.Seed) contract.Installer {
		r := &runtimes{pinned: map[string]string{}, fails: seed.Fails}
		for k, v := range seed.Installed {
			r.pinned[k] = v
		}
		names := []string{"mise"}
		if seed.Absent {
			names = nil
		}
		return mise.New(local.Played(r.play, names...))
	})
}

func TestARuntimeDeclaredButNotInstalledIsNotListed(t *testing.T) {
	run := local.Played(func(local.Cmd) (local.Output, error) {
		return local.Output{Stdout: `{"node":[{"version":"24.11.0","installed":true}],"go":[{"version":"1.25","installed":false}]}`}, nil
	}, "mise")

	got, err := mise.New(run).Inventory(context.Background())

	if err != nil || got["node"] != "24.11.0" || len(got) != 1 {
		t.Fatalf("inventory %v %v", got, err)
	}
}

func TestAManagerCommandInstallsNoRuntimeByItself(t *testing.T) {
	// invariant: a shim run from a folder that pins a runtime mise lacks
	// installs it unless told not to, so a read would install one.
	pinned := mise.Pinned("/home/u")
	for _, kv := range []string{"MISE_CEILING_PATHS=/home/u", "MISE_AUTO_INSTALL=false", "MISE_EXEC_AUTO_INSTALL=false", "MISE_NOT_FOUND_AUTO_INSTALL=false"} {
		if !slices.Contains(pinned, kv) {
			t.Errorf("a manager command runs without %s", kv)
		}
	}
}

func TestAManagerCommandGetsTheGlobalVersionsWhateverItsShellPutFirst(t *testing.T) {
	// invariant: `mise activate` puts a project's install first on PATH, nvm
	// its own versions, and mise's profile or a version named in the
	// environment moves what mise reads as global, so each would answer for
	// another runtime than the one `unuse --global` takes back.
	home := "/home/u"
	vars := map[string]string{
		"HOME": home, "TERM": "dumb",
		"PATH":     home + "/.local/share/mise/installs/node/22.1.0/bin:" + home + "/.nvm/versions/node/v20.0.0/bin:" + home + "/.asdf/installs/nodejs/18.0.0/bin:/usr/bin:" + home + "/.local/share/mise/shims",
		"MISE_ENV": "work", "MISE_PROFILE": "work", "MISE_NODE_VERSION": "22.1.0", "MISE_SHELL": "bash", "__MISE_DIFF": "x", "__MISE_SESSION": "x",
		"MISE_AUTO_INSTALL": "true", "MOVED_PREFIX": "/elsewhere",
	}

	env := mise.Environment(home, vars, "moved_prefix")

	want := []string{"HOME=/home/u", "PATH=/home/u/.local/share/mise/shims:/usr/bin", "TERM=dumb"}
	if got := env[:len(want)]; !slices.Equal(got, want) {
		t.Errorf("the environment opens with %v; want %v", got, want)
	}
	if got := env[len(want):]; !slices.Equal(got, mise.Pinned(home)) {
		t.Errorf("the environment then carries %v; want the pins alone", got)
	}
}

func TestTheShimsMoveWithMisesDataFolder(t *testing.T) {
	env := mise.Environment("/home/u", map[string]string{"MISE_DATA_DIR": "/data/mise", "PATH": "/data/mise/installs/node/24/bin:/usr/bin"})

	if !slices.Contains(env, "PATH=/data/mise/shims:/usr/bin") {
		t.Errorf("the environment is %v; want the moved shims first and the moved installs gone", env)
	}
}

func TestWhereIsTheFolderOfEveryVersionHeldGlobally(t *testing.T) {
	// invariant: `unuse --global` takes back every version the global file
	// names, so each installed one is named, in mise's order.
	run := local.Played(func(local.Cmd) (local.Output, error) {
		return local.Output{Stdout: `{"node":[{"version":"22.1.0","install_path":"/n/22.1.0","installed":true},{"version":"23.0.0","install_path":"/n/23.0.0","installed":false},{"version":"24.22.2","install_path":"/n/24.22.2","installed":true}]}`}, nil
	}, "mise")
	m := mise.New(run)

	if got, err := m.Where(context.Background(), "node"); err != nil || !slices.Equal(got, []string{"/n/22.1.0", "/n/24.22.2"}) {
		t.Errorf("node is at %q (%v); want both installed global versions' folders", got, err)
	}
	if got, err := m.Inventory(context.Background()); err != nil || got["node"] != "22.1.0" {
		t.Errorf("inventory %v (%v); want node at its first installed version", got, err)
	}
	if got, err := m.Where(context.Background(), "python"); err != nil || len(got) != 0 {
		t.Errorf("python is at %q (%v); want none, mise holds no python", got, err)
	}
	if got, err := mise.New(local.Played(nil)).Where(context.Background(), "node"); err != nil || len(got) != 0 {
		t.Errorf("node is at %q (%v) with no mise; want none", got, err)
	}
}
