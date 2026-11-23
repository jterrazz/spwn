package authorizedkeys_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/authorizedkeys"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
)

// value: a machine on the real disk, its system root a folder of the
// test's own, where ci is uid 1001: as root every owner is the system's;
// as any other user every account is the run's, as in a world.
type machine struct {
	t        *testing.T
	root     string
	disk     local.FS
	sys      local.System
	self, ci local.Owner
	run      *local.Owner
}

func newMachine(t *testing.T) *machine {
	t.Helper()
	sys := local.New(nil, nil, nil, nil)
	m := &machine{t: t, root: t.TempDir(), disk: sys.FS, sys: sys, self: local.Self()}
	m.ci = m.self
	if m.self.UID == 0 {
		m.ci = local.Owner{UID: 1001, GID: 1002}
	} else {
		m.run = &m.self
	}
	m.write("etc/passwd", "root:x:0:0:root:/root:/bin/bash\nci:x:1001:1002:CI:/home/ci:/bin/bash\n", 0o644)
	m.mkdir("root", 0o700)
	m.mkdir("home/ci", 0o755)
	m.chown("home/ci", m.ci)
	return m
}

func (m *machine) at(rel string) string { return filepath.Join(m.root, rel) }

func (m *machine) write(rel, data string, perm fs.FileMode) {
	m.t.Helper()
	if err := os.MkdirAll(filepath.Dir(m.at(rel)), 0o700); err != nil {
		m.t.Fatal(err)
	}
	if err := os.WriteFile(m.at(rel), []byte(data), perm); err != nil {
		m.t.Fatal(err)
	}
	if err := m.disk.Chmod(m.at(rel), perm); err != nil {
		m.t.Fatal(err)
	}
}

func (m *machine) mkdir(rel string, perm fs.FileMode) {
	m.t.Helper()
	if err := os.MkdirAll(m.at(rel), 0o700); err != nil {
		m.t.Fatal(err)
	}
	if err := m.disk.Chmod(m.at(rel), perm); err != nil {
		m.t.Fatal(err)
	}
}

func (m *machine) chown(rel string, o local.Owner) {
	m.t.Helper()
	if err := m.disk.Chown(m.at(rel), o.UID, o.GID); err != nil {
		m.t.Fatal(err)
	}
}

func (m *machine) link(target, rel string) {
	m.t.Helper()
	if err := m.disk.Symlink(target, m.at(rel)); err != nil {
		m.t.Fatal(err)
	}
}

func (m *machine) entry(rel string) local.Entry {
	m.t.Helper()
	e, err := local.Beneath{}.Lstat(m.root, rel)
	if err != nil {
		m.t.Fatal(err)
	}
	return e
}

func (m *machine) there(rel string) bool {
	_, err := local.Beneath{}.Lstat(m.root, rel)
	return err == nil
}

func (m *machine) content(rel string) string {
	data, err := os.ReadFile(m.at(rel))
	if err != nil {
		return ""
	}
	return string(data)
}

func (m *machine) open() *authorizedkeys.Applier { return m.openWith(m.disk) }

func (m *machine) openWith(files authorizedkeys.Files) *authorizedkeys.Applier {
	return authorizedkeys.New(files, brief{}, authorizedkeys.Options{
		Root: m.root, Record: m.at(authorizedkeys.Record), Run: m.run,
		Lock: m.at(authorizedkeys.Lock), Take: func(ctx context.Context, at string, _ time.Duration) (func(), bool, error) {
			return m.sys.ExclusiveWithin(ctx, at, 200*time.Millisecond)
		},
	})
}

// value: the system's folders, a held ~/.ssh waited for 200 ms.
type brief struct{ local.Beneath }

func (b brief) Hold(ctx context.Context, base, rel string, _ time.Duration) (*local.Folder, bool, error) {
	return b.Beneath.Hold(ctx, base, rel, 200*time.Millisecond)
}

// value: the disk, but a replace of the file at refused as a full disk.
type full struct {
	local.FS
	at string
}

func (f full) Replace(p string, data []byte, perm fs.FileMode) error {
	if p == f.at {
		return &fs.PathError{Op: "replace", Path: p, Err: errors.New("no space left on device")}
	}
	return f.FS.Replace(p, data, perm)
}

// value: the disk, but the first replace of the record also writes text
// in file, in place, as an editor that takes no lock would between
// station's read of file and its write.
type racing struct {
	local.FS
	record, file, text string
	done               *bool
}

func (r racing) Replace(p string, data []byte, perm fs.FileMode) error {
	if p == r.record && !*r.done {
		*r.done = true
		if err := os.WriteFile(r.file, []byte(r.text), 0o600); err != nil {
			return err
		}
	}
	return r.FS.Replace(p, data, perm)
}

const (
	rootAK = "root/.ssh/authorized_keys"
	ciAK   = "home/ci/.ssh/authorized_keys"
)

const ed25519Type = "\x00\x00\x00\x0bssh-ed25519"

func blob(size string, key []byte) []byte { return append([]byte(ed25519Type+size), key...) }

func key(t *testing.T, seed byte, comment string) string {
	t.Helper()
	pub := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, ed25519.SeedSize))[ed25519.SeedSize:]
	line := "ssh-ed25519 " + base64.StdEncoding.EncodeToString(blob("\x00\x00\x00\x20", pub))
	if comment != "" {
		line += " " + comment
	}
	return line
}

func setting() catalog.Setting {
	return catalog.Setting{Name: "authorized-key", Effect: catalog.AuthorizedKeys{Block: "station:authorized-key"}}
}

func apply(t *testing.T, a *authorizedkeys.Applier, user, keys string) settings.Change {
	t.Helper()
	c, err := a.Apply(context.Background(), setting(), map[string]string{"user": user, "key": keys})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return c
}

func refused(t *testing.T, a *authorizedkeys.Applier, user, keys string) error {
	t.Helper()
	_, err := a.Apply(context.Background(), setting(), map[string]string{"user": user, "key": keys})
	if err == nil {
		t.Fatal("Apply took it; want a refusal")
	}
	return err
}

