package calls_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	machinesfake "github.com/jterrazz/studio-station/apps/engine/machines/adapter/fake"
	machines "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	machinescontract "github.com/jterrazz/studio-station/apps/engine/machines/contract"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func lockingDown(t *testing.T) *machines.Machinist {
	t.Helper()
	w := &world.World{ThisMac: "mac"}
	prefs := memoryPrefs{}
	p := machines.Prefs{Name: setupcontract.Name(prefs), Macs: machinescontract.Macs(prefs), VMUser: machinescontract.VMUser(prefs)}
	if err := p.Name.Set("mac"); err != nil {
		t.Fatal(err)
	}
	store := machinesfake.StoreOf(w, "mac")
	if err := store.Put(t.Context(), model.Record{Name: "mac-1", Image: catalog.MacOSImage, Lifetime: model.Ephemeral, MemoryBytes: 8 << 30}); err != nil {
		t.Fatal(err)
	}
	macos := catalog.Image{Name: catalog.MacOSImage, Kind: catalog.MacOS, FirstLogin: "images/macos-firstlogin.sh"}
	return machines.New(&stillMac{at: time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)}, machinesfake.HypervisorOf(w, "mac", "hypervisor", nil), p, store, noTailnet{}, &images{built: true}, &idleEngine{}, nil, machines.Options{
		ThisMachine: catalog.Mac, Images: []catalog.Image{macos}, KnownHosts: "/state/known_hosts", Logs: "/state/logs",
		Hardware:    func(context.Context) coremachine.Hardware { return coremachine.Hardware{Memory: 16 << 30, CPUs: 8} },
		GuestBinary: func() ([]byte, error) { return []byte("station"), nil },
	})
}

func TestALockdownIsRefusedForAVMUserThatIsTheImagesOwnAccount(t *testing.T) {
	m := lockingDown(t)
	for _, user := range []string{catalog.UpstreamLogin.User, "root", ""} {
		_, ok, err := m.LockdownOf(t.Context(), machinescontract.GuestTarget{VM: "mac-1", User: user})
		if err == nil || ok {
			t.Errorf("VM user %q: lockdown %v, %v; want it refused", user, ok, err)
			continue
		}
		if !strings.Contains(err.Error(), "vm_user") {
			t.Errorf("VM user %q: %q names no way out", user, err)
		}
		if !errors.As(err, new(*machinescontract.LockdownError)) {
			t.Errorf("VM user %q: %v is not a lockdown failure, so its VM would not be stopped", user, err)
		}
	}
	if _, ok, err := m.LockdownOf(t.Context(), machinescontract.GuestTarget{VM: "mac-1", User: "you"}); err != nil || !ok {
		t.Fatalf("VM user you: lockdown %v, %v; want it made", ok, err)
	}
}

func TestALockdownThatCannotBeReadIsNoLockdownFailure(t *testing.T) {
	_, _, err := lockingDown(t).LockdownOf(t.Context(), machinescontract.GuestTarget{VM: "mac-2", User: "you"})
	if err == nil || errors.As(err, new(*machinescontract.LockdownError)) {
		t.Errorf("a VM its store does not hold came back %v; want its own error, no lockdown failure", err)
	}
}

func TestALockdownsValuesCarryTheHashOfTheBinaryItPutsIn(t *testing.T) {
	l, ok, err := lockingDown(t).LockdownOf(t.Context(), machinescontract.GuestTarget{VM: "mac-1", User: "you"})
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	lines := strings.Split(strings.TrimSuffix(l.Values, "\n"), "\n")
	sum := sha256.Sum256([]byte("station"))
	if len(lines) != 5 || lines[0] != "you" || lines[2] != catalog.UpstreamLogin.User || lines[3] != catalog.UpstreamLogin.Password || lines[4] != hex.EncodeToString(sum[:]) || string(l.Binary) != "station" {
		t.Fatalf("values %q, binary %q; want the VM user, the key, the image's login and the binary's sha256", lines, l.Binary)
	}
	if l.Values == strings.Replace(l.Values, lines[4], hex.EncodeToString(make([]byte, 32)), 1) {
		t.Fatal("the hash is not a value the script reads")
	}
}

func TestALockdownFolderIsTakenOnlyAsMktempPrintsIt(t *testing.T) {
	var l machines.Lockdown
	if dir, err := l.Folder(machinescontract.GuestResult{Stdout: "/tmp/station.Ab12Cd34\n"}); err != nil || dir != "/tmp/station.Ab12Cd34" {
		t.Fatalf("a made folder came back %q, %v", dir, err)
	}
	for name, made := range map[string]machinescontract.GuestResult{
		"a failed mktemp":     {Code: 1, Stdout: "/tmp/station.Ab12Cd34\n", Stderr: "mktemp: failed\n"},
		"another folder":      {Stdout: "/var/tmp/station.Ab12Cd34\n"},
		"a path out of it":    {Stdout: "/tmp/station.Ab12/../../etc\n"},
		"a space in it":       {Stdout: "/tmp/station.Ab12 Cd34\n"},
		"two lines":           {Stdout: "/tmp/station.Ab12Cd34\n/tmp/station.Ef56Gh78\n"},
		"a command after it":  {Stdout: "/tmp/station.Ab12Cd34;rm\n"},
		"nothing":             {},
		"the template itself": {Stdout: "/tmp/station.\n"},
	} {
		if dir, err := l.Folder(made); err == nil {
			t.Errorf("%s: %q taken as the folder %q", name, made.Stdout, dir)
		}
	}
}
