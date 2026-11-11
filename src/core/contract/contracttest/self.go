package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/src/core/contract"
	"github.com/jterrazz/studio-station/src/core/machine"
)

type Scene struct {
	// value: a machine an adapter is handed, by what Self must answer of it.
	Name string
	Kind machine.Kind
	OS   machine.OS
}

func Mac() Scene { return Scene{Name: "mac-mini", Kind: machine.Computer, OS: machine.MacOS} }

func MacOSWorkspace() Scene {
	return Scene{Name: "dev", Kind: machine.Workspace, OS: machine.MacOS}
}

func LinuxWorkspace() Scene {
	return Scene{Name: "agents", Kind: machine.Workspace, OS: machine.Linux}
}

func Self(t *testing.T, open func(t *testing.T, s Scene) contract.Self, scenes ...Scene) {
	for _, s := range scenes {
		t.Run("a "+string(s.OS)+" "+string(s.Kind)+" names itself, its kind, its system and its size", func(t *testing.T) {
			ref, kind, system, hw, err := open(t, s).Read(context.Background())
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if ref.Name != s.Name || ref.On != "" {
				t.Errorf("Ref = %+v, want the name %q on no other machine", ref, s.Name)
			}
			if kind != s.Kind || system.OS != s.OS {
				t.Errorf("kind %q on %q, want %q on %q", kind, system.OS, s.Kind, s.OS)
			}
			if system.Arch == "" || hw.Memory <= 0 || hw.CPUs <= 0 {
				t.Errorf("System %+v, Hardware %+v: a machine has an architecture, memory and CPUs", system, hw)
			}
		})
	}
}
