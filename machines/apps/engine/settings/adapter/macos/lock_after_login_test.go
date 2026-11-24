package macos

import (
	"context"
	"strings"
	"testing"

	backgroundcontract "github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type lockJobs struct{ noJobs }

func (lockJobs) Names() backgroundcontract.Names { return backgroundcontract.Names{Command: "jobs"} }
func (lockJobs) Read(context.Context, backgroundcontract.Ref) backgroundcontract.Status {
	return backgroundcontract.Status{}
}
func (lockJobs) Stop(context.Context, backgroundcontract.Ref) error { return nil }
func (lockJobs) Load(context.Context, backgroundcontract.Ref) error { return nil }

func lockApplier(run local.Runner, files MacFiles) func(context.Context) error {
	m := NewMac(run, files, MacOptions{Home: "/Users/kiosk", Jobs: lockJobs{}})
	return func(ctx context.Context) error {
		_, err := m.applyLock(ctx)
		return err
	}
}

func TestLockAfterLoginLocksTheUserTheMacLogsIn(t *testing.T) {
	run := played(map[string]local.Output{
		"defaults read " + loginWindow + " autoLoginUser": {Stdout: "kiosk\n"},
		"id -u kiosk": {Stdout: "501\n"},
		"id -g kiosk": {Stdout: "20\n"},
	}, "defaults", "id", "stat")
	files := &memory{files: map[string][]byte{"/Users/kiosk/.openclaw/scripts/lock-after-login.sh": nil}}
	apply := lockApplier(run, files)

	if err := apply(context.Background()); err != nil {
		t.Fatal(err)
	}

	plist, ok := files.files["/Users/kiosk/Library/LaunchAgents/"+lockLabel+".plist"]
	if !ok || !strings.Contains(string(plist), "/Users/kiosk/.openclaw/scripts/lock-after-login.sh") {
		t.Fatalf("the agent of the login user is not written: %v", files.files)
	}
	for p := range files.files {
		if strings.Contains(p, "jterrazz.agent") {
			t.Fatalf("a path names a person: %s", p)
		}
	}
}

func TestLockAfterLoginRefusesWhenNoUserLogsIn(t *testing.T) {
	run := played(map[string]local.Output{}, "defaults", "id", "stat")
	files := &memory{files: map[string][]byte{}}
	apply := lockApplier(run, files)

	err := apply(context.Background())

	if err == nil || !strings.Contains(err.Error(), "autologin") {
		t.Fatalf("no autologin user must be refused, naming autologin: %v", err)
	}
}
