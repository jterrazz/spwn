package wire

import (
	"context"
	"testing"

	awakefake "github.com/jterrazz/studio-station/apps/engine/awake/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/awake/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func present() map[coremachine.Need]coremachine.Provider {
	return map[coremachine.Need]coremachine.Provider{coremachine.Hypervisor: {Name: "hv", Look: func() bool { return true }}}
}

func TestALaptopAndADesktopDifferByWhatTheirPowerReports(t *testing.T) {
	laptop := profileOf(context.Background(), nil, "mac", "macbook", awakefake.NewPower(contracttest.Laptop()), present)
	desktop := profileOf(context.Background(), nil, "mac", "mini", awakefake.NewPower(contracttest.Desktop()), present)
	if !laptop.Caps.Battery || !laptop.Caps.Lid || !laptop.Caps.Engine || laptop.Caps.Provides[coremachine.Hypervisor].Look == nil || !laptop.Caps.Jobs {
		t.Errorf("the laptop's capabilities: %+v", laptop.Caps)
	}
	if desktop.Caps.Battery || desktop.Caps.Lid || !desktop.Caps.Thermal || !desktop.Caps.Engine {
		t.Errorf("the desktop's capabilities: %+v", desktop.Caps)
	}
	if laptop.Kind != coremachine.Computer || legacyKind(laptop) != catalog.Mac || laptop.Ref.Name != "macbook" {
		t.Errorf("the laptop's profile: %+v", laptop)
	}
}

func TestAPinnedVMRunsNoEngineAndHoldsNoPower(t *testing.T) {
	for pin, want := range map[string]catalog.MachineKind{"linux": catalog.Linux, "macos": catalog.MacOS} {
		p := profileOf(context.Background(), nil, pin, "dev", awakefake.NewPower(contracttest.Laptop()), present)
		if p.Kind != coremachine.Workspace || legacyKind(p) != want || p.Caps.Engine || p.Caps.Battery || p.Caps.Provides != nil {
			t.Errorf("%s: %+v", pin, p)
		}
	}
}
