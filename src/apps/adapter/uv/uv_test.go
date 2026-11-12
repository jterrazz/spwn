package uv_test

import (
	"context"
	"errors"
	"github.com/jterrazz/studio-station/src/apps"
	"regexp"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/apps/adapter/uv"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
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

func (u *tools) Bundle(_ context.Context, lines []string, _ bool, _ contract.Logs) error {
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

func TestOutdatedReadsTheLatestUvMarksOnEachTool(t *testing.T) {
	// outside: a recorded `uv tool list --outdated` (uv 0.13.7): every tool is
	// listed, the behind ones with `[latest: <version>]`.
	listing := "cowsay v6.0 [latest: 6.1]\n- cowsay\nhttpie v3.2.4\n- http\n- httpie\n- https\n"
	var ran local.Cmd
	run := local.Played(func(c local.Cmd) (local.Output, error) {
		ran = c
		return local.Output{Stdout: listing}, nil
	}, "uv")

	got, err := uv.New(run, nil).Outdated(context.Background(), []string{"cowsay", "httpie", "ruff"})

	if err != nil || got["cowsay"] != (apps.Currency{Verdict: apps.Behind, Newest: "6.1"}) || got["httpie"].Verdict != apps.Current || got["ruff"].Verdict != apps.Unknown {
		t.Fatalf("currency %v %v", got, err)
	}
	if want := "tool list --outdated"; strings.Join(ran.Args, " ") != want || ran.Timeout != contract.CurrencyBudget {
		t.Errorf("uv ran %q within %v; want %q within %v", ran.Args, ran.Timeout, want, contract.CurrencyBudget)
	}
}

func TestAUvThatDoesNotAnswerLeavesTheCurrencyUnknown(t *testing.T) {
	run := local.Played(func(local.Cmd) (local.Output, error) { return local.Output{ExitCode: 2}, nil }, "uv")

	got, err := uv.New(run, nil).Outdated(context.Background(), []string{"cowsay"})

	if err != nil || got["cowsay"].Verdict != apps.Unknown {
		t.Errorf("currency %v %v; want unknown", got, err)
	}
}
