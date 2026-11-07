package tart

import (
	"context"
	"encoding/json"
	"io/fs"
	"maps"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/machines/contract"
	"github.com/jterrazz/studio-station/src/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/platform/local/localtest"
)

const (
	bin   = "/home/o/.local/bin/tart"
	state = "/s"
)

type lockedFS struct {
	mu sync.Mutex
	*localtest.FS
}

func (f *lockedFS) ReadFile(p string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.FS.ReadFile(p)
}

func (f *lockedFS) WriteFile(p string, data []byte, perm fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.FS.WriteFile(p, data, perm)
}

func (f *lockedFS) MkdirAll(p string, perm fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.FS.MkdirAll(p, perm)
}

func (f *lockedFS) Remove(p string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.FS.Remove(p)
}

type yielding struct{}

func (yielding) Now() time.Time { return time.Unix(0, 0) }

func (yielding) Sleep(ctx context.Context, _ time.Duration) error {
	runtime.Gosched()
	return ctx.Err()
}

func recorded(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

type kernel struct {
	mu    sync.Mutex
	names map[int]string
	files map[int][]string
}

func (k *kernel) Named(prefix string) ([]int, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	var pids []int
	for pid, name := range k.names {
		if strings.HasPrefix(name, prefix) {
			pids = append(pids, pid)
		}
	}
	slices.Sort(pids)
	return pids, nil
}

func (k *kernel) Files(pid int) ([]string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.files[pid], nil
}

func (k *kernel) start(pid int, name string, files ...string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.names[pid] = name
	k.files[pid] = files
}

type machine struct {
	run    *localtest.Runner
	fs     *lockedFS
	kernel *kernel
	disks  map[string]string
}

func newMachine(t *testing.T) *machine {
	run := localtest.NewRunner(t)
	run.Paths["tart"] = bin
	return &machine{run: run, fs: &lockedFS{FS: localtest.NewFS()}, kernel: &kernel{names: map[int]string{}, files: map[int][]string{}}, disks: map[string]string{}}
}

func (m *machine) hypervisor() contract.Hypervisor {
	return New(m.run, m.fs, yielding{}, m.kernel, state)
}

func (m *machine) listing() local.Output {
	type row struct{ Source, Name, State string }
	rows := make([]row, 0, len(m.disks)+1)
	rows = append(rows, row{Source: "OCI", Name: contracttest.Image, State: "stopped"})
	for _, vm := range slices.Sorted(maps.Keys(m.disks)) {
		rows = append(rows, row{Source: "local", Name: vm, State: m.disks[vm]})
	}
	data, _ := json.Marshal(rows)
	return local.Output{Stdout: string(data)}
}

func (m *machine) boot(vm string, extra ...string) {
	argv := append([]string{bin, "run", "--no-graphics", "--no-audio", "--serial-path", state + "/consoles/" + vm}, extra...)
	m.run.On("sh", wrapped(append(argv, vm), state+"/runs/"+vm+".exit").Args...).Answer(func() local.Output {
		m.disks[vm] = "running"
		return local.Output{}
	})
}

func (m *machine) end(r contract.Run, how machines.EndReason) {
	code := "0"
	if how == machines.GuestPoweroff {
		_ = m.fs.WriteFile(state+"/consoles/"+r.VM(), []byte("[  OK  ] Reached target Power-Off.\nstation-intent=poweroff\n"), 0o600)
	}
	if how == machines.Crashed {
		code = "1"
	}
	_ = m.fs.WriteFile(state+"/runs/"+r.VM()+".exit", []byte(code+"\n"), 0o600)
	m.disks[r.VM()] = "stopped"
	m.run.Exit(r.PID())
}

func TestTartKeepsTheHypervisorContract(t *testing.T) {
	contracttest.Hypervisor(t, func(t *testing.T) contracttest.Subject {
		m := newMachine(t)
		vm, renamed := contracttest.VM, contracttest.Renamed
		m.run.On(bin, "pull", contracttest.Image)
		m.run.On(bin, "clone", contracttest.Image, vm).Answer(func() local.Output {
			m.disks[vm] = "stopped"
			return local.Output{}
		})
		m.run.On(bin, "set", vm, "--cpu", "4", "--memory", "8192", "--disk-size", "50")
		m.run.On(bin, "list", "--source", "local", "--format", "json").Answer(m.listing)
		m.boot(vm)
		m.run.On(bin, "ip", vm, "--wait", "5").Reply(local.Output{Stdout: "192.168.64.5\n"})
		m.run.On(bin, "rename", vm, renamed).Answer(func() local.Output {
			m.disks[renamed] = m.disks[vm]
			delete(m.disks, vm)
			return local.Output{}
		})
		for _, name := range []string{vm, renamed} {
			m.run.On(bin, "delete", name).Answer(func() local.Output {
				if m.disks[name] == "running" {
					return local.Output{ExitCode: 1, Stderr: "Error: VM \"" + name + "\" is running\n"}
				}
				delete(m.disks, name)
				return local.Output{}
			})
		}
		return contracttest.Subject{Hypervisor: m.hypervisor(), End: m.end, Restart: m.hypervisor}
	})
}

func TestDisksKeepLocalVMsAndTheirSize(t *testing.T) {
	m := newMachine(t)
	m.run.On(bin, "list", "--source", "local", "--format", "json").Reply(local.Output{Stdout: recorded(t, "list.json")})
	disks, err := m.hypervisor().Disks(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := []contract.Disk{
		{VM: "station-image-ubuntu-1", Bytes: 7e9, State: contract.DiskStopped},
		{VM: "dev", Bytes: 12e9, State: contract.DiskRunning},
		{VM: "paused", Bytes: 9e9, State: contract.DiskSuspended},
	}
	if !slices.Equal(disks, want) {
		t.Fatalf("disks %+v, want %+v without the OCI entry", disks, want)
	}
}

func TestABridgedVMBootsOnTheMacsDefaultInterface(t *testing.T) {
	m := newMachine(t)
	m.run.On("route", "-n", "get", "default").Reply(local.Output{Stdout: "   route to: default\ndestination: default\n  interface: en7\n"})
	m.boot("dev", "--net-bridged=en7")
	r, err := m.hypervisor().Boot(t.Context(), "dev", contract.BootOptions{Network: machines.Network{Bridged: true}})
	if err != nil {
		t.Fatal(err)
	}
	m.end(r, machines.GuestPoweroff)
	<-r.Ended()
}

func TestTheEndOfAnOlderRunKeepsTheNewerRunAdoptable(t *testing.T) {
	m := newMachine(t)
	m.boot("dev")
	h := m.hypervisor()
	older, err := h.Boot(t.Context(), "dev", contract.BootOptions{})
	if err != nil {
		t.Fatal(err)
	}
	newer, err := h.Boot(t.Context(), "dev", contract.BootOptions{})
	if err != nil {
		t.Fatal(err)
	}
	m.run.Exit(older.PID())
	<-older.Ended()
	if again, ok := m.hypervisor().Adopt(t.Context(), "dev"); !ok || again.PID() != newer.PID() {
		t.Errorf("after the older run ended, a restart adopted %v (ok %v), want the newer run", again, ok)
	}
	m.end(newer, machines.GuestReboot)
	<-newer.Ended()
}

func TestARunThatEndedWhileNoEngineWatchedIsNotAdopted(t *testing.T) {
	m := newMachine(t)
	_ = m.fs.WriteFile(state+"/runs/dev.pid", []byte("77\n"), 0o600)
	if _, ok := m.hypervisor().Adopt(t.Context(), "dev"); ok {
		t.Fatalf("a run whose process is gone was adopted")
	}
	if _, err := m.fs.ReadFile(state + "/runs/dev.pid"); err == nil {
		t.Errorf("the pid file of the ended run was kept")
	}
}

func TestAFailedCommandCarriesItsStderr(t *testing.T) {
	m := newMachine(t)
	m.run.On(bin, "delete", "dev").Reply(local.Output{ExitCode: 1, Stderr: "no such VM"})
	err := m.hypervisor().Delete(t.Context(), "dev")
	if err == nil || !strings.Contains(err.Error(), "no such VM") {
		t.Fatalf("err = %v", err)
	}
}

func TestAVMIsMeasuredByTheServiceThatHoldsItsGuestNotByItsWrapper(t *testing.T) {
	m := newMachine(t)
	m.boot("dev")
	h := m.hypervisor()
	r, err := h.Boot(t.Context(), "dev", contract.BootOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Holder(); got != 0 {
		t.Errorf("before Virtualization started the guest, the run's holder is %d; want none", got)
	}
	const home = "/Users/o/.local/share/station/vms/vms/"
	m.kernel.start(r.PID()+1, "tart", home+"dev/config.json")
	m.kernel.start(7000, "com.apple.Virtualization.Virtual", home+"other/nvram.bin", home+"other/disk.img")
	m.kernel.start(7001, "com.apple.Virtualization.Virtual", home+"dev/nvram.bin", home+"dev/disk.img")
	if got := r.Holder(); got != 7001 {
		t.Errorf("the run of dev is held by %d; want 7001, the Virtualization service holding dev's disk", got)
	}
	adopted, ok := m.hypervisor().Adopt(t.Context(), "dev")
	if !ok || adopted.Holder() != 7001 {
		t.Errorf("after an engine restart the adopted run is held by %v (ok %v); want 7001", adopted, ok)
	}
	m.end(r, machines.GuestPoweroff)
	<-r.Ended()
}

func (m *machine) crash(r contract.Run, console, log string) {
	_ = m.fs.WriteFile(state+"/consoles/"+r.VM(), []byte(console), 0o600)
	_ = m.fs.WriteFile(state+"/logs/"+r.VM()+".log", []byte(log), 0o600)
	_ = m.fs.WriteFile(state+"/runs/"+r.VM()+".exit", []byte("1\n"), 0o600)
	m.disks[r.VM()] = "stopped"
	m.run.Exit(r.PID())
}

func TestARunThatCrashedEndsWithTheLinesThatTellWhy(t *testing.T) {
	cases := []struct {
		name, console string
		lines         []string
		cause         string
	}{
		{"out of memory", "console-oom.txt", []string{
			"[ 4127.551204] Out of memory: Killed process 812 (k3s-server) total-vm:2318764kB, anon-rss:1987312kB, file-rss:0kB, shmem-rss:0kB, UID:0 pgtables:4312kB oom_score_adj:0",
			"[ 4127.602331] systemd[1]: k3s.service: A process of this unit has been killed by the OOM killer.",
			"[ 4127.611008] systemd[1]: k3s.service: Main process exited, code=killed, status=9/KILL",
		}, "out of memory"},
		{"kernel panic", "console-panic.txt", []string{
			"[   92.004402]  dump_backtrace+0xa0/0x128",
			"[   92.004517] ---[ end Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000009 ]---",
			"Error: The virtual machine stopped unexpectedly.",
		}, "kernel panic"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := newMachine(t)
			m.boot("dev")
			r, err := m.hypervisor().Boot(t.Context(), "dev", contract.BootOptions{})
			if err != nil {
				t.Fatal(err)
			}
			m.crash(r, recorded(t, c.console), recorded(t, "tart-run.log"))
			end := <-r.Ended()
			if end.Reason != machines.Crashed || !slices.Equal(end.LastLines, c.lines) {
				t.Fatalf("the run ended as %q with %q, want a crash with %q", end.Reason, end.LastLines, c.lines)
			}
			if got := machines.DownCause(end); got != c.cause {
				t.Errorf("its cause reads %q, want %q", got, c.cause)
			}
		})
	}
}

func TestTheConsoleIsItsLastMeaningfulLinesThenTheHypervisorsOwn(t *testing.T) {
	m := newMachine(t)
	h := m.hypervisor()
	if lines, err := h.Console(t.Context(), "dev", 40); err != nil || len(lines) != 0 {
		t.Fatalf("a VM that never ran has %q (%v), want none", lines, err)
	}
	_ = m.fs.WriteFile(state+"/consoles/dev", []byte(recorded(t, "console-oom.txt")+"station-intent=poweroff\n"), 0o600)
	_ = m.fs.WriteFile(state+"/logs/dev.log", []byte(recorded(t, "tart-run.log")), 0o600)
	lines, err := h.Console(t.Context(), "dev", 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"[ 4127.611008] systemd[1]: k3s.service: Main process exited, code=killed, status=9/KILL",
		"[ 4131.002116] systemd[1]: Reached target Power-Off.",
		"Error: The virtual machine stopped unexpectedly.",
	}
	if !slices.Equal(lines, want) {
		t.Errorf("the last 2 of each output: %q, want %q", lines, want)
	}
	all, _ := h.Console(t.Context(), "dev", 0)
	if len(all) != 10 || all[0] != "[    0.000000] Booting Linux on physical CPU 0x0000000000 [0x610f0000]" {
		t.Errorf("the whole log holds %d lines from %q, want 10 from the boot", len(all), all[0])
	}
}