func detail(c settings.Change) string {
	if len(c.Steps) != 1 {
		return ""
	}
	return c.Steps[0].Detail
}

func block(keys ...string) string {
	return "# >>> station:authorized-key >>>\n" + strings.Join(keys, "\n") + "\n# <<< station:authorized-key <<<\n"
}

func TestApplier(t *testing.T) {
	contracttest.Applier(t, func(t *testing.T) contracttest.Opened {
		m := newMachine(t)
		m.mkdir("root/.ssh", 0o700)
		m.write(rootAK, key(t, 9, "owner@laptop")+"\n", 0o600)
		edit := func(*testing.T) {
			m.write(rootAK, strings.Replace(m.content(rootAK), "ci@deploy", "mine", 1), 0o600)
		}
		return contracttest.Opened{
			Applier: m.open(), Package: setting(), Edit: edit,
			Values: map[string]string{"user": "root", "key": key(t, 1, "ci@deploy")},
		}
	})
}

func TestAnEmptyFileTakesTheBlockAlone(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o700)
	m.write(rootAK, "", 0o600)
	apply(t, m.open(), "root", key(t, 1, ""))
	if got, want := m.content(rootAK), block(key(t, 1, "")); got != want {
		t.Fatalf("authorized_keys = %q; want %q", got, want)
	}
}

func TestAFileThatIsNotThereIsMadeWithTheModesSshdAsks(t *testing.T) {
	m := newMachine(t)
	apply(t, m.open(), "ci", key(t, 1, ""))
	dir, f := m.entry("home/ci/.ssh"), m.entry(ciAK)
	if !dir.Mode.IsDir() || dir.Mode.Perm() != 0o700 || !f.Mode.IsRegular() || f.Mode.Perm() != 0o600 {
		t.Fatalf("folder %v, file %v; want 0700 and 0600", dir.Mode, f.Mode)
	}
	if dir.Owner != m.ci || f.Owner != m.ci {
		t.Fatalf("folder %+v, file %+v; want both the user's, %+v", dir.Owner, f.Owner, m.ci)
	}
	if got := m.content(ciAK); got != block(key(t, 1, "")) {
		t.Fatalf("authorized_keys = %q", got)
	}
}

func TestTheOwnersLinesKeepTheirBytesAndUndoGivesThemBack(t *testing.T) {
	for name, own := range map[string]string{
		"with a final newline":    "# my laptop\n" + key(t, 9, "owner@laptop") + "\n\n" + key(t, 8, "") + "\n",
		"without a final newline": key(t, 9, "owner@laptop") + "\n" + key(t, 8, "phone"),
		"with a blank last line":  key(t, 9, "owner@laptop") + "\n\n",
	} {
		t.Run(name, func(t *testing.T) {
			m := newMachine(t)
			m.mkdir("root/.ssh", 0o700)
			m.write(rootAK, own, 0o600)
			a := m.open()
			apply(t, a, "root", key(t, 1, "ci@deploy"))
			written := m.content(rootAK)
			if !strings.Contains(written, block(key(t, 1, "ci@deploy"))) {
				t.Fatalf("no block in %q", written)
			}
			for line := range strings.SplitSeq(strings.TrimRight(own, "\n"), "\n") {
				if line != "" && !strings.Contains(written, line+"\n") && !strings.HasSuffix(written, line) {
					t.Fatalf("the owner's line %q lost in %q", line, written)
				}
			}
			if _, err := a.Undo(context.Background(), setting()); err != nil {
				t.Fatalf("Undo: %v", err)
			}
			if got := m.content(rootAK); got != own {
				t.Fatalf("after Undo authorized_keys = %q; want the owner's bytes %q", got, own)
			}
		})
	}
}

func TestABlockAlreadyThereIsNotWrittenAgain(t *testing.T) {
	m := newMachine(t)
	a := m.open()
	apply(t, a, "root", key(t, 1, ""))
	before := m.content(rootAK)
	change := apply(t, a, "root", key(t, 1, ""))
	if got := m.content(rootAK); got != before || detail(change) != "already there" {
		t.Fatalf("authorized_keys = %q, step %+v; want %q, already there", got, change.Steps, before)
	}
}

// invariant: a folder or file others can write loses only that write;
// what others may read or open stays, as sshd takes it.
func TestLooseModesAreSetAsSshdAsks(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o775)
	m.write(rootAK, key(t, 9, "")+"\n", 0o666)
	apply(t, m.open(), "root", key(t, 1, ""))
	if dir, f := m.entry("root/.ssh").Mode.Perm(), m.entry(rootAK).Mode.Perm(); dir != 0o755 || f != 0o644 {
		t.Fatalf("folder %o, file %o; want 755 and 644", dir, f)
	}
}

// invariant: a folder and a file only their owner can write are what sshd
// takes, others reading them or not: apply keeps their modes and reads
// them applied.
func TestStrictModesAreKept(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o755)
	m.write(rootAK, key(t, 9, "")+"\n", 0o644)
	apply(t, m.open(), "root", key(t, 1, ""))
	if dir, f := m.entry("root/.ssh").Mode.Perm(), m.entry(rootAK).Mode.Perm(); dir != 0o755 || f != 0o644 {
		t.Fatalf("folder %o, file %o; want 755 and 644", dir, f)
	}
	if o, err := m.open().Observe(context.Background(), setting()); err != nil || o.State != catalog.StateOK {
		t.Fatalf("Observe = %+v, %v; want ok", o, err)
	}
}

