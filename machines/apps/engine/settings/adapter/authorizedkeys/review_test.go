package authorizedkeys_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func known(keys ...string) catalog.Setting {
	it := setting()
	it.Inputs = []catalog.Input{{Name: "user", Default: "root"}, {Name: "key", Default: strings.Join(keys, "\n")}}
	return it
}

func (m *machine) sshOf(home string, o local.Owner) {
	m.t.Helper()
	m.mkdir(home+"/.ssh", 0o700)
	m.chown(home, o)
	m.chown(home+"/.ssh", o)
}

func TestTheAccountListIsReadOrRefused(t *testing.T) {
	m := newMachine(t)
	if err := m.disk.Remove(m.at("etc/passwd")); err != nil {
		t.Fatal(err)
	}
	m.mkdir("etc/passwd", 0o755)
	if _, err := m.open().Undo(context.Background(), known(key(t, 1, ""))); err == nil || !strings.Contains(err.Error(), "/etc/passwd") {
		t.Fatalf("Undo with an unreadable /etc/passwd = %v; want a refusal naming it", err)
	}
}

func TestOffWithNoRecordReachesAnAccountThatDoesNotSignIn(t *testing.T) {
	m := newMachine(t)
	m.write("etc/passwd", "root:x:0:0:root:/root:/bin/bash\nbackup:x:34:34:backup:/home/ci:/usr/sbin/nologin\n", 0o644)
	m.sshOf("home/ci", m.ci)
	own := key(t, 9, "laptop") + "\n"
	k := key(t, 1, "x")
	m.write(ciAK, own+block(k), 0o600)
	m.chown(ciAK, m.ci)
	if _, err := m.open().Undo(context.Background(), known(k)); err != nil || m.content(ciAK) != own {
		t.Fatalf("Undo with no record = %v; backup's file %q; want the block out", err, m.content(ciAK))
	}
}
