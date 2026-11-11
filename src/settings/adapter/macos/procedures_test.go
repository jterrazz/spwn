package macos

import (
	"bytes"
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	backgroundcontract "github.com/jterrazz/studio-station/src/background/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/settings"
	"github.com/jterrazz/studio-station/src/settings/contract"
)

type memory struct{ files map[string][]byte }

func (m *memory) Stat(p string) (fs.FileInfo, error) {
	if _, ok := m.files[p]; ok {
		return fstest.MapFS{"f": {}}.Stat("f")
	}
	return nil, &fs.PathError{Op: "stat", Path: p, Err: fs.ErrNotExist}
}

func (m *memory) ReadFile(p string) ([]byte, error) {
	if b, ok := m.files[p]; ok {
		return b, nil
	}
	return nil, &fs.PathError{Op: "open", Path: p, Err: fs.ErrNotExist}
}

func (m *memory) WriteFile(p string, b []byte, _ fs.FileMode) error {
	m.files[p] = append([]byte(nil), b...)
	return nil
}

func (m *memory) MkdirAll(p string, _ fs.FileMode) error {
	m.files[p] = nil
	return nil
}

func (m *memory) Remove(p string) error              { delete(m.files, p); return nil }
func (*memory) Chmod(string, fs.FileMode) error      { return nil }
func (*memory) Chown(string, int, int) error         { return nil }
func (*memory) WalkDir(string, fs.WalkDirFunc) error { return nil }

type noJobs struct{ backgroundcontract.Jobs }

func (noJobs) Overrides(context.Context, backgroundcontract.Domain) map[string]bool { return nil }

func setting(t *testing.T, name string) catalog.Setting {
	t.Helper()
	c, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	it, ok := c.Setting(name)
	if !ok {
		t.Fatalf("no setting %s", name)
	}
	return it
}

func applierOf(t *testing.T, run local.Runner, files MacFiles, it catalog.Setting) contract.Applier {
	t.Helper()
	a, ok := contract.NewAppliers(NewMac(run, files, MacOptions{Home: "/home/u", Jobs: noJobs{}}).Appliers()...).For(it)
	if !ok {
		t.Fatalf("%s: no Applier", it.Name)
	}
	return a
}

func played(answers map[string]local.Output, names ...string) local.Runner {
	return local.Played(func(c local.Cmd) (local.Output, error) {
		return answers[strings.Join(append([]string{c.Name}, c.Args...), " ")], nil
	}, names...)
}

func TestRemoteLoginNeverReadsAsDoneWhenSystemsetupFails(t *testing.T) {
	run := played(map[string]local.Output{
		"systemsetup -setremotelogin on": {ExitCode: 1, Stderr: "setremotelogin: Turning Remote Login on or off requires Full Disk Access privileges.\n"},
	}, "systemsetup")
	sshd := setting(t, "sshd")

	_, err := applierOf(t, run, &memory{files: map[string][]byte{}}, sshd).Apply(context.Background(), sshd, nil)

	if err == nil || !strings.Contains(err.Error(), "Full Disk Access") {
		t.Fatalf("a failing systemsetup read as done: %v", err)
	}
}

func TestAutologinWritesThePasswordItIsGivenAsTheEncodedKcpassword(t *testing.T) {
	run := played(map[string]local.Output{
		"defaults read " + loginWindow + " autoLoginUser": {Stdout: "jterrazz.agent\n"},
	}, "defaults", "sysadminctl")
	files := &memory{files: map[string][]byte{agentHome: nil}}
	autologin := setting(t, "autologin")

	if _, err := applierOf(t, run, files, autologin).Apply(context.Background(), autologin, map[string]string{"password": "typed"}); err != nil {
		t.Fatal(err)
	}

	if got := files.files[kcpasswordTo]; !bytes.Equal(got, encodeKCPassword("typed")) {
		t.Fatalf("kcpassword %x", got)
	}
}

func TestAProcedureReadsItsValuesNowBesideTheOnesStationSets(t *testing.T) {
	run := played(map[string]local.Output{
		"pmset -g custom": {Stdout: "AC Power:\n autorestart 0\n sleep 0\n displaysleep 5\n disksleep 0\n powernap 0\n womp 1\n"},
	}, "pmset")
	power := setting(t, "restart-after-power-cut")

	o, err := applierOf(t, run, &memory{files: map[string][]byte{}}, power).Observe(context.Background(), power)

	if err != nil || o.State != catalog.StateDegraded {
		t.Fatalf("one value off the policy reads %s (%v), want partly there", o.State, err)
	}
	if len(o.Before) != len(o.After) || o.Before[0] != (settings.KeyValue{Key: "autorestart", Value: "0"}) || o.After[0] != (settings.KeyValue{Key: "autorestart", Value: "1"}) {
		t.Fatalf("before %v, after %v", o.Before, o.After)
	}
}
