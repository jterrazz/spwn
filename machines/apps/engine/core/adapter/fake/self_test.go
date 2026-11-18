package fake

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestTheFakeSelfKeepsTheContract(t *testing.T) {
	contracttest.Self(t, func(_ *testing.T, s contracttest.Scene) contract.Self {
		return NewSelf(machine.Profile{
			Ref: machine.Ref{Name: s.Name}, Kind: s.Kind,
			System:   machine.System{OS: s.OS, Version: "26.1", Arch: "arm64"},
			Hardware: machine.Hardware{Model: "Mac16,10", Memory: 16 << 30, CPUs: 10},
		})
	}, contracttest.Mac(), contracttest.MacOSWorkspace(), contracttest.LinuxWorkspace())
}

func TestTheFakePreferencesKeepTheContract(t *testing.T) {
	contracttest.Preferences(t, func(*testing.T, []string) contract.Preferences {
		return NewPreferences()
	})
}
