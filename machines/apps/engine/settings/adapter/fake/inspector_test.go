package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestInspector(t *testing.T) {
	contracttest.Inspector(t, func(*testing.T) contracttest.InspectorSubject {
		w := &world.World{Macs: map[string]*world.Mac{"mac-mini": {Facts: map[catalog.Fact]settings.Fact{
			catalog.FactFileVault: {State: catalog.StateOK, Value: "on"},
			catalog.FactFirewall:  {State: catalog.StateAbsent, Value: "off"},
		}}}}
		return contracttest.InspectorSubject{Inspector: fake.NewInspector(w, "mac-mini"), On: catalog.FactFileVault, Off: catalog.FactFirewall, Unreadable: catalog.FactSIP}
	})
}