// invariant: a folder or file root owns in another user's ~/.ssh that
// others can write reads changed since, with root's own fix, and apply
// refuses it, never chmodding it for that user.
func TestAWritableFolderOrFileRootOwnsForAnotherUserIsDriftNeverChmodded(t *testing.T) {
	m := newMachine(t)
	if m.self.UID != 0 {
		t.Skip("only root owns what another user's ~/.ssh holds")
	}
	apply(t, m.open(), "ci", key(t, 1, ""))
	m.chown("home/ci/.ssh", local.Owner{})
	m.chown(ciAK, local.Owner{})
	m.mkdir("home/ci/.ssh", 0o775)
	if err := m.disk.Chmod(m.at(ciAK), 0o664); err != nil {
		t.Fatal(err)
	}
	seen := func(want string) {
		t.Helper()
		o, err := m.open().Observe(context.Background(), setting())
		if err != nil || o.State != catalog.StateDegraded || o.Detail != want || !o.Loose {
			t.Fatalf("Observe = %s %q, %v; want degraded %q, loose", o.State, o.Detail, err, want)
		}
	}
	seen("/home/ci/.ssh is 0775, writable by others, so its keys are ignored: Machines changes no mode of root's for another user, take the write away with sudo chmod go-w /home/ci/.ssh")
	why := refused(t, m.open(), "ci", key(t, 1, "")).Error()
	if !strings.Contains(why, "/home/ci/.ssh is uid 0's, not the home's") {
		t.Fatalf("Apply in a ~/.ssh root owns = %q; want it refused", why)
	}
	if dir, f := m.entry("home/ci/.ssh"), m.entry(ciAK); dir.Mode.Perm() != 0o775 || f.Mode.Perm() != 0o664 || dir.UID != 0 || f.UID != 0 {
		t.Fatalf("folder %v uid %d, file %v uid %d; want root's 0775 and 0664 untouched", dir.Mode, dir.UID, f.Mode, f.UID)
	}
	if err := m.disk.Chmod(m.at("home/ci/.ssh"), 0o755); err != nil {
		t.Fatal(err)
	}
	seen("/home/ci/.ssh/authorized_keys is 0664, writable by others, so its keys are ignored: Machines changes no mode of root's for another user, take the write away with sudo chmod go-w /home/ci/.ssh/authorized_keys")
}

// invariant: an authorized_keys with another name, a hard link, is
// refused: neither read, written nor chmodded, so whatever it shares its
// bytes with stays as it was.
func TestAHardLinkedAuthorizedKeysIsRefusedAndItsTargetUntouched(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o700)
	m.write("etc/shadow", "root:secret\n", 0o666)
	if err := unix.Link(m.at("etc/shadow"), m.at(rootAK)); err != nil {
		t.Fatal(err)
	}
	why := refused(t, m.open(), "root", key(t, 1, "")).Error()
	if !strings.Contains(why, "/root/.ssh/authorized_keys has another name, a hard link, not touched") {
		t.Fatalf("Apply on a hard link = %q; want it refused as a hard link", why)
	}
	_, _ = m.open().Undo(context.Background(), setting())
	if got, mode := m.content("etc/shadow"), m.entry("etc/shadow").Mode.Perm(); got != "root:secret\n" || mode != 0o666 {
		t.Fatalf("the file it links = %q at %o; want it untouched at 666", got, mode)
	}
}

// invariant: a user who links ~/.ssh to a folder of the system's gets it
// refused: nothing there is made, chmodded or given to the user.
func TestALinkedSshFolderIsRefusedAndItsTargetUntouched(t *testing.T) {
	m := newMachine(t)
	m.mkdir("etc/sudoers.d", 0o755)
	m.write("etc/sudoers.d/ci", "ci ALL=(ALL) ALL\n", 0o644)
	m.link(m.at("etc/sudoers.d"), "home/ci/.ssh")
	before := m.entry("etc/sudoers.d")
	err := refused(t, m.open(), "ci", key(t, 1, ""))
	if !strings.Contains(err.Error(), "/home/ci/.ssh is a link, not touched") {
		t.Fatalf("Apply = %v; want the link named", err)
	}
	if after := m.entry("etc/sudoers.d"); after != before || m.there("etc/sudoers.d/authorized_keys") {
		t.Fatalf("the link's target moved: %+v, then %+v, authorized_keys there %v", before, after, m.there("etc/sudoers.d/authorized_keys"))
	}
}

// invariant: an authorized_keys that is a link is refused at apply and at
// off: what it links to is neither read nor copied, and the link stays.
func TestALinkedAuthorizedKeysIsNeitherReadNorReplaced(t *testing.T) {
	shadow := "root:$6$hash:19000:0:99999:7:::\n"
	for name, swap := range map[string]func(t *testing.T, m *machine, a *authorizedkeys.Applier) error{
		"at apply": func(t *testing.T, m *machine, a *authorizedkeys.Applier) error {
			m.mkdir("home/ci/.ssh", 0o700)
			m.chown("home/ci/.ssh", m.ci)
			m.link(m.at("etc/shadow"), ciAK)
			_, err := a.Apply(context.Background(), setting(), map[string]string{"user": "ci", "key": key(t, 1, "")})
			return err
		},
		"at off, swapped in after apply": func(t *testing.T, m *machine, a *authorizedkeys.Applier) error {
			apply(t, a, "ci", key(t, 1, ""))
			_ = m.disk.Remove(m.at(ciAK))
			m.link(m.at("etc/shadow"), ciAK)
			_, err := a.Undo(context.Background(), setting())
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := newMachine(t)
			m.write("etc/shadow", shadow, 0o600)
			before := m.entry("etc/shadow")
			err := swap(t, m, m.open())
			if err == nil || !strings.Contains(err.Error(), "/home/ci/.ssh/authorized_keys is a link, not touched") {
				t.Fatalf("got %v; want the link refused by name", err)
			}
			if m.entry(ciAK).Mode&fs.ModeSymlink == 0 || m.content("etc/shadow") != shadow || m.entry("etc/shadow") != before {
				t.Fatalf("authorized_keys %v, target %q %+v; want the link kept and its target as it was", m.entry(ciAK).Mode, m.content("etc/shadow"), m.entry("etc/shadow"))
			}
		})
	}
}

