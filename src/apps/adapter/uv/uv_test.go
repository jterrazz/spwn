package uv_test

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/apps/adapter/uv"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/platform/local"
)

var line = regexp.MustCompile(`^uv "([^"\[]+)`)

// value: uv's tools as these tests play them, and the bundle that installs
// their lines.
type tools struct {
	installed map[string]string
	fails     bool
}

func (u *tools) play(c local.Cmd) (local.Output, error) {
	args := strings.Join(c.Args, " ")
	switch {
	case args == "tool list":
		out := make([]string, 0, 2*len(u.installed))
		for name, version := range u.installed {
			out = append(out, name+" v"+version, "- "+name)
		}
		return local.Output{Stdout: strings.Join(out, "\n")}, nil
	case strings.HasPrefix(args, "tool uninstall "):
		delete(u.installed, strings.TrimPrefix(args, "tool uninstall "))
		return local.Output{}, nil
	}
	return local.Output{ExitCode: 97}, nil
}

func (u *tools) Name() string { return "the bundle" }

func (u *tools) Bundle(_ context.Context, lines []string, _ bool, _ call.Progress) error {
	if u.fails {
		return errors.New("the bundle exited 1")
	}
	for _, l := range lines {
		if m := line.FindStringSubmatch(l); m != nil {
			u.installed[m[1]] = "0.1.0"
		}
	}
	return nil
}

func TestInstaller(t *testing.T) {
	contracttest.Installer(t, catalog.UvInstall{Package: "markitdown[all]"}, func(_ *testing.T, seed contracttest.Seed) contract.Installer {
		u := &tools{installed: map[string]string{}, fails: seed.Fails}
		for k, v := range seed.Installed {
			u.installed[k] = v
		}
		names := []string{"uv"}
		if seed.Absent {
			names = nil
		}
		return uv.New(local.Played(u.play, names...), u)
	})
}

func TestAToolIsListedWithoutItsExecutablesAndTheV(t *testing.T) {
	run := local.Played(func(local.Cmd) (local.Output, error) {
		return local.Output{Stdout: "markitdown v0.1.2\n- markitdown\n- markitdown-mcp\n"}, nil
	}, "uv")

	got, err := uv.New(run, nil).Inventory(context.Background())

	if err != nil || len(got) != 1 || got["markitdown"] != "0.1.2" {
		t.Fatalf("inventory %v %v", got, err)
	}
}
