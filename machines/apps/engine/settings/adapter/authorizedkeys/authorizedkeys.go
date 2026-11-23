package authorizedkeys

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type Files interface {
	ReadFile(path string) ([]byte, error)
	Replace(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Remove(path string) error
}

type Home interface {
	// value: the files under a folder the system owns, a user's home among
	// them, each reached without following a link (local.Beneath).
	Lstat(base, rel string) (local.Entry, error)
	ReadFile(base, rel string) ([]byte, error)
	Replace(base, rel string, data []byte, perm fs.FileMode, o local.Owner) error
	Mkdir(base, rel string, perm fs.FileMode, o local.Owner) error
	Chmod(base, rel string, perm fs.FileMode) error
	Chown(base, rel string, o local.Owner) error
}

type Options struct {
	// value: the folder the system's absolute paths are read under ("" on
	// the machine itself), and the file station keeps what it last wrote in.
	Root, Record string
	// value: in a world, the owner every account's files carry, the run's
	// own, so a world gives no file away; nil on a machine.
	Run *local.Owner
	// value: the lock every writer of a block takes from its read to its
	// rename (Lock, beside Record), and how it is taken (the system's
	// ExclusiveWithin).
	Lock string
	Take Exclusive
}

type Exclusive func(ctx context.Context, path string, within time.Duration) (release func(), held bool, err error)

const wait = 10 * time.Second

type Applier struct {
	files Files
	home  Home
	o     Options
}

var _ contract.Applier = (*Applier)(nil)

func New(files Files, home Home, o Options) *Applier {
	return &Applier{files: files, home: home, o: o}
}

func (*Applier) Kind() catalog.EffectKind { return catalog.EffectAuthorizedKeys }

const (
	InputUser = "user"
	InputKey  = "key"
	// value: the machine's own record of every block station wrote, root's,
	// the same whoever runs station: never under the home of a sudo's
	// invoker.
	Record = "/var/lib/station/authorized-keys.json"
	Lock   = "/var/lib/station/authorized-keys.lock"
)

// value: every authorized_keys station wrote its block in and has not
// taken it out of yet, as the system names it, with the block's text; the
// last is the one the last apply wrote.
type record struct {
	Files []written `json:"files"`
}

type written struct {
	File string `json:"file"`
	Body string `json:"body"`
}

type account struct {
	name, home, shell string
	local.Owner
}

func of(it catalog.Setting) (catalog.AuthorizedKeys, error) {
	a, ok := it.Effect.(catalog.AuthorizedKeys)
	if !ok || a.Block == "" {
		return catalog.AuthorizedKeys{}, errors.New(it.Name + " is not authorized keys Machines writes")
	}
	return a, nil
}

func (a *Applier) at(p string) string { return path.Join("/", a.o.Root, p) }

func (a *Applier) read() (record, error) {
	data, err := a.files.ReadFile(a.o.Record)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return record{}, nil
	case err != nil:
		return record{}, err
	}
	var r record
	if err := json.Unmarshal(data, &r); err != nil {
		return record{}, fmt.Errorf("%s: %w", a.o.Record, err)
	}
	return r, nil
}

// invariant: a record with no file left is no record: its file is removed.
func (a *Applier) save(r record) error {
	if len(r.Files) == 0 {
		err := a.files.Remove(a.o.Record)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	err = a.files.MkdirAll(path.Dir(a.o.Record), 0o700)
	if err != nil {
		return err
	}
	return a.files.Replace(a.o.Record, data, 0o600)
}

// value: what taking station's block out of w's file did, whether the
// record keeps w, and why a block of station's stays there: a block gone
// already, or taken out, is no longer station's to hold; one the owner
// changed is kept as it is and stays recorded, and so does one in a file
// station cannot read or write, with that error.
func (a *Applier) takeOut(w written, id string) (settings.ChangeStep, bool, error) {
	text, _, err := a.text(w.File)
	if err != nil {
		return settings.ChangeStep{Label: w.File, State: catalog.StateBroken, Detail: "not touched"}, true, err
	}
	have, state := settings.PackageBlock(text, id)
	switch {
	case state == settings.BlockAbsent:
		return settings.ChangeStep{Label: w.File, State: catalog.StateOK, Detail: "already gone"}, false, nil
	case state != settings.BlockHere || have != w.Body:
		return settings.ChangeStep{Label: w.File, State: catalog.StateOK, Detail: "yours, kept"}, true, nil
	}
	updated, _ := settings.RemovePackageBlock(text, id)
	base, rel := a.beneath(w.File)
	err = a.home.Replace(base, rel, []byte(updated), 0o600, local.Owner{})
	if err != nil {
		return settings.ChangeStep{Label: w.File, State: catalog.StateBroken, Detail: "not touched"}, true, a.linked(err)
	}
	return settings.ChangeStep{Label: w.File, State: catalog.StateOK, Detail: "removed"}, false, nil
}

// value: the folder of the system that holds the home of the
// authorized_keys the system names file, and the file's path under it.
func (a *Applier) beneath(file string) (base, rel string) {
	home := path.Dir(path.Dir(file))
	return a.at(path.Dir(home)), path.Join(path.Base(home), ".ssh", "authorized_keys")
}

func (a *Applier) text(file string) (text string, there bool, err error) {
	data, err := a.home.ReadFile(a.beneath(file))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", false, nil
	case err != nil:
		return "", false, a.linked(err)
	}
	return string(data), true, nil
}

