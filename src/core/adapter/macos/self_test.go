package macos

import (
	"context"
	"runtime"
	"testing"

	"github.com/jterrazz/studio-station/src/core/contract"
	"github.com/jterrazz/studio-station/src/core/contract/contracttest"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/platform/local"
)

type sysctl struct {
	stdout string
	err    error
}

func (s sysctl) Output(_ context.Context, c local.Cmd) (local.Output, error) {
	if c.Name != "sysctl" || len(c.Args) != 2 || c.Args[1] != "kern.hv_vmm_present" {
		return local.Output{ExitCode: 97}, nil
	}
	return local.Output{Stdout: s.stdout}, s.err
}

func on(run Commands, name string) *Self {
	s := New(run, name)
	s.hardware = func() (machine.System, machine.Hardware) {
		return machine.System{Version: "26.1"}, machine.Hardware{Model: "Mac16,10", Memory: 16 << 30}
	}
	return s
}

func TestTheMacOSSelfKeepsTheContract(t *testing.T) {
	contracttest.Self(t, func(_ *testing.T, s contracttest.Scene) contract.Self {
		vmm := "0\n"
		if s.Kind == machine.Workspace {
			vmm = "1\n"
		}
		return on(sysctl{stdout: vmm}, s.Name)
	}, contracttest.Mac(), contracttest.MacOSWorkspace())
}

func TestAMacWhoseSysctlFailsIsStillAMac(t *testing.T) {
	_, kind, _, _, err := on(sysctl{err: local.ErrNotFound}, "mac-mini").Read(context.Background())
	if err != nil || kind != machine.Computer {
		t.Fatalf("kind %q, %v; want a Mac", kind, err)
	}
}

func TestThisMacReportsItsSizeWithoutASubprocess(t *testing.T) {
	system, hw := hardware()
	if runtime.GOOS == "darwin" && (system.Version == "" || hw.Model == "" || hw.Memory <= 0) {
		t.Fatalf("this Mac read as %+v %+v", system, hw)
	}
}