// invariant: a block whose end marker is gone, or that is there twice, is
// refused before any write, so a second apply never takes the owner's keys
// after a lone start marker for station's.
func TestADamagedBlockIsRefusedAndTheOwnersKeysKept(t *testing.T) {
	for name, own := range map[string]string{
		"a start with no end": key(t, 9, "laptop") + "\n# >>> station:authorized-key >>>\n" + key(t, 8, "phone") + "\n" + key(t, 7, "tablet") + "\n",
		"two blocks":          block(key(t, 1, "")) + key(t, 9, "laptop") + "\n" + block(key(t, 2, "")),
	} {
		t.Run(name, func(t *testing.T) {
			m := newMachine(t)
			m.mkdir("root/.ssh", 0o700)
			m.write(rootAK, own, 0o600)
			a := m.open()
			for range 2 {
				err := refused(t, a, "root", key(t, 1, "ci"))
				if !strings.Contains(err.Error(), "/root/.ssh/authorized_keys holds a damaged station:authorized-key block") || !strings.Contains(err.Error(), "then apply again") {
					t.Fatalf("Apply = %v; want the damage named with its repair", err)
				}
			}
			if got := m.content(rootAK); got != own {
				t.Fatalf("authorized_keys = %q; want the owner's bytes %q", got, own)
			}
		})
	}
}

// invariant: applying for another user takes station's block out of the
// first user's file, so the key it granted there stops signing in; off
// takes it out of every file station wrote.
func TestAnotherUserTakesTheBlockOutOfTheFirstUsersFile(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o700)
	own := key(t, 9, "laptop") + "\n"
	m.write(rootAK, own, 0o600)
	a := m.open()
	apply(t, a, "root", key(t, 1, "ci"))
	change := apply(t, a, "ci", key(t, 1, "ci"))
	if got := m.content(rootAK); got != own {
		t.Fatalf("root's authorized_keys = %q; want the owner's line alone, %q", got, own)
	}
	if len(change.Steps) != 2 || change.Steps[1].Label != "/root/.ssh/authorized_keys" || change.Steps[1].Detail != "removed" {
		t.Fatalf("steps %+v; want root's block named removed", change.Steps)
	}
	if got := m.content(ciAK); got != block(key(t, 1, "ci")) {
		t.Fatalf("ci's authorized_keys = %q", got)
	}
	if _, err := a.Undo(context.Background(), setting()); err != nil {
		t.Fatal(err)
	}
	if m.content(ciAK) != "" || m.content(rootAK) != own || m.there(strings.TrimPrefix(authorizedkeys.Record, "/")) {
		t.Fatalf("after off ci %q, root %q, record there %v; want both blocks gone and no record", m.content(ciAK), m.content(rootAK), m.there(strings.TrimPrefix(authorizedkeys.Record, "/")))
	}
}

func TestABlockTheOwnerChangedIsKeptWhenAnotherUserTakesIt(t *testing.T) {
	m := newMachine(t)
	a := m.open()
	apply(t, a, "root", key(t, 1, "ci"))
	edited := strings.Replace(m.content(rootAK), " ci\n", " mine\n", 1)
	m.write(rootAK, edited, 0o600)
	change := apply(t, a, "ci", key(t, 1, "ci"))
	if len(change.Steps) != 2 || change.Steps[1].Detail != "yours, kept" || m.content(rootAK) != edited {
		t.Fatalf("steps %+v, root %q; want root's block kept as the owner left it", change.Steps, m.content(rootAK))
	}
	off, err := a.Undo(context.Background(), setting())
	if err == nil || !strings.Contains(err.Error(), "still signs in") || len(off.Steps) != 3 || off.Steps[0].Detail != "yours, kept" || off.Steps[1].Detail != "removed, and the empty file Machines made" || off.Steps[2].Label != "/home/ci/.ssh" || m.content(rootAK) != edited {
		t.Fatalf("Undo = %+v, %v; want root's kept, ci's file and folder removed", off.Steps, err)
	}
}

// invariant: an apply cut before its record lands leaves no block of
// station's that off cannot find: the record is written before the block.
func TestAnApplyWhoseRecordFailsLeavesNoBlockOffCannotFind(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o700)
	own := key(t, 9, "laptop") + "\n"
	m.write(rootAK, own, 0o600)
	if _, err := m.openWith(full{FS: m.disk, at: m.at(authorizedkeys.Record)}).Apply(context.Background(), setting(), map[string]string{"user": "root", "key": key(t, 1, "")}); err == nil {
		t.Fatal("Apply took a record it could not write")
	}
	if _, err := m.open().Undo(context.Background(), setting()); err != nil {
		t.Fatal(err)
	}
	if got := m.content(rootAK); got != own {
		t.Fatalf("after off authorized_keys = %q; want the owner's line alone, no block of station's", got)
	}
}

// invariant: an apply that made ~/.ssh and fails before its record names
// it takes the folder away again: no folder is left that off does not know
// it made.
func TestAnApplyWhoseRecordFailsTakesAwayTheFolderItMade(t *testing.T) {
	m := newMachine(t)
	if _, err := m.openWith(full{FS: m.disk, at: m.at(authorizedkeys.Record)}).Apply(context.Background(), setting(), map[string]string{"user": "root", "key": key(t, 1, "")}); err == nil {
		t.Fatal("Apply took a record it could not write")
	}
	if m.there("root/.ssh") {
		t.Fatal("Apply left the ~/.ssh it made with no record of it")
	}
}

// invariant: a file that changed between station's read and its write,
// by a writer that takes no lock, is refused with nothing touched: the
// write is judged on the bytes it replaces.
func TestAFileChangedAfterItsReadIsRefusedAndTheChangeKept(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o700)
	m.write(rootAK, key(t, 9, "laptop")+"\n", 0o600)
	edited := key(t, 9, "laptop") + "\n" + key(t, 8, "phone") + "\n"
	_, err := m.openWith(racing{FS: m.disk, record: m.at(authorizedkeys.Record), file: m.at(rootAK), text: edited, done: new(bool)}).Apply(context.Background(), setting(), map[string]string{"user": "root", "key": key(t, 1, "")})
	if err == nil || !strings.Contains(err.Error(), "/root/.ssh/authorized_keys changed since Machines read it, not touched") || m.content(rootAK) != edited {
		t.Fatalf("Apply over a file changed since its read = %v; file %q; want it refused and the change kept", err, m.content(rootAK))
	}
}