func (a *Applier) Observe(_ context.Context, it catalog.Setting) (settings.Observation, error) {
	k, err := of(it)
	if err != nil {
		return settings.Observation{}, err
	}
	rec, err := a.read()
	switch {
	case errors.Is(err, fs.ErrPermission):
		return settings.Observation{State: catalog.StateUnknown, Detail: "needs admin to read " + strings.TrimPrefix(a.o.Record, path.Join("/", a.o.Root)), Blind: true}, nil
	case err != nil:
		return settings.Observation{State: catalog.StateAbsent}, err
	}
	var r written
	unrecorded := len(rec.Files) == 0
	if unrecorded {
		file, ok := a.fallback(it)
		if !ok {
			return settings.Observation{State: catalog.StateAbsent}, nil
		}
		r.File = file
	} else {
		r = rec.Files[len(rec.Files)-1]
	}
	text, _, err := a.text(r.File)
	if err != nil {
		return settings.Observation{}, err
	}
	o := settings.Observation{Where: r.File}
	have, state := settings.PackageBlock(text, k.Block)
	switch {
	case state == settings.BlockAbsent:
		o.State = catalog.StateAbsent
		return o, nil
	case state == settings.BlockHere && (unrecorded || have == r.Body):
		o.State, o.Detail = catalog.StateOK, r.File
		l, err := a.loose(r.File, it.Name)
		if err != nil {
			return settings.Observation{}, err
		}
		if l.why != "" {
			o.State, o.Detail, o.Loose = catalog.StateDegraded, l.why, true
			o.Diff = settings.UnifiedDiff(l.want, l.have, "Machines' "+l.at, l.at)
			have += "\x00" + l.have
		}
	default:
		o.State, o.Detail = catalog.StateDegraded, "differs from Machines': station settings apply "+it.Name
		o.Diff = settings.UnifiedDiff(r.Body, have, "Machines' "+r.File, r.File)
	}
	sum := sha256.Sum256([]byte(r.File + "\x00" + have))
	o.Digest = "sha256:" + hex.EncodeToString(sum[:])
	return o, nil
}

