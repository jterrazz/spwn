package mise_test

import (
	"context"
	"encoding/json"
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
	case strings.HasPrefix(args, "unuse --global "):
		delete(r.pinned, strings.TrimPrefix(args, "unuse --global "))
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
