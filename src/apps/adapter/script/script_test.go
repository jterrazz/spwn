package script_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/src/apps/adapter/script"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

func exits(code int) func(local.Cmd) (local.Output, error) {
	return func(local.Cmd) (local.Output, error) { return local.Output{ExitCode: code}, nil }
}

func TestInstaller(t *testing.T) {
	contracttest.Installer(t, catalog.ScriptInstall{URL: "https://claude.ai/install.sh"}, func(_ *testing.T, seed contracttest.Seed) contract.Installer {
		code := 0
		if seed.Fails {
			code = 1
		}
		return script.New(local.Played(exits(code), "bash"), "/home/u", false)
	})
}

func TestAnInstallScriptRunsUnderBashWithTheTerminalKeptAndItsEnvironment(t *testing.T) {
	var ran []local.Cmd
	run := local.Played(func(c local.Cmd) (local.Output, error) {
		ran = append(ran, c)
		return local.Output{}, nil
	}, "bash")

	err := script.New(run, "/home/u", false).Install(context.Background(), []catalog.Install{catalog.ScriptInstall{URL: "https://x/install.sh", Env: []string{"RUNZSH=no"}, Args: []string{"--unattended"}}}, nil)

	want := []string{"-c", `script="$(curl -fsSL "$0")" && exec bash -c "$script" bash "$@"`, "https://x/install.sh", "--unattended"}
	if err != nil || len(ran) != 1 || !slices.Equal(ran[0].Args, want) || !slices.Equal(ran[0].Env, []string{"RUNZSH=no"}) {
		t.Fatalf("ran %+v %v", ran, err)
	}
}

func TestAVariableUnderTheHomeIsGivenItsPath(t *testing.T) {
	var ran []local.Cmd
	run := local.Played(func(c local.Cmd) (local.Output, error) {
		ran = append(ran, c)
		return local.Output{}, nil
	}, "bash")
	in := catalog.ScriptInstall{URL: "https://x/get", Env: []string{"HELM_INSTALL_DIR=~/.local/bin", "USE_SUDO=false", "NOTE=a=~/b"}}

	err := script.New(run, "/home/u", false).Install(context.Background(), []catalog.Install{in}, nil)

	if want := []string{"HELM_INSTALL_DIR=/home/u/.local/bin", "USE_SUDO=false", "NOTE=a=~/b"}; err != nil || len(ran) != 1 || !slices.Equal(ran[0].Env, want) {
		t.Fatalf("ran %+v %v; want the env %v", ran, err, want)
	}
}