func (a *Applier) Apply(ctx context.Context, it catalog.Setting, values map[string]string) (settings.Change, error) {
	// invariant: every value is read, and refused, before the first write:
	// a key that does not parse never reaches the file sshd reads.
	k, err := of(it)
	if err != nil {
		return settings.Change{}, err
	}
	body, err := Keys(values[InputKey])
	if err != nil {
		return settings.Change{}, err
	}
	user := strings.TrimSpace(values[InputUser])
	who, err := a.account(user)
	if err != nil {
		return settings.Change{}, err
	}
	err = signsIn(who)
	if err != nil {
		return settings.Change{}, err
	}
	base, err := a.homeOf(who)
	if err != nil {
		return settings.Change{}, err
	}
	dir := path.Join(who.home, ".ssh")
	file := path.Join(dir, "authorized_keys")
	ssh := path.Join(path.Base(who.home), ".ssh")
	release, err := a.turn(ctx)
	if err != nil {
		return settings.Change{}, err
	}
	defer release()
	text, there, err := a.text(file)
	if err != nil {
		return settings.Change{}, err
	}
	if _, state := settings.PackageBlock(text, k.Block); state == settings.BlockDamaged {
		start, end := settings.BlockMarkers(k.Block)
		return settings.Change{}, fmt.Errorf("%s holds a damaged %s block, a start with no end or two blocks, not touched: leave one `%s` line and one `%s` line around Machines' keys, then apply again", file, k.Block, start, end)
	}
	updated, changed := settings.UpsertPackageBlock(text, k.Block, body)
	rec, err := a.read()
	if err != nil {
		return settings.Change{}, err
	}
	// invariant: the record names the file before its block is written, so
	// a run cut between the two leaves off a block it knows, never one it
	// does not: a recorded block that never landed reads gone.
	var olds []written
	for _, w := range rec.Files {
		if w.File != file {
			olds = append(olds, w)
		}
	}
	err = a.save(record{Files: append(slices.Clone(olds), written{File: file, Body: body})})
	if err != nil {
		return settings.Change{}, err
	}
	err = a.folder(base, ssh, who)
	if err != nil {
		return settings.Change{}, err
	}
	if changed || !there {
		_, rel := a.beneath(file)
		err = a.home.Replace(base, rel, []byte(updated), 0o600, who.Owner)
		if err != nil {
			return settings.Change{}, a.linked(err)
		}
	}
	for _, p := range []struct {
		rel  string
		mode fs.FileMode
	}{{ssh, 0o700}, {path.Join(ssh, "authorized_keys"), 0o600}} {
		err = a.home.Chmod(base, p.rel, p.mode)
		if err != nil {
			return settings.Change{}, a.linked(err)
		}
		err = a.home.Chown(base, p.rel, who.Owner)
		if err != nil {
			return settings.Change{}, a.linked(err)
		}
	}
	detail := "written"
	if !changed && there {
		detail = "already there"
	}
	steps := []settings.ChangeStep{{Label: file, State: catalog.StateOK, Detail: detail}}
	var kept []written
	var stays []error
	for _, w := range olds {
		step, keep, why := a.takeOut(w, k.Block)
		steps = append(steps, step)
		if keep {
			kept = append(kept, w)
		}
		stays = append(stays, why)
	}
	err = a.save(record{Files: append(kept, written{File: file, Body: body})})
	if err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: steps}, errors.Join(stays...)
}

func (a *Applier) Undo(ctx context.Context, it catalog.Setting) (settings.Change, error) {
	// invariant: the block leaves every file station wrote it in, each only
	// while it is still exactly station's, and nothing else of a file moves:
	// off never destroys a line of the owner's.
	k, err := of(it)
	if err != nil {
		return settings.Change{}, err
	}
	release, err := a.turn(ctx)
	if err != nil {
		return settings.Change{}, err
	}
	defer release()
	rec, err := a.read()
	if err != nil {
		return settings.Change{}, err
	}
	if len(rec.Files) == 0 {
		return a.unrecorded(it, k.Block)
	}
	var steps []settings.ChangeStep
	var kept []written
	var stays []error
	for _, w := range rec.Files {
		step, keep, why := a.takeOut(w, k.Block)
		if step.Detail != "already gone" {
			steps = append(steps, step)
		}
		if keep {
			kept = append(kept, w)
		}
		stays = append(stays, why)
	}
	err = a.save(record{Files: kept})
	if err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: steps}, errors.Join(stays...)
}

// invariant: with no record, the block's markers are the record: off looks
// in the file of the package's default user, and a block of station's
// there leaves it, its owner's lines as they were; a damaged one is
// refused, so off never reads done while a block of station's is left.
type loosened struct {
	// value: why the keys are ignored, at what path, and the mode and owner
	// Machines sets there against what is there now; why empty when strict.
	why, at, want, have string
}

func (a *Applier) loose(file, name string) (loosened, error) {
	// invariant: the keys count only while the sign-in server reads them:
	// the folder 0700 and the file 0600, nothing for group or others, and
	// both the home owner's or root's; anything else is drift, with apply as
	// its fix.
	base, rel := a.beneath(file)
	home := path.Dir(path.Dir(rel))
	owner, err := a.home.Lstat(base, home)
	if err != nil {
		return loosened{}, a.linked(err)
	}
	for _, p := range []struct {
		rel  string
		mode fs.FileMode
	}{{path.Dir(rel), 0o700}, {rel, 0o600}} {
		var e local.Entry
		e, err = a.home.Lstat(base, p.rel)
		if err != nil {
			return loosened{}, a.linked(err)
		}
		l := loosened{at: path.Join(path.Dir(path.Dir(file)), strings.TrimPrefix(p.rel, home+"/"))}
		switch {
		case e.Mode.Perm()&0o077 != 0:
			l.want, l.have = octal(p.mode)+"\n", octal(e.Mode.Perm())+"\n"
			l.why = l.at + " is " + octal(e.Mode.Perm()) + ", open to others, so its keys are ignored: station settings apply " + name + " sets it to " + octal(p.mode)
			return l, nil
		case e.UID != owner.UID && e.UID != 0:
			l.want, l.have = fmt.Sprintf("uid %d\n", owner.UID), fmt.Sprintf("uid %d\n", e.UID)
			l.why = fmt.Sprintf("%s is uid %d's, not the home's (uid %d), so its keys are ignored: station settings apply %s gives it back", l.at, e.UID, owner.UID, name)
			return l, nil
		}
	}
	return loosened{}, nil
}

