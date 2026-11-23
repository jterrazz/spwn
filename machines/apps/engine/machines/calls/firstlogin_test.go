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
	for _, user := range []string{catalog.UpstreamLogin.User, "Admin", "root", "ROOT", "", "you.", "You.ops"} {
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
	for _, user := range []string{"you", "jterrazz.ops"} {
		if _, ok, err := m.LockdownOf(t.Context(), machinescontract.GuestTarget{VM: "mac-1", User: user}); err != nil || !ok {
			t.Fatalf("VM user %s: lockdown %v, %v; want it made", user, ok, err)
		}
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

func TestAVMUserIsTakenOnlyAsALowercaseShortNameNeverAdminNorRoot(t *testing.T) {
	for _, user := range []string{"you", "_build", "dev-2", "a23456789012345678901234567890ab", "jterrazz", "jterrazz.ops", "jterrazz.agent", "a.b-c_d"} {
		prefs := memoryPrefs{}
		if err := machinescontract.VMUser(prefs).Set(user); err != nil {
			t.Fatal(err)
		}
		if got, err := machinescontract.VMUser(prefs).Get(); err != nil || got != user {
			t.Errorf("vm_user %q read back %q, %v", user, got, err)
		}
	}
	for _, user := range []string{"Admin", "admin", "ROOT", "root", "you\nroot", "you\n", "2you", "you me", "-you", ".you", "you.", "Jterrazz.ops", "you:x", "you/x", "a23456789012345678901234567890abc"} {
		if err := machinescontract.CheckVMUser(user, false); err == nil || !strings.Contains(err.Error(), "vm_user") {
			t.Errorf("vm_user %q: %v; want it refused, naming vm_user", user, err)
		}
	}
}

func TestABadVMUserIsReadBackAsWrittenAndRefusedOnlyWhereAVMIsMade(t *testing.T) {
	prefs := memoryPrefs{}
	if err := machinescontract.VMUser(prefs).Set("Admin"); err != nil {
		t.Fatal(err)
	}
	if got, err := machinescontract.VMUser(prefs).Get(); err != nil || got != "Admin" {
		t.Errorf("vm_user read back %q, %v; want it as written, for a command that makes no VM", got, err)
	}
}

func TestInitLeavesVMUserUnsetForAnAccountTheRuleRefuses(t *testing.T) {
	for _, user := range []string{"Jterrazz", "jterrazz.", "admin", "root"} {
		prefs := memoryPrefs{}
		m := initing(t, prefs, user)
		note, err := m.Init("mac")
		if err != nil {
			t.Fatal(err)
		}
		if got, _ := machinescontract.VMUser(prefs).Get(); got != "" {
			t.Errorf("account %q: init wrote vm_user %q", user, got)
		}
		if !strings.Contains(note, "vm_user") || !strings.Contains(note, user) || strings.Contains(note, "\n") {
			t.Errorf("account %q: init said %q; want one line naming the account and vm_user", user, note)
		}
	}
	prefs := memoryPrefs{}
	note, err := initing(t, prefs, "jterrazz.ops").Init("mac")
	if got, _ := machinescontract.VMUser(prefs).Get(); err != nil || got != "jterrazz.ops" || note != "" {
		t.Errorf("a name the rule takes: vm_user %q, note %q, %v; want it written, nothing said", got, note, err)
	}
}

func TestTheLockdownScriptChecksTheVMUserByTheContractsRule(t *testing.T) {
	script, err := catalog.Recipes.ReadFile("images/macos-firstlogin.sh")
	if err != nil {
		t.Fatal(err)
	}
	if rule := "vm_user_rule='" + machinescontract.VMUserPattern + "'\n"; !strings.Contains(string(script), rule) {
		t.Errorf("the lockdown script lacks %q: its check of the VM user is not the contract's rule", rule)
	}
}

func TestThisMacsAccountIsCheckedByTheVMUsersRule(t *testing.T) {
	for user, ok := range map[string]bool{"jterrazz": true, "jterrazz.ops": true, "jterrazz.agent": true, "Jterrazz": false, "jterrazz.": false, "admin": false, "root": false, "": false} {
		err := machinescontract.CheckVMUser(user, true)
		if (err == nil) != ok {
			t.Errorf("this Mac's account %q: %v; want taken %v", user, err, ok)
			continue
		}
		var bad *model.BadVMUserError
		if !ok && (!errors.As(err, &bad) || !bad.Account || !strings.Contains(err.Error(), "this Mac's account")) {
			t.Errorf("this Mac's account %q: %v; want it named as the account, no vm_user set", user, err)
		}
	}
}

func initing(t *testing.T, prefs memoryPrefs, user string) *machines.Machinist {
	t.Helper()
	w := &world.World{ThisMac: "mac"}
	p := machines.Prefs{Name: setupcontract.Name(prefs), Macs: machinescontract.Macs(prefs), VMUser: machinescontract.VMUser(prefs)}
	return machines.New(&stillMac{at: time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)}, machinesfake.HypervisorOf(w, "mac", "hypervisor", nil), p, machinesfake.StoreOf(w, "mac"), noTailnet{}, &images{built: true}, &idleEngine{}, nil, machines.Options{
		ThisMachine: catalog.Mac, User: user, KnownHosts: "/state/known_hosts", Logs: "/state/logs",
	})
}
