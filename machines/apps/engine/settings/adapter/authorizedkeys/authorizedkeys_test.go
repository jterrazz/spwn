package authorizedkeys_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"io/fs"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/authorizedkeys"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
)

type file struct {
	data []byte
	mode fs.FileMode
	uid  int
	gid  int
}

type disk struct {
	files    map[string]*file
	replaced []string
}

func newDisk() *disk { return &disk{files: map[string]*file{}} }

func (d *disk) ReadFile(p string) ([]byte, error) {
	if f, ok := d.files[p]; ok && !f.mode.IsDir() {
		return append([]byte(nil), f.data...), nil
	}
	return nil, &fs.PathError{Op: "open", Path: p, Err: fs.ErrNotExist}
}

func (d *disk) Replace(p string, data []byte, perm fs.FileMode) error {
	d.replaced = append(d.replaced, p)
	if f, ok := d.files[p]; ok {
		f.data = append([]byte(nil), data...)
		return nil
	}
	d.files[p] = &file{data: append([]byte(nil), data...), mode: perm, uid: -1, gid: -1}
	return nil
}

func (d *disk) MkdirAll(p string, perm fs.FileMode) error {
	if _, ok := d.files[p]; !ok {
		d.files[p] = &file{mode: fs.ModeDir | perm, uid: -1, gid: -1}
	}
	return nil
}

func (d *disk) Chmod(p string, perm fs.FileMode) error {
	f, ok := d.files[p]
	if !ok {
		return &fs.PathError{Op: "chmod", Path: p, Err: fs.ErrNotExist}
	}
	f.mode = f.mode.Type() | perm
	return nil
}

func (d *disk) Remove(p string) error {
	if _, ok := d.files[p]; !ok {
		return &fs.PathError{Op: "remove", Path: p, Err: fs.ErrNotExist}
	}
	delete(d.files, p)
	return nil
}

func (d *disk) give(p string, uid, gid int) error {
	f, ok := d.files[p]
	if !ok {
		return &fs.PathError{Op: "chown", Path: p, Err: fs.ErrNotExist}
	}
	f.uid, f.gid = uid, gid
	return nil
}

const (
	root     = "/sys"
	record   = "/state/authorized-keys.json"
	rootAK   = root + "/root/.ssh/authorized_keys"
	ciAK     = root + "/home/ci/.ssh/authorized_keys"
	accounts = "root:x:0:0:root:/root:/bin/bash\nci:x:1001:1002:CI:/home/ci:/bin/bash\n"
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

func open(d *disk) *authorizedkeys.Applier {
	if _, ok := d.files[root+"/etc/passwd"]; !ok {
		d.files[root+"/etc/passwd"] = &file{data: []byte(accounts), mode: 0o644}
	}
	return authorizedkeys.New(d, authorizedkeys.Options{Root: root, Record: record, Give: d.give})
}

func apply(t *testing.T, a *authorizedkeys.Applier, user, keys string) {
	t.Helper()
	if _, err := a.Apply(context.Background(), setting(), map[string]string{"user": user, "key": keys}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
}

func content(d *disk, p string) string {
	if f, ok := d.files[p]; ok {
		return string(f.data)
	}
	return ""
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
		d := newDisk()
		d.files[rootAK] = &file{data: []byte(key(t, 9, "owner@laptop") + "\n"), mode: 0o600}
		edit := func(*testing.T) {
			d.files[rootAK].data = bytes.Replace(d.files[rootAK].data, []byte("ci@deploy"), []byte("mine"), 1)
		}
		return contracttest.Opened{
			Applier: open(d), Package: setting(), Edit: edit,
			Values: map[string]string{"user": "root", "key": key(t, 1, "ci@deploy")},
		}
	})
}

func TestAnEmptyFileTakesTheBlockAlone(t *testing.T) {
	d := newDisk()
	d.files[rootAK] = &file{mode: 0o600}
	apply(t, open(d), "root", key(t, 1, ""))
	if got, want := content(d, rootAK), block(key(t, 1, "")); got != want {
		t.Fatalf("authorized_keys = %q; want %q", got, want)
	}
}