func octal(m fs.FileMode) string { return "0" + strconv.FormatUint(uint64(m.Perm()), 8) }

func (a *Applier) turn(ctx context.Context) (func(), error) {
	release, held, err := a.o.Take(ctx, a.o.Lock, wait)
	if err != nil {
		return nil, err
	}
	if held {
		return nil, fmt.Errorf("%s is held by another run writing these keys, nothing touched: apply again once it ends", strings.TrimPrefix(a.o.Lock, path.Join("/", a.o.Root)))
	}
	return release, nil
}

func (a *Applier) fallback(it catalog.Setting) (string, bool) {
	user := ""
	for _, in := range it.Inputs {
		if in.Name == InputUser {
			user = in.Default
		}
	}
	if user == "" {
		return "", false
	}
	who, err := a.account(user)
	if err != nil {
		return "", false
	}
	return path.Join(who.home, ".ssh", "authorized_keys"), true
}

func (a *Applier) unrecorded(it catalog.Setting, id string) (settings.Change, error) {
	file, ok := a.fallback(it)
	if !ok {
		return settings.Change{}, nil
	}
	text, _, err := a.text(file)
	if err != nil {
		return settings.Change{}, err
	}
	switch _, state := settings.PackageBlock(text, id); state {
	case settings.BlockAbsent:
		return settings.Change{}, nil
	case settings.BlockDamaged:
		return settings.Change{}, fmt.Errorf("%s holds a damaged %s block Machines has no record of, not touched: remove it by hand", file, id)
	case settings.BlockHere:
	}
	updated, _ := settings.RemovePackageBlock(text, id)
	base, rel := a.beneath(file)
	err = a.home.Replace(base, rel, []byte(updated), 0o600, local.Owner{})
	if err != nil {
		return settings.Change{}, a.linked(err)
	}
	return settings.Change{Steps: []settings.ChangeStep{{Label: file, State: catalog.StateOK, Detail: "removed, found with no record"}}}, nil
}

func Keys(given string) (string, error) {
	// value: the block's text for the public keys given one per line, each
	// `type base64 [comment]`, or the refusal of the first that does not parse.
	var b strings.Builder
	n := 0
	for i, line := range strings.Split(given, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, err := parse(line)
		if err != nil {
			return "", fmt.Errorf("line %d of the key %w", i+1, err)
		}
		b.WriteString(key + "\n")
		n++
	}
	if n == 0 {
		return "", errors.New("no public key given: `type base64 [comment]`, one per line")
	}
	return b.String(), nil
}

func fields(kind string) int {
	// value: how many length-prefixed strings a key type's blob holds, its
	// type the first; 0 for a type Machines does not write (a hardware
	// key's sk- types among them).
	switch kind {
	case "ssh-ed25519":
		return 2
	case "ssh-rsa", "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521":
		return 3
	}
	return 0
}

// invariant: a key is its type, then its blob in base64, whose first string
// is that type and whose strings, as many as the type holds, fill it
// exactly; an ed25519 key is 32 bytes. Anything before the type (sshd's
// options) is refused.
func parse(line string) (string, error) {
	f := strings.Fields(line)
	if len(f) > 1 && fields(f[0]) == 0 {
		for _, w := range f[1:] {
			if fields(w) > 0 {
				return "", errors.New("carries options: a key alone, `type base64 [comment]`")
			}
		}
	}
	refused := errors.New("is not a public key: `type base64 [comment]`")
	if len(f) < 2 || fields(f[0]) == 0 {
		return "", refused
	}
	blob, err := base64.StdEncoding.DecodeString(f[1])
	if err != nil {
		return "", refused
	}
	var parts [][]byte
	for len(blob) > 0 {
		if len(blob) < 4 {
			return "", refused
		}
		size, rest := int64(binary.BigEndian.Uint32(blob)), blob[4:]
		if size > int64(len(rest)) {
			return "", refused
		}
		parts, blob = append(parts, rest[:size]), rest[size:]
	}
	if len(parts) != fields(f[0]) || string(parts[0]) != f[0] || (f[0] == "ssh-ed25519" && len(parts[1]) != 32) {
		return "", refused
	}
	key := f[0] + " " + f[1]
	if comment := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(line, f[0])), f[1])); comment != "" {
		key += " " + comment
	}
	return key, nil
}

