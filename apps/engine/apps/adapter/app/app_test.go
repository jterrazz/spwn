package app_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/app"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestInstaller(t *testing.T) {
	contracttest.Installer(t, catalog.AppInstall{Hint: "download it from its website"}, func(*testing.T, contracttest.Seed) contract.Installer {
		return app.New()
	})
}
