package shell_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/shell"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func build(_ *testing.T, seed contracttest.Seed) contract.Installer {
	return shell.New(local.Played(func(c local.Cmd) (local.Output, error) {
		if seed.Fails || len(c.Args) != 2 || c.Args[0] != "-c" {
			return local.Output{ExitCode: 1}, nil
		}
		return local.Output{}, nil
	}, "bash"))
}

func TestInstaller(t *testing.T) {
	contracttest.Installer(t, catalog.ShellInstall{Line: "npx playwright install chromium"}, build)
}

func TestAnInstallerThatCanUndo(t *testing.T) {
	contracttest.Installer(t, catalog.ShellInstall{Line: "make tool", Undo: "make untool"}, build)
}

func TestAnUndoRunsAsItsLineDoesAndALineWithNoneIsRefusedBeforeAnyRuns(t *testing.T) {
	var ran []local.Cmd
	run := local.Played(func(c local.Cmd) (local.Output, error) {
		ran = append(ran, c)
		return local.Output{}, nil
	}, "bash")
	i := shell.New(run)

	err := i.Remove(context.Background(), []catalog.Install{catalog.ShellInstall{Line: "a", Undo: "undo a"}, catalog.ShellInstall{Line: "b"}}, nil)
	if !errors.Is(err, contract.ErrCannot) || len(ran) != 0 {
		t.Fatalf("a call holding a line with no undo ran %+v and answered %v", ran, err)
	}

	err = i.Remove(context.Background(), []catalog.Install{catalog.ShellInstall{Line: "a", Undo: "undo a"}}, nil)
	if want := []string{"-c", "undo a"}; err != nil || len(ran) != 1 || !slices.Equal(ran[0].Args, want) {
		t.Fatalf("ran %+v %v; want bash %v", ran, err, want)
	}
}
