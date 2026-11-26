package fake

import (
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestTheFakeHypervisorKeepsTheContract(t *testing.T) {
	contracttest.Hypervisor(t, func(*testing.T) contracttest.Subject {
		w := &world.World{}
		return contracttest.Subject{
			Hypervisor: HypervisorOf(w, "mac-mini", "vz", nil),
			End: func(r contract.Run, how machines.EndReason) {
				end := machines.RunEnd{Reason: how}
				if how == machines.Crashed {
					end.ExitCode = 1
				}
				End(r, end)
			},
			Restart: func() contract.Hypervisor { return HypervisorOf(w, "mac-mini", "vz", nil) },
			Gone: func(vm string) {
				w.Lock()
				defer w.Unlock()
				delete(w.Mac("mac-mini").VMs, vm)
			},
			ByHand: func(vm string) {
				w.Lock()
				defer w.Unlock()
				w.Mac("mac-mini").VMs[vm] = &world.VM{Disk: true}
			},
		}
	})
}

func TestAStoppedVMsLogReadsItsConsoleThenItsHypervisorsLines(t *testing.T) {
	w := &world.World{}
	w.Mac("mac-mini").VMs = map[string]*world.VM{"api": {
		Disk:       true,
		Console:    []string{"[ 4.1] boot", " ", "[ 4.2] power off"},
		Hypervisor: []string{"Error: exited", "[ 9.9]", "Error: again"},
	}}
	h := HypervisorOf(w, "mac-mini", "vz", nil)
	got, err := h.Console(t.Context(), "api", 0)
	want := []string{"[ 4.1] boot", "[ 4.2] power off", "Error: exited", "Error: again"}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("console = %q, %v; want %q", got, err, want)
	}
	if got, _ = h.Console(t.Context(), "api", 1); !slices.Equal(got, []string{"[ 4.2] power off", "Error: again"}) {
		t.Errorf("last 1 of each = %q; want the last console line then the last hypervisor line", got)
	}
}

func TestABootTheWorldScriptsToFailWritesItsConsoleLines(t *testing.T) {
	w := &world.World{}
	w.Script = append(w.Script, world.Fault{Call: "Hypervisor.Boot"})
	w.Mac("mac-mini").VMs = map[string]*world.VM{"web": {Disk: true}}
	h := HypervisorOf(w, "mac-mini", "vz", nil)
	if _, err := h.Boot(t.Context(), "web", contract.BootOptions{}); !errors.Is(err, ErrScripted) {
		t.Fatalf("boot: %v; want the scripted refusal", err)
	}
	lines, err := h.Console(t.Context(), "web", 40)
	if err != nil || !slices.Equal(lines, bootRefused()) {
		t.Errorf("console of web: %q, %v; want the lines the refused boot wrote, %q", lines, err, bootRefused())
	}
}

func TestAHypervisorTheWorldNamesAnotherThanItsOwnHasNoPackageToInstall(t *testing.T) {
	w := &world.World{}
	h := HypervisorOf(w, "mac-mini", "own", nil)
	if got := h.Package(); got != "own" {
		t.Errorf("a Mac of the world's default hypervisor installs %q, want its own", got)
	}
	w.Mac("mac-mini").Hypervisor = "vz"
	if got := h.Package(); got != "" {
		t.Errorf("a Mac whose world names vz installs %q, want nothing", got)
	}
}