// invariant: with no record at all, off finds station's block by its
// markers in the default user's file and takes it out.
func TestOffFindsABlockWithNoRecordInTheDefaultUsersFile(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o700)
	own := key(t, 9, "laptop") + "\n"
	m.write(rootAK, own+block(key(t, 1, "ci")), 0o600)
	it := setting()
	it.Inputs = []catalog.Input{{Name: "user", Default: "root"}, {Name: "key"}}
	seen, err := m.open().Observe(context.Background(), it)
	if err != nil || seen.State != catalog.StateOK || seen.Where != "/root/.ssh/authorized_keys" {
		t.Fatalf("Observe = %+v, %v; want the block seen in the default user's file", seen, err)
	}
	change, err := m.open().Undo(context.Background(), it)
	if err != nil || detail(change) != "removed, found with no record" || m.content(rootAK) != own {
		t.Fatalf("Undo = %+v, %v; file %q; want the block found and removed", change, err, m.content(rootAK))
	}
}

// invariant: a home that is a link, not the user's, or not there is
// refused: Machines writes in a real home and never makes one.
func TestAHomeThatIsNotTheUsersRealFolderIsRefused(t *testing.T) {
	for name, c := range map[string]struct {
		make func(m *machine)
		says string
	}{
		"not there": {func(m *machine) { _ = m.disk.Remove(m.at("home/ci")) }, "ci has no home on this machine (/home/ci is not there): Machines never makes one"},
		"a link": {func(m *machine) {
			_ = m.disk.Remove(m.at("home/ci"))
			m.mkdir("srv", 0o755)
			m.link(m.at("srv"), "home/ci")
		}, "/home/ci is a link, not touched"},
		"a file": {func(m *machine) { _ = m.disk.Remove(m.at("home/ci")); m.write("home/ci", "", 0o644) }, "/home/ci is not a folder, not touched"},
		"the system's root": {func(m *machine) {
			m.write("etc/passwd", "ci:x:1001:1002:CI:/:/bin/bash\n", 0o644)
		}, "ci has no home of its own (/): Machines writes only in a user's home"},
	} {
		t.Run(name, func(t *testing.T) {
			m := newMachine(t)
			c.make(m)
			if err := refused(t, m.open(), "ci", key(t, 1, "")); !strings.Contains(err.Error(), c.says) {
				t.Fatalf("Apply = %v; want %q", err, c.says)
			}
			if m.there("srv/.ssh") || m.there("home/ci/.ssh") {
				t.Fatal("a .ssh was made")
			}
		})
	}
}

func TestAHomeAnotherUserOwnsIsRefused(t *testing.T) {
	m := newMachine(t)
	if m.self.UID != 0 {
		t.Skip("an owner other than the run's is given by root alone")
	}
	m.chown("home/ci", local.Owner{UID: 1500, GID: 1500})
	if err := refused(t, m.open(), "ci", key(t, 1, "")); !strings.Contains(err.Error(), "/home/ci is not ci's (its owner is uid 1500), not touched") {
		t.Fatalf("Apply = %v", err)
	}
}

func TestKeys(t *testing.T) {
	one, two := key(t, 1, "ci@deploy runner 1"), key(t, 2, "")
	for _, c := range []struct{ name, given, want string }{
		{"a key with a comment keeps it whole", one, one + "\n"},
		{"two keys, one per line, blanks and spaces dropped", "  " + one + "  \n\n" + two + "\n", one + "\n" + two + "\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := authorizedkeys.Keys(c.given)
			if err != nil || got != c.want {
				t.Fatalf("Keys = %q, %v; want %q", got, err, c.want)
			}
		})
	}
	for name, given := range map[string]string{
		"nothing":                       " \n",
		"not base64":                    "ssh-ed25519 not-a-key me",
		"an unknown type":               "ssh-foo AAAAC3NzaC1lZDI1NTE5AAAAIA== me",
		"a private key line":            "-----BEGIN OPENSSH PRIVATE KEY-----",
		"options before it":             `from="10.0.0.1" ` + one,
		"one good, one bad":             one + "\nssh-ed25519 AAAA",
		"a type its blob does not name": "ssh-rsa " + strings.Fields(two)[1],
		"a short ed25519 key":           "ssh-ed25519 " + base64.StdEncoding.EncodeToString(blob("\x00\x00\x00\x1f", make([]byte, 31))),
		"bytes past the key":            "ssh-ed25519 " + base64.StdEncoding.EncodeToString(append(blob("\x00\x00\x00\x20", make([]byte, 32)), 0)),
	} {
		t.Run("refused: "+name, func(t *testing.T) {
			if got, err := authorizedkeys.Keys(given); err == nil {
				t.Fatalf("Keys(%q) = %q; want a refusal", given, got)
			}
		})
	}
}

func TestABadKeyIsRefusedBeforeAnyWrite(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o755)
	m.write(rootAK, "mine\n", 0o644)
	_ = refused(t, m.open(), "root", "ssh-ed25519 AAAA")
	if m.content(rootAK) != "mine\n" || m.entry(rootAK).Mode.Perm() != 0o644 || m.entry("root/.ssh").Mode.Perm() != 0o755 {
		t.Fatalf("file %q mode %o, folder %o; want nothing touched", m.content(rootAK), m.entry(rootAK).Mode.Perm(), m.entry("root/.ssh").Mode.Perm())
	}
}

