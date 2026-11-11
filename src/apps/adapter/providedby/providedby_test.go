package providedby_test

import (
	"testing"

	"github.com/jterrazz/studio-station/src/apps/adapter/providedby"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local/localtest"
)

func TestInstaller(t *testing.T) {
	sample := catalog.ProvidedByInstall{Item: "desk-app", Commands: []string{"dk"}}
	contracttest.Installer(t, sample, func(t *testing.T, seed contracttest.Seed) contract.Installer {
		run := localtest.NewRunner(t)
		if !seed.Fails {
			run.Paths = map[string]string{"dk": "/home/u/.local/bin/dk"}
		}
		return providedby.New(run)
	})
}
