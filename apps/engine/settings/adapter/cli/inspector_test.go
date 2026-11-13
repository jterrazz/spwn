package cli_test

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/cli"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
)

type files struct{ m fstest.MapFS }

func (f files) Stat(p string) (fs.FileInfo, error) { return fs.Stat(f.m, p[1:]) }

// value: an owner who set their email, signs, and is signed in to GitHub on
// stderr, as gh 2.x prints it; no gpg on the PATH.
func owner(c local.Cmd) (local.Output, error) {
	switch c.Name {
	case "git":
		if c.Args[2] == "commit.gpgsign" {
			return local.Output{Stdout: "true\n"}, nil
		}
		if c.Args[2] == "user.email" {
			return local.Output{Stdout: "admin@jterrazz.com\n"}, nil
		}
		return local.Output{ExitCode: 1}, nil
	case "gh":
		return local.Output{Stderr: "github.com\n  - Logged in to github.com account jterrazz (keyring)\n"}, nil
	}
	return local.Output{ExitCode: 97}, nil
}

func TestInspector(t *testing.T) {
	home := files{fstest.MapFS{"home/.ssh/id_ed25519": {}}}
	run := local.Played(owner, "git", "gh")
	contracttest.Inspector(t, func(*testing.T) contracttest.InspectorSubject {
		return contracttest.InspectorSubject{Inspector: cli.New(run, home, cli.Options{Home: "/home"}), Unreadable: catalog.FactFileVault, Present: catalog.FactGitEmail}
	})
	i := cli.New(run, home, cli.Options{Home: "/home"})
	for f, want := range map[catalog.Fact]string{
		catalog.FactGitEmail: "admin@jterrazz.com", catalog.FactGitName: "", catalog.FactGitSigning: "on",
		catalog.FactGPGKey: "", catalog.FactGitHub: "jterrazz", catalog.FactSSHKey: "present",
	} {
		if got, err := i.Read(context.Background(), f); err != nil || got.Value != want {
			t.Errorf("Read(%s) = %+v, %v; want %q", f, got, err, want)
		}
	}
}