func TestAnUnknownUserIsRefused(t *testing.T) {
	m := newMachine(t)
	if err := refused(t, m.open(), "nobody", key(t, 1, "")); err.Error() != "no user nobody on this machine" {
		t.Fatalf("Apply = %v", err)
	}
}

func TestAHandEditInsideTheBlockReadsDriftAndSurvivesUndo(t *testing.T) {
	m := newMachine(t)
	a := m.open()
	apply(t, a, "root", key(t, 1, "ci"))
	m.write(rootAK, strings.Replace(m.content(rootAK), " ci\n", " mine\n", 1), 0o600)
	o, err := a.Observe(context.Background(), setting())
	if err != nil || o.State != catalog.StateDegraded || o.Diff == "" || o.Loose {
		t.Fatalf("Observe = %+v, %v; want degraded with a diff, its content the owner's", o, err)
	}
	edited := m.content(rootAK)
	change, err := a.Undo(context.Background(), setting())
	if err == nil || m.content(rootAK) != edited || detail(change) != "yours, kept" {
		t.Fatalf("Undo = %+v, %v; file %q; want it kept", change, err, m.content(rootAK))
	}
}

func TestAWriterHoldingTheLockKeepsApplyAndOffOutOfTheFile(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o700)
	own := key(t, 9, "laptop") + "\n"
	m.write(rootAK, own, 0o600)
	release, held, err := m.sys.ExclusiveWithin(context.Background(), m.at(authorizedkeys.Lock), time.Second)
	if err != nil || held {
		t.Fatalf("ExclusiveWithin = %v, %v; want the lock taken", held, err)
	}
	why := refused(t, m.open(), "root", key(t, 1, "ci")).Error()
	if !strings.Contains(why, "/var/lib/station/authorized-keys.lock is held by another run") || m.content(rootAK) != own {
		t.Fatalf("Apply with the lock held = %q; file %q; want it refused and the file as it was", why, m.content(rootAK))
	}
	release()
	apply(t, m.open(), "root", key(t, 1, "ci"))
	release, _, err = m.sys.ExclusiveWithin(context.Background(), m.at(authorizedkeys.Lock), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	written := m.content(rootAK)
	_, err = m.open().Undo(context.Background(), setting())
	if err == nil || !strings.Contains(err.Error(), "is held by another run") || m.content(rootAK) != written {
		t.Fatalf("Undo with the lock held = %v; file %q; want it refused and the block kept", err, m.content(rootAK))
	}
}

