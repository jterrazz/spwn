package local

import (
	"context"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

var sudoVars = Invoking{User: "SUDO_USER", UID: "SUDO_UID", GID: "SUDO_GID"}

func owner(name, home string) func(string) (*user.User, error) {
	return func(asked string) (*user.User, error) {
		if asked != name {
			return nil, errors.New("unknown user " + asked)
		}
		return &user.User{Username: name, Uid: "501", Gid: "20", HomeDir: home}, nil
	}
}

// invariant: sudo hands station HOME=/var/root and SUDO_USER, the owner who
// typed it: the run's home and name are the owner's, never root's, and only
// under euid 0 with SUDO_USER another account than root.
func TestARunUnderSudoIsTheOwnersWhoTypedIt(t *testing.T) {
	vars := map[string]string{"HOME": "/var/root", "USER": "root", "SUDO_USER": "jterrazz", "SUDO_UID": "501", "SUDO_GID": "20"}
	sudo := invokerOf(vars, sudoVars, 0, owner("jterrazz", "/Users/jterrazz"))
	env := identity{vars: vars, sudo: sudo}
	if env.Home() != "/Users/jterrazz" || env.User() != "jterrazz" {
		t.Fatalf("under sudo: home %q, user %q; want the owner's", env.Home(), env.User())
	}
	for name, v := range map[string]map[string]string{"no SUDO_USER": {"HOME": "/var/root"}, "sudo -u root": {"HOME": "/var/root", "SUDO_USER": "root"}} {
		if got := invokerOf(v, sudoVars, 0, owner("root", "/var/root")); got != nil {
			t.Errorf("%s: an invoker %+v; want none", name, got)
		}
	}
	if got := invokerOf(vars, sudoVars, 501, owner("jterrazz", "/Users/jterrazz")); got != nil {
		t.Errorf("not root: an invoker %+v; want none", got)
	}
	if env := (identity{vars: map[string]string{"HOME": "/var/root"}}); env.Home() != "/var/root" {
		t.Errorf("root by itself: home %q", env.Home())
	}
}

// invariant: what a run under sudo writes under the owner's home, the
// folders it makes, the files and the turn's lock, is given to the owner;
// a path outside that home stays root's.
func TestWhatARunUnderSudoWritesInTheOwnersHomeIsTheOwners(t *testing.T) {
	home := t.TempDir()
	var given []string
	o := &invoker{name: "jterrazz", home: home, uid: 501, gid: 20, chown: func(p string, uid, gid int) error {
		if uid != 501 || gid != 20 {
			t.Errorf("%s given to %d:%d", p, uid, gid)
		}
		given = append(given, p)
		return nil
	}}
	d := disk{owner: o}
	state := filepath.Join(home, ".local", "state", "station", "settings")
	if err := d.MkdirAll(state, 0o700); err != nil {
		t.Fatal(err)
	}
	rec := filepath.Join(state, "restart-after-power-cut.json")
	if err := d.Replace(rec, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(state, "runs.lock")
	release, held, err := (System{owner: o}).ExclusiveWithin(context.Background(), lock, time.Second)
	if err != nil || held {
		t.Fatalf("the turn's lock: held %v, %v", held, err)
	}
	release()
	outside := filepath.Join(t.TempDir(), "etc")
	if err := d.WriteFile(outside, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(home, ".local"), filepath.Join(home, ".local", "state"), filepath.Join(home, ".local", "state", "station"), state, rec, lock}
	if !slices.Equal(given, want) {
		t.Fatalf("given %q; want %q", given, want)
	}
	if _, err := os.Stat(rec); err != nil {
		t.Fatal(err)
	}
}
