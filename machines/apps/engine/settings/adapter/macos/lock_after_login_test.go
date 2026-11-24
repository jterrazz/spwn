package macos

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	backgroundcontract "github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type lockJobs struct {
	noJobs
	stopped *[]string
}

func (lockJobs) Names() backgroundcontract.Names { return backgroundcontract.Names{Command: "jobs"} }
func (lockJobs) Read(context.Context, backgroundcontract.Ref) backgroundcontract.Status {
	return backgroundcontract.Status{Loaded: true}
}
func (j lockJobs) Stop(_ context.Context, r backgroundcontract.Ref) error {
	*j.stopped = append(*j.stopped, r.File)
	return nil
}
func (lockJobs) Load(context.Context, backgroundcontract.Ref) error { return nil }

type homes struct{ *memory }

func (h homes) WalkDir(root string, fn fs.WalkDirFunc) error {
	seen := map[string]bool{}
	for p := range h.files {
		rest, ok := strings.CutPrefix(p, root+"/")
		if !ok {
			continue
		}
		name, _, _ := strings.Cut(rest, "/")
		seen[name] = true
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		info, _ := fstest.MapFS{name: {Mode: fs.ModeDir}}.Stat(name)
		entry := fs.FileInfoToDirEntry(info)
		if err := fn(path.Join(root, name), entry, nil); err != nil && !errors.Is(err, fs.SkipDir) {
			return err
		}
	}
	return nil
}

func lockPlistOf(user string) string {
	return "/Users/" + user + "/Library/LaunchAgents/" + lockLabel + ".plist"
}

func lockWorld(login string, files ...string) (*Mac, homes, *[]string) {
	answers := map[string]local.Output{
		"id -u a": {Stdout: "501\n"}, "id -g a": {Stdout: "20\n"},
		"id -u b": {Stdout: "502\n"}, "id -g b": {Stdout: "20\n"},
		"id -u kiosk": {Stdout: "503\n"}, "id -g kiosk": {Stdout: "20\n"},
	}
	if login != "" {
		answers["defaults read "+loginWindow+" autoLoginUser"] = local.Output{Stdout: login + "\n"}
	}
	mem := &memory{files: map[string][]byte{}}
	for _, f := range files {
		mem.files[f] = nil
	}
	stopped := &[]string{}
	f := homes{mem}
	return NewMac(played(answers, "defaults", "id", "stat"), f, MacOptions{Home: "/Users/kiosk", Jobs: lockJobs{stopped: stopped}}), f, stopped
}

func TestLockAfterLoginLocksTheUserTheMacLogsIn(t *testing.T) {
	m, files, _ := lockWorld("kiosk", "/Users/kiosk/.openclaw/scripts/lock-after-login.sh")

	if _, err := m.applyLock(context.Background()); err != nil {
		t.Fatal(err)
	}

	plist, ok := files.files[lockPlistOf("kiosk")]
	if !ok || !strings.Contains(string(plist), "/Users/kiosk/.openclaw/scripts/lock-after-login.sh") {
		t.Fatalf("the agent of the login user is not written: %v", files.files)
	}
}

func TestLockAfterLoginRefusesWhenNoUserLogsIn(t *testing.T) {
	m, _, _ := lockWorld("")

	_, err := m.applyLock(context.Background())

	if err == nil || !strings.Contains(err.Error(), "autologin") {
		t.Fatalf("no autologin user must be refused, naming autologin: %v", err)
	}
}

func TestLockAfterLoginRefusesAUserThatIsNoShortName(t *testing.T) {
	for _, user := range []string{"../etc", "a/b", "a&b", "a<b", "..", "a b", "-a"} {
		m, files, _ := lockWorld(user)

		_, err := m.applyLock(context.Background())

		if err == nil || !strings.Contains(err.Error(), "short name") {
			t.Errorf("autoLoginUser %q: %v, want a refusal naming the short name", user, err)
		}
		if len(files.files) != 0 {
			t.Errorf("autoLoginUser %q wrote %v", user, files.files)
		}
	}
}

func TestLockAfterLoginIsUndoneAfterAutologinIsGone(t *testing.T) {
	m, files, stopped := lockWorld("", lockPlistOf("kiosk"))

	if _, err := m.undoLock(context.Background()); err != nil {
		t.Fatal(err)
	}

	if _, left := files.files[lockPlistOf("kiosk")]; left {
		t.Fatalf("the agent stayed installed: %v", files.files)
	}
	if len(*stopped) != 1 || (*stopped)[0] != lockPlistOf("kiosk") {
		t.Fatalf("the loaded agent was not stopped: %v", *stopped)
	}
}

func TestLockAfterLoginReadsInstalledAfterAutologinIsGone(t *testing.T) {
	m, _, _ := lockWorld("", lockPlistOf("kiosk"))

	o := m.observeLock(context.Background(), nil)

	if o.State == catalog.StateAbsent {
		t.Fatalf("an agent still installed and loaded reads %s", o.State)
	}
}

func TestLockAfterLoginMovesToTheNewLoginUserLeavingNoCopyUnderTheOld(t *testing.T) {
	m, files, stopped := lockWorld("b", lockPlistOf("a"), "/Users/b/.openclaw/scripts/lock-after-login.sh")

	if _, err := m.applyLock(context.Background()); err != nil {
		t.Fatal(err)
	}

	if _, left := files.files[lockPlistOf("a")]; left {
		t.Fatalf("a copy stayed under the previous user: %v", files.files)
	}
	if _, ok := files.files[lockPlistOf("b")]; !ok {
		t.Fatalf("no agent under the new user: %v", files.files)
	}
	if len(*stopped) == 0 || (*stopped)[0] != lockPlistOf("a") {
		t.Fatalf("the previous user's loaded agent was not stopped first: %v", *stopped)
	}
}
