package shell_test

import (
	"testing"

	"github.com/jterrazz/studio-station/src/apps/adapter/shell"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

func TestInstaller(t *testing.T) {
	contracttest.Installer(t, catalog.ShellInstall{Line: "npx playwright install chromium"}, func(_ *testing.T, seed contracttest.Seed) contract.Installer {
		return shell.New(local.Played(func(c local.Cmd) (local.Output, error) {
			if seed.Fails || len(c.Args) != 2 || c.Args[0] != "-c" {
				return local.Output{ExitCode: 1}, nil
			}
			return local.Output{}, nil
		}, "bash"))
	})
}
