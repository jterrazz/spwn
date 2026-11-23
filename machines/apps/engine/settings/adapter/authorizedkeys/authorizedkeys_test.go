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

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/authorizedkeys"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
)

// value: a machine on the real disk, its system root a folder of the
// test's own: as root, ci is uid 1001 and every owner is the system's; as
// any other user every account is the run's, as in a world.
type machine struct {
	t        *testing.T
	root     string
	disk     local.FS
	self, ci local.Owner
	run      *local.Owner
}

func newMachine(t *testing.T) *machine {
	t.Helper()
	m := &machine{t: t, root: t.TempDir(), disk: local.New(nil, nil, nil, nil).FS, self: local.Self()}
	m.ci = m.self
	if m.self.UID == 0 {
		m.ci = local.Owner{UID: 1001, GID: 1002}
	} else {
		m.run = &m.self
	}
	m.write("etc/passwd", "root:x:0:0:root:/root:/bin/bash\nci:x:"+strconv.Itoa(m.ci.UID)+":"+strconv.Itoa(m.ci.GID)+":CI:/home/ci:/bin/bash\n", 0o644)
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
	return authorizedkeys.New(files, local.Beneath{}, authorizedkeys.Options{Root: m.root, Record: m.at(authorizedkeys.Record), Run: m.run})
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

func TestLooseModesAreSetAsSshdAsks(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o755)
	m.write(rootAK, key(t, 9, "")+"\n", 0o664)
	apply(t, m.open(), "root", key(t, 1, ""))
	if dir, f := m.entry("root/.ssh").Mode.Perm(), m.entry(rootAK).Mode.Perm(); dir != 0o700 || f != 0o600 {
		t.Fatalf("folder %o, file %o; want 700 and 600", dir, f)
	}
}

func TestStrictModesAreKept(t *testing.T) {
	m := newMachine(t)
	m.mkdir("root/.ssh", 0o700)
	m.write(rootAK, key(t, 9, "")+"\n", 0o600)
	apply(t, m.open(), "root", key(t, 1, ""))
	if dir, f := m.entry("root/.ssh").Mode.Perm(), m.entry(rootAK).Mode.Perm(); dir != 0o700 || f != 0o600 {
		t.Fatalf("folder %o, file %o; want 700 and 600", dir, f)
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
	if err != nil || len(off.Steps) != 2 || off.Steps[0].Detail != "yours, kept" || off.Steps[1].Detail != "removed" || m.content(rootAK) != edited {
		t.Fatalf("Undo = %+v, %v; want root's kept, ci's removed", off.Steps, err)
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
	if err != nil || o.State != catalog.StateDegraded || o.Diff == "" {
		t.Fatalf("Observe = %+v, %v; want degraded with a diff", o, err)
	}
	edited := m.content(rootAK)
	change, err := a.Undo(context.Background(), setting())
	if err != nil || m.content(rootAK) != edited || detail(change) != "yours, kept" {
		t.Fatalf("Undo = %+v, %v; file %q; want it kept", change, err, m.content(rootAK))
	}
}
