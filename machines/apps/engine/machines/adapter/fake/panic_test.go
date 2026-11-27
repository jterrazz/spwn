package fake

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func panics(f func()) (broke bool) {
	defer func() { broke = recover() != nil }()
	f()
	return false
}

func answersWithin(t *testing.T, what string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		f()
		close(done)
	}()
	bound, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case <-done:
	case <-bound.Done():
		t.Fatalf("%s after a world panic did not answer: the fake kept the world locked", what)
	}
}

func TestAGuestDialTheWorldScriptsToPanicLeavesTheWorldFree(t *testing.T) {
	for _, call := range []string{"Guest.Dial", "Guest.NoRoute"} {
		t.Run(call, func(t *testing.T) {
			w := &world.World{ThisMac: "this-mac"}
			w.Mac("this-mac").VMs["dev"] = &world.VM{PID: 1001, Guest: &world.Guest{}}
			w.Script = []world.Fault{{Call: call, Times: 1, Panic: true}}
			dial := func() error {
				_, err := GuestsOf(w).Dial(t.Context(), contract.GuestTarget{VM: "dev"})
				return err
			}
			if !panics(func() { _ = dial() }) {
				t.Fatalf("%s did not panic: the test proves nothing", call)
			}
			answersWithin(t, "the next dial", func() {
				if err := dial(); err != nil {
					t.Errorf("the next dial: %v; want the guest", err)
				}
			})
		})
	}
}

func TestARunEndTheWorldScriptsToPanicLeavesTheWorldFree(t *testing.T) {
	w := &world.World{}
	w.Mac("mac-mini").VMs = map[string]*world.VM{"web": {Disk: true}}
	h := HypervisorOf(w, "mac-mini", "vz", nil)
	run, err := h.Boot(t.Context(), "web", contract.BootOptions{})
	if err != nil {
		t.Fatal(err)
	}
	w.Mac("mac-mini").VMs["web"].PID, w.Mac("mac-mini").VMs["web"].Ended = 0, &machines.RunEnd{Reason: machines.GuestPoweroff}
	w.Script = []world.Fault{{Call: "Hypervisor.Ended", Times: 1, Panic: true}}
	if !panics(func() { run.Ended() }) {
		t.Fatal("Hypervisor.Ended did not panic: the test proves nothing")
	}
	answersWithin(t, "the next boot", func() {
		if _, err := h.Boot(t.Context(), "web", contract.BootOptions{}); err != nil {
			t.Errorf("the next boot: %v", err)
		}
	})
}
