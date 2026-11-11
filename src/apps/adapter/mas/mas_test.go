package mas_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/adapter/mas"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

// value: the App Store as these tests play it: mas installs, updates and
// removes by id, and a failing store exits 1.
type store struct {
	apps  map[string]string
	fails bool
}

func (s *store) play(c local.Cmd) (local.Output, error) {
	verb, id, _ := strings.Cut(strings.Join(c.Args, " "), " ")
	switch {
	case verb == "list":
		out := make([]string, 0, len(s.apps))
		for id, version := range s.apps {
			out = append(out, id+"  App  ("+version+")")
		}
		return local.Output{Stdout: strings.Join(out, "\n")}, nil
	case s.fails:
		return local.Output{ExitCode: 1, Stderr: "Failed to download"}, nil
	case verb == "install" || verb == "update":
		s.apps[id] = "1.0"
		return local.Output{Stderr: "==> Downloading App"}, nil
	case verb == "uninstall":
		delete(s.apps, id)
		return local.Output{}, nil
	}
	return local.Output{ExitCode: 97}, nil
}

func TestInstaller(t *testing.T) {
	contracttest.Installer(t, catalog.MasInstall{Name: "Xcode", ID: 497799835}, func(_ *testing.T, seed contracttest.Seed) contract.Installer {
		s := &store{apps: map[string]string{}, fails: seed.Fails}
		for k, v := range seed.Installed {
			s.apps[k] = v
		}
		names := []string{"mas"}
		if seed.Absent {
			names = nil
		}
		return mas.New(local.Played(s.play, names...))
	})
}

func TestASignedOutAppStoreRefusesWithTheSignInHint(t *testing.T) {
	signedOut := local.Played(func(c local.Cmd) (local.Output, error) {
		if c.Answer <= 0 {
			t.Fatalf("mas %v ran with no bound on its first word", c.Args)
		}
		return local.Output{}, local.ErrSilent
	}, "mas")

	err := mas.New(signedOut).Install(context.Background(), []catalog.Install{catalog.MasInstall{Name: "Speedtest", ID: 1153157709}}, nil)

	var out *apps.SignedOutError
	if !errors.As(err, &out) || !strings.Contains(err.Error(), "open the App Store and sign in") {
		t.Fatalf("a silent App Store ended as %v", err)
	}
}

func TestAnAppIsListedByItsIDWithTheVersionInParentheses(t *testing.T) {
	run := local.Played(func(local.Cmd) (local.Output, error) {
		return local.Output{Stdout: "497799835  Xcode  (26.1)\n409183694  Keynote 14  (14.4)\n"}, nil
	}, "mas")

	got, err := mas.New(run).Inventory(context.Background())

	if err != nil || got["497799835"] != "26.1" || got["409183694"] != "14.4" {
		t.Fatalf("inventory %v %v", got, err)
	}
}

func TestABareUpdateLeavesTheAppStoreItsOwnApps(t *testing.T) {
	if mas.New(nil).Can().UpdateAll {
		t.Error("a bare update reaches the App Store's apps")
	}
}