func TestAFolderOrFileTheSignInServerIgnoresIsDrift(t *testing.T) {
	m := newMachine(t)
	apply(t, m.open(), "root", key(t, 1, "ci"))
	seen := func(t *testing.T) settings.Observation {
		t.Helper()
		o, err := m.open().Observe(context.Background(), setting())
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	if o := seen(t); o.State != catalog.StateOK {
		t.Fatalf("Observe after apply = %+v; want ok", o)
	}
	loosen := func(t *testing.T, rel string, mode, strict fs.FileMode, want string) {
		t.Helper()
		if err := m.disk.Chmod(m.at(rel), mode); err != nil {
			t.Fatal(err)
		}
		o := seen(t)
		if o.State != catalog.StateDegraded || o.Detail != want || !o.Loose {
			t.Fatalf("Observe with %s at %04o = %s %q, loose %v; want degraded %q, loose", rel, mode, o.State, o.Detail, o.Loose, want)
		}
		if err := m.disk.Chmod(m.at(rel), strict); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("a folder others can write", func(t *testing.T) {
		loosen(t, "root/.ssh", 0o775, 0o700, "/root/.ssh is 0775, writable by others, so its keys are ignored: station settings apply authorized-key sets it to 0755")
	})
	t.Run("a file others can write", func(t *testing.T) {
		loosen(t, rootAK, 0o664, 0o600, "/root/.ssh/authorized_keys is 0664, writable by others, so its keys are ignored: station settings apply authorized-key sets it to 0644")
	})
	t.Run("a file another user owns", func(t *testing.T) {
		if m.self.UID != 0 {
			t.Skip("only root gives a file away")
		}
		m.chown(rootAK, m.ci)
		o := seen(t)
		want := "/root/.ssh/authorized_keys is uid " + strconv.Itoa(m.ci.UID) + "'s, not the home's (uid 0), so its keys are ignored: Machines never takes a file from its owner, give it back with sudo chown 0 /root/.ssh/authorized_keys"
		if o.State != catalog.StateDegraded || o.Detail != want || !o.Loose {
			t.Fatalf("Observe with the file given away = %s %q; want degraded %q", o.State, o.Detail, want)
		}
	})
}

// invariant: a run without admin rights cannot read the machine's record;
// it reads the setting blind, as an admin setting it cannot see, never as
// an error or as a block that is not there.
func TestARecordOnlyRootReadsIsReadBlindWithoutAdmin(t *testing.T) {
	m := newMachine(t)
	if m.self.UID == 0 {
		t.Skip("root reads every record")
	}
	apply(t, m.open(), "root", key(t, 1, "ci"))
	if err := m.disk.Chmod(m.at(authorizedkeys.Record), 0); err != nil {
		t.Fatal(err)
	}
	o, err := m.open().Observe(context.Background(), setting())
	if err != nil || !o.Blind || o.State == catalog.StateAbsent || o.Detail != "needs admin to read /var/lib/station/authorized-keys.json" {
		t.Fatalf("Observe = %+v, %v; want it read blind, needing admin", o, err)
	}
}

// invariant: an account whose shell is nologin or false never signs in, and
// gets no key: a key there would still open its tunnels; root always does.
func TestAnAccountThatDoesNotSignInIsRefused(t *testing.T) {
	for _, shell := range []string{"/usr/sbin/nologin", "/sbin/nologin", "/bin/false", "/usr/bin/false"} {
		t.Run(shell, func(t *testing.T) {
			m := newMachine(t)
			m.write("etc/passwd", "root:x:0:0:root:/root:/bin/bash\nci:x:1001:1002:CI:/home/ci:"+shell+"\n", 0o644)
			err := refused(t, m.open(), "ci", key(t, 1, ""))
			if want := "ci does not sign in (its shell is " + shell + "), not touched"; !strings.Contains(err.Error(), want) {
				t.Fatalf("Apply = %v; want %q", err, want)
			}
			if m.there("home/ci/.ssh") {
				t.Fatal("a .ssh was made")
			}
		})
	}
	t.Run("root with nologin", func(t *testing.T) {
		m := newMachine(t)
		m.write("etc/passwd", "root:x:0:0:root:/root:/usr/sbin/nologin\n", 0o644)
		if _, err := m.open().Apply(context.Background(), setting(), map[string]string{"user": "root", "key": key(t, 1, "")}); err != nil {
			t.Fatalf("Apply for root = %v; want it taken", err)
		}
	})
}

// invariant: off leaves a home as station found it: the authorized_keys
// and the ~/.ssh station made are taken away once they hold nothing else,
// and what was there before, or what the owner added since, stays.
func TestOffTakesAwayTheFileAndFolderItMadeOnceEmpty(t *testing.T) {
	off := func(t *testing.T, m *machine) {
		t.Helper()
		if _, err := m.open().Undo(context.Background(), setting()); err != nil {
			t.Fatalf("Undo: %v", err)
		}
	}
	t.Run("both made, applied twice", func(t *testing.T) {
		m := newMachine(t)
		apply(t, m.open(), "ci", key(t, 1, ""))
		apply(t, m.open(), "ci", key(t, 2, ""))
		off(t, m)
		if m.there("home/ci/.ssh") {
			t.Fatalf("home/ci/.ssh is still there, %q in it", m.content(ciAK))
		}
	})
	t.Run("the folder holds another file", func(t *testing.T) {
		m := newMachine(t)
		apply(t, m.open(), "ci", key(t, 1, ""))
		m.write("home/ci/.ssh/known_hosts", "host\n", 0o600)
		off(t, m)
		if m.there(ciAK) || m.content("home/ci/.ssh/known_hosts") != "host\n" {
			t.Fatalf("authorized_keys there %v, known_hosts %q; want the file gone, the folder kept", m.there(ciAK), m.content("home/ci/.ssh/known_hosts"))
		}
	})
	t.Run("the owner added a line", func(t *testing.T) {
		m := newMachine(t)
		apply(t, m.open(), "ci", key(t, 1, ""))
		m.write(ciAK, m.content(ciAK)+key(t, 9, "laptop")+"\n", 0o600)
		off(t, m)
		if got := m.content(ciAK); got != key(t, 9, "laptop")+"\n" {
			t.Fatalf("authorized_keys = %q; want the owner's line kept", got)
		}
	})
	t.Run("both there before", func(t *testing.T) {
		m := newMachine(t)
		m.mkdir("home/ci/.ssh", 0o700)
		m.chown("home/ci/.ssh", m.ci)
		m.write(ciAK, "", 0o600)
		m.chown(ciAK, m.ci)
		apply(t, m.open(), "ci", key(t, 1, ""))
		off(t, m)
		if !m.there(ciAK) || m.content(ciAK) != "" {
			t.Fatalf("authorized_keys there %v, %q; want the empty file kept", m.there(ciAK), m.content(ciAK))
		}
	})
}

// invariant: a strict folder or file keeps the mode its owner gave it:
// apply only takes away what others can write.
func TestAStrictModeIsKeptAsItIs(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o700)
	m.write(rootAK, key(t, 9, "")+"\n", 0o400)
	if m.self.UID == 0 {
		m.mkdir("root/.ssh", 0o500)
	}
	apply(t, m.open(), "root", key(t, 1, ""))
	dir, f := m.entry("root/.ssh").Mode.Perm(), m.entry(rootAK).Mode.Perm()
	if f != 0o400 || (m.self.UID == 0 && dir != 0o500) {
		t.Fatalf("folder %o, file %o; want the strict modes kept", dir, f)
	}
}

// invariant: station writes keys only in a ~/.ssh and a file the home's
// owner owns: one another user or root owns is refused with nothing
// touched, its owner kept, and one given away after an apply reads drift
// with the owner's own fix.
func TestAFolderOrFileNotTheUsersIsRefusedAndKeepsItsOwner(t *testing.T) {
	m := newMachine(t)
	if m.self.UID != 0 {
		t.Skip("only root gives a file away")
	}
	m.mkdir("home/ci/.ssh", 0o700)
	own := key(t, 9, "") + "\n"
	m.write(ciAK, own, 0o600)
	other := local.Owner{UID: 1500, GID: 1500}
	m.chown(ciAK, m.ci)
	ci := strconv.Itoa(m.ci.UID)
	why := refused(t, m.open(), "ci", key(t, 1, "")).Error()
	if !strings.Contains(why, "/home/ci/.ssh is uid 0's, not the home's (uid "+ci+"), not touched") || m.entry("home/ci/.ssh").Owner != (local.Owner{}) || m.content(ciAK) != own {
		t.Fatalf("Apply in a ~/.ssh root owns = %q; want it refused, nothing touched", why)
	}
	m.chown("home/ci/.ssh", m.ci)
	m.chown(ciAK, other)
	why = refused(t, m.open(), "ci", key(t, 1, "")).Error()
	if !strings.Contains(why, "/home/ci/.ssh/authorized_keys is uid 1500's, not the home's (uid "+ci+"), not touched") || m.entry(ciAK).Owner != other || m.content(ciAK) != own {
		t.Fatalf("Apply on a file uid 1500 owns = %q; want it refused, nothing touched", why)
	}
	m.chown(ciAK, m.ci)
	apply(t, m.open(), "ci", key(t, 1, ""))
	m.chown(ciAK, other)
	o, err := m.open().Observe(context.Background(), setting())
	want := "/home/ci/.ssh/authorized_keys is uid 1500's, not the home's (uid " + ci + "), so its keys are ignored: Machines never takes a file from its owner, give it back with sudo chown " + ci + " /home/ci/.ssh/authorized_keys"
	if err != nil || o.State != catalog.StateDegraded || o.Detail != want {
		t.Fatalf("Observe = %s %q, %v; want degraded %q", o.State, o.Detail, err, want)
	}
}

func (m *machine) hold(rel string) func() {
	m.t.Helper()
	fd, err := unix.Open(m.at(rel), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		m.t.Fatal(err)
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		m.t.Fatal(err)
	}
	return func() { _ = unix.Close(fd) }
}

func TestAHolderOfTheSSHFolderKeepsApplyAndOffOutOfTheFile(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o700)
	own := key(t, 9, "laptop") + "\n"
	m.write(rootAK, own, 0o600)
	release := m.hold("root/.ssh")
	why := refused(t, m.open(), "root", key(t, 1, "ci")).Error()
	if !strings.Contains(why, "/root/.ssh is held by another run") || m.content(rootAK) != own {
		t.Fatalf("Apply with ~/.ssh held = %q; file %q; want it refused and the file as it was", why, m.content(rootAK))
	}
	release()
	apply(t, m.open(), "root", key(t, 1, "ci"))
	release = m.hold("root/.ssh")
	defer release()
	written := m.content(rootAK)
	_, err := m.open().Undo(context.Background(), setting())
	if err == nil || !strings.Contains(err.Error(), "/root/.ssh is held by another run") || m.content(rootAK) != written {
		t.Fatalf("Undo with ~/.ssh held = %v; file %q; want it refused and the block kept", err, m.content(rootAK))
	}
}