func TestAFileThatIsNotThereIsMadeWithTheModesSshdAsks(t *testing.T) {
	d := newDisk()
	apply(t, open(d), "ci", key(t, 1, ""))
	dir, f := d.files[root+"/home/ci/.ssh"], d.files[ciAK]
	if dir == nil || f == nil || dir.mode.Perm() != 0o700 || f.mode.Perm() != 0o600 {
		t.Fatalf("folder %+v, file %+v; want 0700 and 0600", dir, f)
	}
	if dir.uid != 1001 || dir.gid != 1002 || f.uid != 1001 || f.gid != 1002 {
		t.Fatalf("folder %d:%d, file %d:%d; want both the user's, 1001:1002", dir.uid, dir.gid, f.uid, f.gid)
	}
	if got := string(f.data); got != block(key(t, 1, "")) {
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
			d := newDisk()
			d.files[rootAK] = &file{data: []byte(own), mode: 0o600}
			a := open(d)
			apply(t, a, "root", key(t, 1, "ci@deploy"))
			written := content(d, rootAK)
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
			if got := content(d, rootAK); got != own {
				t.Fatalf("after Undo authorized_keys = %q; want the owner's bytes %q", got, own)
			}
		})
	}
}

func TestABlockAlreadyThereIsNotWrittenAgain(t *testing.T) {
	d := newDisk()
	a := open(d)
	apply(t, a, "root", key(t, 1, ""))
	before := content(d, rootAK)
	d.replaced = nil
	change, err := a.Apply(context.Background(), setting(), map[string]string{"user": "root", "key": key(t, 1, "")})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range d.replaced {
		if p == rootAK {
			t.Fatalf("authorized_keys was rewritten: %v", d.replaced)
		}
	}
	if got := content(d, rootAK); got != before || detail(change) != "already there" {
		t.Fatalf("authorized_keys = %q, step %+v; want %q, already there", got, change.Steps, before)
	}
}

func TestLooseModesAreSetAsSshdAsks(t *testing.T) {
	d := newDisk()
	d.files[root+"/root/.ssh"] = &file{mode: fs.ModeDir | 0o755}
	d.files[rootAK] = &file{data: []byte(key(t, 9, "") + "\n"), mode: 0o664}
	apply(t, open(d), "root", key(t, 1, ""))
	if dir, f := d.files[root+"/root/.ssh"].mode.Perm(), d.files[rootAK].mode.Perm(); dir != 0o700 || f != 0o600 {
		t.Fatalf("folder %o, file %o; want 700 and 600", dir, f)
	}
}

func TestStrictModesAreKept(t *testing.T) {
	d := newDisk()
	d.files[root+"/root/.ssh"] = &file{mode: fs.ModeDir | 0o700}
	d.files[rootAK] = &file{data: []byte(key(t, 9, "") + "\n"), mode: 0o600}
	apply(t, open(d), "root", key(t, 1, ""))
	if dir, f := d.files[root+"/root/.ssh"].mode.Perm(), d.files[rootAK].mode.Perm(); dir != 0o700 || f != 0o600 {
		t.Fatalf("folder %o, file %o; want 700 and 600", dir, f)
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
	d := newDisk()
	d.files[rootAK] = &file{data: []byte("mine\n"), mode: 0o644}
	_, err := open(d).Apply(context.Background(), setting(), map[string]string{"user": "root", "key": "ssh-ed25519 AAAA"})
	if err == nil || len(d.replaced) != 0 || d.files[rootAK].mode.Perm() != 0o644 {
		t.Fatalf("Apply = %v, wrote %v, mode %o; want a refusal and nothing touched", err, d.replaced, d.files[rootAK].mode.Perm())
	}
}

func TestAnUnknownUserIsRefused(t *testing.T) {
	d := newDisk()
	_, err := open(d).Apply(context.Background(), setting(), map[string]string{"user": "nobody", "key": key(t, 1, "")})
	if err == nil || err.Error() != "no user nobody on this machine" || len(d.replaced) != 0 {
		t.Fatalf("Apply = %v, wrote %v", err, d.replaced)
	}
}

func TestAHandEditInsideTheBlockReadsDriftAndSurvivesUndo(t *testing.T) {
	d := newDisk()
	a := open(d)
	apply(t, a, "root", key(t, 1, "ci"))
	if d.files[rootAK] == nil {
		t.Fatal("no authorized_keys written")
	}
	d.files[rootAK].data = bytes.Replace(d.files[rootAK].data, []byte(" ci\n"), []byte(" mine\n"), 1)
	o, err := a.Observe(context.Background(), setting())
	if err != nil || o.State != catalog.StateDegraded || o.Diff == "" {
		t.Fatalf("Observe = %+v, %v; want degraded with a diff", o, err)
	}
	edited := content(d, rootAK)
	change, err := a.Undo(context.Background(), setting())
	if err != nil || content(d, rootAK) != edited || detail(change) != "yours, kept" {
		t.Fatalf("Undo = %+v, %v; file %q; want it kept", change, err, d.files[rootAK].data)
	}
}
