package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
)

func TestApplier(t *testing.T) {
	contracttest.Applier(t, func(*testing.T) contracttest.Opened {
		a := fake.NewApplier(catalog.EffectDotfile)
		return contracttest.Opened{Applier: a, Package: catalog.Setting{Name: "starship"}, Edit: func(*testing.T) { a.Edit("starship", "yours") }}
	})
}