func TestAStrayStartAboveTheBlockIsRefusedWithItsLine(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o700)
	text := key(t, 8, "a") + "\n# >>> station:authorized-key >>>\n" + key(t, 9, "b") + "\n" + block(key(t, 1, "old"))
	m.write(rootAK, text, 0o600)
	why := refused(t, m.open(), "root", key(t, 2, "new")).Error()
	if m.content(rootAK) != text || !strings.Contains(why, "delete line 2 of /root/.ssh/authorized_keys") || strings.Contains(why, "around Machines' keys") {
		t.Fatalf("Apply over a stray start = %q; file %q; want it refused, line 2 named and the file as it was", why, m.content(rootAK))
	}
}

func TestASystemAccountIsRefused(t *testing.T) {
	m := newMachine(t)
	m.write("etc/passwd", "root:x:0:0:root:/root:/bin/bash\ndaemon:x:1:"+strconv.Itoa(m.ci.GID)+":daemon:/home/ci:/bin/bash\n", 0o644)
	why := refused(t, m.open(), "daemon", key(t, 1, "")).Error()
	if !strings.Contains(why, "daemon is a system account (uid 1)") || m.there(ciAK) {
		t.Fatalf("Apply for a system account = %q; want it refused, nothing made", why)
	}
}

func TestAHomeUnderAFolderOthersCanWriteIsRefused(t *testing.T) {
	m := newMachine(t)
	m.mkdir("home", 0o777)
	why := refused(t, m.open(), "ci", key(t, 1, "")).Error()
	if !strings.Contains(why, "/home is writable by others") || m.there(ciAK) {
		t.Fatalf("Apply under a /home others can write = %q; want it refused, nothing made", why)
	}
	if m.self.UID != 0 {
		return
	}
	m.mkdir("home", 0o755)
	m.chown("home", m.ci)
	why = refused(t, m.open(), "ci", key(t, 1, "")).Error()
	if !strings.Contains(why, "/home is not root's") || m.there(ciAK) {
		t.Fatalf("Apply under a /home ci owns = %q; want it refused, nothing made", why)
	}
}

func TestOffOnAnEditedBlockSaysTheKeyStillSignsIn(t *testing.T) {
	m := newMachine(t)
	a := m.open()
	apply(t, a, "root", key(t, 1, "ci"))
	m.write(rootAK, strings.Replace(m.content(rootAK), " ci\n", " mine\n", 1), 0o600)
	edited := m.content(rootAK)
	_, err := a.Undo(context.Background(), setting())
	if err == nil || !strings.Contains(err.Error(), "still signs in") || !strings.Contains(err.Error(), "lines 1 to 3 of /root/.ssh/authorized_keys") || m.content(rootAK) != edited {
		t.Fatalf("Undo of an edited block = %v; file %q; want it kept, and a refusal naming its lines", err, m.content(rootAK))
	}
}

func TestWithoutAdminAndNoRecordTheReadIsNeverRefused(t *testing.T) {
	m := newMachine(t)
	if m.self.UID == 0 {
		t.Skip("root reads every folder")
	}
	m.mkdir("root", 0o000)
	t.Cleanup(func() { _ = m.disk.Chmod(m.at("root"), 0o700) })
	it := setting()
	it.Inputs = []catalog.Input{{Name: "user", Default: "root"}, {Name: "key"}}
	o, err := m.open().Observe(context.Background(), it)
	if err != nil || (o.State != catalog.StateAbsent && !o.Blind) {
		t.Fatalf("Observe without admin = %+v, %v; want absent or blind", o, err)
	}
}

func TestOffWithNoRecordFindsABlockInEveryAccountThatSignsIn(t *testing.T) {
	m := newMachine(t)
	m.mkdir("home/ci/.ssh", 0o700)
	m.chown("home/ci/.ssh", m.ci)
	own := key(t, 9, "laptop") + "\n"
	m.write(ciAK, own+block(key(t, 1, "x")), 0o600)
	m.chown(ciAK, m.ci)
	it := setting()
	it.Inputs = []catalog.Input{{Name: "user", Default: "root"}, {Name: "key"}}
	_, err := m.open().Undo(context.Background(), it)
	if err != nil || m.content(ciAK) != own {
		t.Fatalf("Undo with no record = %v; ci's file %q; want the block found and removed", err, m.content(ciAK))
	}
}
