package contracttest

import (
	"context"
	"slices"
	"testing"
	"time"

	imagescontract "github.com/jterrazz/studio-station/apps/engine/images/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

const (
	VM      = "dev"
	Renamed = "dev-2"
	Image   = "ghcr.io/example/linux:1"
	Wait    = 5 * time.Second
)

type Subject struct {
	// value: the adapter on a machine that holds no VM yet.
	Hypervisor contract.Hypervisor
	// value: ends a run the way its guest would: a poweroff, a reboot, a crash.
	End func(r contract.Run, how machines.EndReason)
	// value: the engine restarting: a new adapter over the same machine.
	Restart func() contract.Hypervisor
}

type progress struct{ steps, done int }

func (p *progress) Step(string, string)   { p.steps++ }
func (p *progress) Done(string, error)    { p.done++ }
func image() imagescontract.ImageLocation { return imagescontract.ImageLocation{Ref: Image} }

func Hypervisor(t *testing.T, subject func(t *testing.T) Subject) {
	t.Run("an image is fetched with its progress, then cloned into a VM it holds stopped", func(t *testing.T) {
		s := subject(t)
		created(t, s)
		if got := state(t, s.Hypervisor, VM); got != contract.DiskStopped {
			t.Errorf("the cloned VM reads %q, want stopped", got)
		}
		if lines, err := s.Hypervisor.Console(t.Context(), VM, 40); err != nil || len(lines) != 0 {
			t.Errorf("a VM that never ran has the console %q (%v), want none", lines, err)
		}
	})
	for _, c := range []struct {
		how  machines.EndReason
		name string
	}{
		{machines.GuestPoweroff, "a guest poweroff ends the run as a poweroff"},
		{machines.GuestReboot, "a guest reboot ends the run as a reboot, never as a poweroff"},
		{machines.Crashed, "a run that crashes ends as a crash, with its exit code"},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := subject(t)
			created(t, s)
			r := booted(t, s.Hypervisor)
			s.End(r, c.how)
			end := ended(t, r)
			if end.Reason != c.how {
				t.Fatalf("the run ended as %q, want %q", end.Reason, c.how)
			}
			if c.how == machines.Crashed && end.ExitCode == 0 {
				t.Errorf("a crash ended with exit 0")
			}
		})
	}
	t.Run("a run outlives an engine restart and is adopted, its end read as before", func(t *testing.T) {
		s := subject(t)
		created(t, s)
		r := booted(t, s.Hypervisor)
		again := s.Restart()
		adopted, ok := again.Adopt(t.Context(), VM)
		if !ok || adopted.VM() != VM || adopted.PID() != r.PID() {
			t.Fatalf("after a restart: adopted %v (ok %v), want the run of %s", adopted, ok, VM)
		}
		if _, ok := again.Adopt(t.Context(), Renamed); ok {
			t.Errorf("a VM with no run was adopted")
		}
		s.End(adopted, machines.GuestReboot)
		if end := ended(t, adopted); end.Reason != machines.GuestReboot {
			t.Errorf("the adopted run ended as %q, want a reboot", end.Reason)
		}
	})
	t.Run("a running VM is not deleted: the delete is refused until its run has ended", func(t *testing.T) {
		s := subject(t)
		created(t, s)
		r := booted(t, s.Hypervisor)
		if err := s.Hypervisor.Delete(t.Context(), VM); err == nil {
			t.Fatal("a running VM was deleted")
		}
		if got := state(t, s.Hypervisor, VM); got != contract.DiskRunning {
			t.Errorf("after a refused delete it reads %q, want running", got)
		}
		s.End(r, machines.GuestPoweroff)
		ended(t, r)
		if err := s.Hypervisor.Delete(t.Context(), VM); err != nil {
			t.Fatal(err)
		}
		if got := names(t, s.Hypervisor); len(got) != 0 {
			t.Errorf("after the delete it holds %v, want none", got)
		}
	})
	t.Run("a renamed VM is held under its new name, a deleted one no longer", func(t *testing.T) {
		s := subject(t)
		created(t, s)
		if err := s.Hypervisor.Rename(t.Context(), VM, Renamed); err != nil {
			t.Fatal(err)
		}
		if got := names(t, s.Hypervisor); !slices.Equal(got, []string{Renamed}) {
			t.Fatalf("after the rename it holds %v, want [%s]", got, Renamed)
		}
		if err := s.Hypervisor.Delete(t.Context(), Renamed); err != nil {
			t.Fatal(err)
		}
		if got := names(t, s.Hypervisor); len(got) != 0 {
			t.Errorf("after the delete it holds %v, want none", got)
		}
	})
}

func created(t *testing.T, s Subject) {
	t.Helper()
	p := &progress{}
	if err := s.Hypervisor.Fetch(t.Context(), image(), p); err != nil {
		t.Fatal(err)
	}
	if p.steps == 0 || p.done == 0 {
		t.Errorf("the fetch told %d steps and %d ends, want its progress", p.steps, p.done)
	}
	if err := s.Hypervisor.Clone(t.Context(), image(), VM); err != nil {
		t.Fatal(err)
	}
	if err := s.Hypervisor.Resize(t.Context(), VM, machines.Size{CPUs: 4, Memory: 8 << 30, Disk: 50e9}); err != nil {
		t.Fatal(err)
	}
}

func booted(t *testing.T, h contract.Hypervisor) contract.Run {
	t.Helper()
	r, err := h.Boot(t.Context(), VM, contract.BootOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if r.VM() != VM {
		t.Errorf("the run is of %q, want %q", r.VM(), VM)
	}
	if got := state(t, h, VM); got != contract.DiskRunning {
		t.Errorf("a booted VM reads %q, want running", got)
	}
	if addr, err := h.Address(t.Context(), VM, Wait); err != nil || !addr.IsValid() {
		t.Errorf("a booted VM has the address %v (%v), want one", addr, err)
	}
	return r
}

func ended(t *testing.T, r contract.Run) machines.RunEnd {
	t.Helper()
	wait, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	select {
	case end := <-r.Ended():
		return end
	case <-wait.Done():
		t.Fatal("the run never ended")
	}
	return machines.RunEnd{}
}

func names(t *testing.T, h contract.Hypervisor) []string {
	t.Helper()
	disks, err := h.Disks(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(disks))
	for _, d := range disks {
		out = append(out, d.VM)
	}
	return out
}

func state(t *testing.T, h contract.Hypervisor, vm string) contract.DiskState {
	t.Helper()
	disks, err := h.Disks(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range disks {
		if d.VM == vm {
			return d.State
		}
	}
	return ""
}