// invariant: a user's home is a folder there already, the user's own,
// never a link: Machines writes in a home, it never makes one. The value is
// the folder that holds it, which the system owns, read under the root;
// every path under it is reached without following a link.
func (a *Applier) homeOf(who account) (string, error) {
	if who.home == "/" {
		return "", fmt.Errorf("%s has no home of its own (/): Machines writes only in a user's home", who.name)
	}
	base := a.at(path.Dir(who.home))
	e, err := a.home.Lstat(base, path.Base(who.home))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", fmt.Errorf("%s has no home on this machine (%s is not there): Machines never makes one", who.name, who.home)
	case err != nil:
		return "", a.linked(err)
	case e.Mode&fs.ModeSymlink != 0:
		return "", fmt.Errorf("%s is a link, not touched: Machines follows no link in a home", who.home)
	case !e.Mode.IsDir():
		return "", fmt.Errorf("%s is not a folder, not touched", who.home)
	case e.UID != who.UID:
		return "", fmt.Errorf("%s is not %s's (its owner is uid %d), not touched", who.home, who.name, e.UID)
	}
	return base, nil
}

// invariant: a key goes only to an account that signs in: one whose shell
// is nologin or false never does, and a key there would still open its
// tunnels; root is always taken.
func signsIn(who account) error {
	if who.name == "root" {
		return nil
	}
	switch path.Base(who.shell) {
	case "nologin", "false":
		return fmt.Errorf("%s does not sign in (its shell is %s), not touched: Machines writes keys only for an account that signs in", who.name, who.shell)
	}
	return nil
}

// invariant: ~/.ssh is a real folder, made the user's at 0700 when it is
// not there; a link there is refused, never followed.
func (a *Applier) folder(base, rel string, who account) error {
	e, err := a.home.Lstat(base, rel)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return a.linked(a.home.Mkdir(base, rel, 0o700, who.Owner))
	case err != nil:
		return a.linked(err)
	case e.Mode&fs.ModeSymlink != 0:
		return fmt.Errorf("%s is a link, not touched: Machines follows no link in a home", path.Join(path.Dir(who.home), rel))
	case !e.Mode.IsDir():
		return fmt.Errorf("%s is not a folder, not touched", path.Join(path.Dir(who.home), rel))
	}
	return nil
}

// value: err, a link met on the way said in the owner's words, by the
// path the system names it.
func (a *Applier) linked(err error) error {
	var pe *fs.PathError
	if errors.Is(err, local.ErrLink) && errors.As(err, &pe) {
		at := strings.TrimPrefix(pe.Path, path.Join("/", a.o.Root))
		return fmt.Errorf("%s is a link, not touched: Machines follows no link in a home", path.Join("/", at))
	}
	return err
}

// invariant: root's home is /root; any other user's, and its ids, are its
// line of the system's /etc/passwd. In a world every account is the run's.
func (a *Applier) account(user string) (account, error) {
	if user == "" {
		return account{}, errors.New("no user given: the account whose ~/.ssh/authorized_keys takes the key")
	}
	data, err := a.files.ReadFile(a.at("/etc/passwd"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return account{}, err
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		f := strings.Split(line, ":")
		if len(f) < 7 || f[0] != user || !path.IsAbs(f[5]) {
			continue
		}
		uid, uerr := strconv.Atoi(f[2])
		gid, gerr := strconv.Atoi(f[3])
		if uerr == nil && gerr == nil {
			return a.as(account{name: user, home: path.Clean(f[5]), shell: f[6], Owner: local.Owner{UID: uid, GID: gid}}), nil
		}
	}
	if user == "root" {
		return a.as(account{name: user, home: "/root"}), nil
	}
	return account{}, fmt.Errorf("no user %s on this machine", user)
}

func (a *Applier) as(who account) account {
	if a.o.Run != nil {
		who.Owner = *a.o.Run
	}
	return who
}
