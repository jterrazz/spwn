package authorizedkeys_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/authorizedkeys"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
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

func TestOffWithNoRecordReachesABlockInAnAccountThatIsNotTheDefault(t *testing.T) {
	m := newMachine(t)
	m.sshOf("home/ci", m.ci)
	own := key(t, 9, "laptop") + "\n"
	k := key(t, 1, "x")
	m.write(ciAK, own+block(k), 0o600)
	m.chown(ciAK, m.ci)
	it := known(k)
	seen, err := m.open().Observe(context.Background(), it)
	if err != nil || seen.State != catalog.StateOK || seen.Where != "/home/ci/.ssh/authorized_keys" {
		t.Fatalf("Observe with no record = %+v, %v; want ci's block seen", seen, err)
	}
	if _, err := m.open().Undo(context.Background(), it); err != nil || m.content(ciAK) != own {
		t.Fatalf("Undo with no record = %v; ci's file %q; want the block out", err, m.content(ciAK))
	}
}

func TestAUserHoldingTheirSSHFolderKeepsOnlyTheirOwnAccount(t *testing.T) {
	m := newMachine(t)
	k := key(t, 1, "x")
	own := key(t, 9, "laptop") + "\n"
	m.mkdir("root/.ssh", 0o700)
	m.write(rootAK, own+block(k), 0o600)
	m.sshOf("home/ci", m.ci)
	m.write(ciAK, own+block(k), 0o600)
	m.chown(ciAK, m.ci)
	release := m.hold("home/ci/.ssh")
	defer release()
	_, err := m.open().Undo(context.Background(), known(k))
	if err == nil || !errors.Is(err, contract.ErrKept) || !strings.Contains(err.Error(), "/home/ci/.ssh is held by another run") || !strings.Contains(err.Error(), "ci's keys") {
		t.Fatalf("Undo with ci's ~/.ssh held = %v; want needs-you naming ci", err)
	}
	if m.content(rootAK) != own || m.content(ciAK) != own+block(k) {
		t.Fatalf("root's %q, ci's %q; want root's block out and ci's kept", m.content(rootAK), m.content(ciAK))
	}
}

func TestWithNoRecordALineMachinesDoesNotKnowKeepsTheBlock(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o700)
	k, mine := key(t, 1, "x"), key(t, 7, "mine")
	text := key(t, 9, "laptop") + "\n" + block(k, mine)
	m.write(rootAK, text, 0o600)
	_, err := m.open().Undo(context.Background(), known(k))
	if err == nil || !errors.Is(err, contract.ErrKept) || !strings.Contains(err.Error(), "line 4 of /root/.ssh/authorized_keys") || m.content(rootAK) != text {
		t.Fatalf("Undo of a block holding a key Machines does not know = %v; file %q; want it kept, line 4 named", err, m.content(rootAK))
	}
	_, err = m.open().Undo(context.Background(), setting())
	if err == nil || !errors.Is(err, contract.ErrKept) || m.content(rootAK) != text {
		t.Fatalf("Undo with no key configured = %v; want the block kept", err)
	}
}

func TestOffOnAFileItCannotReadNeedsYou(t *testing.T) {
	m := newMachine(t)
	apply(t, m.open(), "root", key(t, 1, "ci"))
	if err := unix.Link(m.at(rootAK), m.at("root/other")); err != nil {
		t.Fatal(err)
	}
	if err := m.open().Kept(context.Background(), setting()); err == nil || !strings.Contains(err.Error(), "/root/.ssh/authorized_keys") {
		t.Fatalf("Kept on a hard-linked file = %v; want it named, needing you", err)
	}
	if _, err := m.open().Undo(context.Background(), setting()); !errors.Is(err, contract.ErrKept) {
		t.Fatalf("Undo on a hard-linked file = %v; want needs-you", err)
	}
}

func TestAWriteThatFailsLeavesMachinesOldBlockMachines(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o700)
	own := key(t, 9, "laptop") + "\n"
	m.write(rootAK, own, 0o600)
	apply(t, m.open(), "root", key(t, 1, "old"))
	extra := key(t, 8, "phone") + "\n"
	stale := m.content(rootAK) + extra
	_, err := m.openWith(racing{FS: m.disk, record: m.at(authorizedkeys.Record), file: m.at(rootAK), text: stale, done: new(bool)}).Apply(context.Background(), setting(), map[string]string{"user": "root", "key": key(t, 2, "new")})
	if err == nil || m.content(rootAK) != stale {
		t.Fatalf("Apply over a changed file = %v; want it refused", err)
	}
	if _, err := m.open().Undo(context.Background(), setting()); err != nil || m.content(rootAK) != own+extra {
		t.Fatalf("Undo after a failed write = %v; file %q; want Machines' old block out", err, m.content(rootAK))
	}
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

func TestOffRefusesAHomeUnderAFolderOthersCanWrite(t *testing.T) {
	m := newMachine(t)
	apply(t, m.open(), "ci", key(t, 1, ""))
	written := m.content(ciAK)
	m.mkdir("home", 0o777)
	t.Cleanup(func() { _ = m.disk.Chmod(m.at("home"), 0o755) })
	_, err := m.open().Undo(context.Background(), setting())
	if err == nil || !strings.Contains(err.Error(), "/home is writable by others") || m.content(ciAK) != written {
		t.Fatalf("Undo under a /home others can write = %v; file %q; want it refused, the block kept", err, m.content(ciAK))
	}
}

func TestAFileRootOwnsInAnotherUsersFolderReadsAsApplyRefusesIt(t *testing.T) {
	m := newMachine(t)
	if m.self.UID != 0 {
		t.Skip("only root owns what another user's ~/.ssh holds")
	}
	apply(t, m.open(), "ci", key(t, 1, ""))
	m.chown(ciAK, local.Owner{})
	o, err := m.open().Observe(context.Background(), setting())
	if err != nil || o.State != catalog.StateDegraded || !strings.Contains(o.Detail, "/home/ci/.ssh/authorized_keys is uid 0's, not the home's") {
		t.Fatalf("Observe of a file root owns in ci's ~/.ssh = %s %q, %v; want the drift apply refuses", o.State, o.Detail, err)
	}
}
