//go:build linux

package linux

import (
	"os"
	"testing"
)

func TestTheHardwareVersionIsTheOSVersionNotTheKernelRelease(t *testing.T) {
	system, _ := hardware(os.ReadFile)
	if want := osVersion(os.ReadFile); system.Version != want {
		t.Fatalf("version = %q, want the os-release %q", system.Version, want)
	}
}
