package git_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/git"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
)

// value: a runner that answers `git config --global` over a map, as git does.
type config struct {
	local.Runner
	keys   map[string]string
	writes int
}

func (c *config) Output(_ context.Context, cmd local.Cmd) (local.Output, error) {
	args := cmd.Args
	if cmd.Name != "git" || len(args) < 3 || args[0] != "config" || args[1] != "--global" {
		return local.Output{ExitCode: 97, Stderr: "unexpected invocation"}, nil
	}
	switch rest := args[2:]; {
	case len(rest) == 1:
		v, ok := c.keys[rest[0]]
		if !ok {
			return local.Output{ExitCode: 1}, nil
		}
		return local.Output{Stdout: v + "\n"}, nil
	case len(rest) == 2 && rest[0] == "--unset":
		delete(c.keys, rest[1])
		c.writes++
	case len(rest) == 2:
		c.keys[rest[0]] = rest[1]
		c.writes++
	}
	return local.Output{}, nil
}

var identity = catalog.Setting{Name: "git-identity", Effect: catalog.ProcGitIdentity}

func TestApplier(t *testing.T) {
	contracttest.Applier(t, func(*testing.T) contracttest.Opened {
		c := &config{keys: map[string]string{}}
		return contracttest.Opened{
			Applier: git.NewIdentity(c),
			Package: identity,
			Values:  map[string]string{"name": "Jane Doe", "email": "jane@example.com"},
			Edit:    func(*testing.T) { c.keys["user.email"] = "mine@example.com" },
		}
	})
}

func TestApplyingTwiceWritesNothingTheSecondTime(t *testing.T) {
	c := &config{keys: map[string]string{}}
	applier := git.NewIdentity(c)
	values := map[string]string{"name": "Jane Doe", "email": "jane@example.com"}
	if _, err := applier.Apply(context.Background(), identity, values); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	written := c.writes
	change, err := applier.Apply(context.Background(), identity, values)
	if err != nil || c.writes != written {
		t.Fatalf("second Apply wrote %d more, err %v", c.writes-written, err)
	}
	if got := change.Steps[0].Detail; !strings.HasPrefix(got, "already") {
		t.Fatalf("second Apply said %q; want it already there", got)
	}
}

func TestAnIdentityMachinesDidNotWriteIsKeptByUndo(t *testing.T) {
	c := &config{keys: map[string]string{"user.name": "Jane Doe", "user.email": "jane@example.com"}}
	applier := git.NewIdentity(c)
	if _, err := applier.Apply(context.Background(), identity, map[string]string{"name": "Jane Doe", "email": "jane@example.com"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if _, err := applier.Undo(context.Background(), identity); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if c.keys["user.name"] != "Jane Doe" || c.keys["user.email"] != "jane@example.com" {
		t.Fatalf("Undo took an identity Machines never wrote: %v", c.keys)
	}
}

func TestABadValueIsRefusedBeforeAnythingIsWritten(t *testing.T) {
	for name, values := range map[string]map[string]string{
		"no name":        {"email": "jane@example.com"},
		"blank email":    {"name": "Jane Doe", "email": "  "},
		"a newline":      {"name": "Jane\nDoe", "email": "jane@example.com"},
		"a bell in mail": {"name": "Jane Doe", "email": "jane@example.com\a"},
	} {
		t.Run(name, func(t *testing.T) {
			c := &config{keys: map[string]string{}}
			if _, err := git.NewIdentity(c).Apply(context.Background(), identity, values); err == nil {
				t.Fatal("Apply accepted it")
			}
			if c.writes != 0 {
				t.Fatalf("a refused value still wrote %d keys", c.writes)
			}
		})
	}
}
