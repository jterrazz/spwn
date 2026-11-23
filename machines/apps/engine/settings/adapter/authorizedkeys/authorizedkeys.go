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
	"maps"
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
	// value: the folders under a folder the system owns, a user's home
	// among them, each reached without following a link (local.Beneath):
	// a ~/.ssh opened, or held, once, and every name in it reached through
	// its descriptor.
	Lstat(base, rel string) (local.Entry, error)
	Mkdir(base, rel string, perm fs.FileMode, o local.Owner) error
	Open(base, rel string) (*local.Folder, error)
	Hold(ctx context.Context, base, rel string, within time.Duration) (*local.Folder, bool, error)
}

type Options struct {
	// value: the folder the system's absolute paths are read under ("" on
	// the machine itself), and the file station keeps what it last wrote in.
	Root, Record string
	// value: in a world, the owner every account's files carry, the run's
	// own, so a world gives no file away; nil on a machine.
	Run *local.Owner
	// value: the lock every run of the setting takes around its read and
	// its write of Record, beside it, and how it is taken (the system's
	// ExclusiveWithin); each ~/.ssh it writes in is held inside it.
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

var (
	_ contract.Applier = (*Applier)(nil)
	_ contract.Keeper  = (*Applier)(nil)
)

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
// taken it out of yet, as the system names it, with the block's text and
// whether station made the file and its ~/.ssh; the last is the one the
// last apply wrote. Was is the block station's file held when an apply
// began writing another, kept as station's until that write lands.
type record struct {
	Files []written `json:"files"`
}

type written struct {
	File       string `json:"file"`
	Body       string `json:"body"`
	MadeFile   bool   `json:"made_file,omitempty"`
	MadeFolder bool   `json:"made_folder,omitempty"`
	Was        string `json:"was,omitempty"`
}

// value: have, the block a file holds, is one station wrote there: the
// last, or the one a write not landed yet would have replaced.
func (w written) ours(have string) bool { return have == w.Body || w.Was != "" && have == w.Was }

type account struct {
	name, home, shell string
	// value: the uid /etc/passwd gives the account, before a world makes
	// every account the run's.
	uid int
	local.Owner
}

// value: the first uid Linux gives a person's account; every uid below it
// but root's is the system's.
const firstPerson = 1000

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

func (a *Applier) takeOut(w written, h holding, id string) ([]settings.ChangeStep, bool, error) {
	switch {
	case errors.Is(h.err, fs.ErrNotExist):
		return []settings.ChangeStep{{Label: w.File, State: catalog.StateOK, Detail: "already gone"}}, false, nil
	case h.err != nil:
		return []settings.ChangeStep{{Label: w.File, State: catalog.StateBroken, Detail: "not touched"}}, true, cannot(h.err)
	}
	text, e, there, err := a.textIn(h.folder)
	if err == nil {
		err = a.owned(h.folder, w.File, e, there, h.owner)
	}
	if err != nil {
		return []settings.ChangeStep{{Label: w.File, State: catalog.StateBroken, Detail: "not touched"}}, true, cannot(err)
	}
	have, state := settings.PackageBlock(text, id)
	switch {
	case state == settings.BlockAbsent:
		return []settings.ChangeStep{{Label: w.File, State: catalog.StateOK, Detail: "already gone"}}, false, nil
	case state == settings.BlockDamaged || !w.ours(have):
		return []settings.ChangeStep{{Label: w.File, State: catalog.StateOK, Detail: "yours, kept"}}, true, stillSigns(w.File, text, id)
	}
	updated, _ := settings.RemovePackageBlock(text, id)
	seen := local.Seen{Data: []byte(text), There: true}
	if updated == "" && w.MadeFile {
		return a.unmade(w, h.folder, seen)
	}
	err = h.folder.Replace(keysFile, seen, []byte(updated), 0o600, local.Owner{})
	if err != nil {
		return []settings.ChangeStep{{Label: w.File, State: catalog.StateBroken, Detail: "not touched"}}, true, a.linked(err)
	}
	return []settings.ChangeStep{{Label: w.File, State: catalog.StateOK, Detail: "removed"}}, false, nil
}

// value: why a block of station's the owner changed stays, its keys still
// signing in: off ends on it, needing the owner (contract.ErrKept); an
// apply for another user keeps it recorded and goes on.
type edited struct{ error }

func (*edited) Is(target error) bool { return target == contract.ErrKept }

// value: why a file of station's block was left unread and untouched, so
// its keys may still sign in: the run ends on it needing the owner, every
// other file handled.
type unreached struct{ error }

func (*unreached) Is(target error) bool { return target == contract.ErrKept }

func cannot(err error) error {
	if err == nil || errors.Is(err, contract.ErrKept) {
		return err
	}
	return &unreached{err}
}

// value: the account /etc/passwd gives the home of file, or that home when
// none does.
func (a *Applier) whose(file string) string {
	home := path.Dir(path.Dir(file))
	data, _ := a.files.ReadFile(a.at("/etc/passwd"))
	for line := range strings.SplitSeq(string(data), "\n") {
		if f := strings.Split(line, ":"); len(f) >= 7 && path.Clean(f[5]) == home {
			return f[0]
		}
	}
	if home == "/root" {
		return "root"
	}
	return home
}

func stillSigns(file, text, id string) error {
	if _, state := settings.PackageBlock(text, id); state == settings.BlockDamaged {
		return &edited{fmt.Errorf("%s holds a damaged %s block, kept, so the Machines key still signs in: delete %s of %s, then the rest of the block", file, id, strings.Join(settings.DamagedLines(text, id), " and "), file)}
	}
	first, last := settings.BlockLines(text, id)
	return &edited{fmt.Errorf("%s keeps the Machines block you changed, so the Machines key still signs in: delete lines %d to %d of %s to take it out", file, first, last, file)}
}

func (a *Applier) Kept(_ context.Context, it catalog.Setting) error {
	// invariant: a recorded block the owner changed still lets the Machines
	// key sign in, and so may a block in a file off cannot read (a link, a
	// hard link, a permission), recorded or found by the no-record scan: off,
	// which keeps either, ends needing the owner. A record it cannot read
	// says nothing of the kind.
	k, err := of(it)
	if err != nil {
		return err
	}
	rec, err := a.read()
	if err != nil {
		return err
	}
	files := map[string]*written{}
	for i := range rec.Files {
		files[rec.Files[i].File] = &rec.Files[i]
	}
	if len(rec.Files) == 0 {
		homes, err := a.homes()
		if err != nil {
			return err
		}
		for _, file := range homes {
			files[file] = nil
		}
	}
	var why []error
	for _, file := range slices.Sorted(maps.Keys(files)) {
		text, _, err := a.text(file)
		if err != nil {
			why = append(why, cannot(err))
			continue
		}
		w := files[file]
		if have, state := settings.PackageBlock(text, k.Block); state == settings.BlockDamaged || w != nil && state == settings.BlockHere && !w.ours(have) {
			why = append(why, stillSigns(file, text, k.Block))
		}
	}
	return errors.Join(why...)
}

// invariant: a file station made that holds nothing once its block is out
// is taken away, and so is the ~/.ssh station made once it is empty: off
// leaves a home as station found it. A folder that holds anything else
// stays.
func (a *Applier) unmade(w written, f *local.Folder, seen local.Seen) ([]settings.ChangeStep, bool, error) {
	err := f.Remove(keysFile, seen)
	if err != nil {
		return []settings.ChangeStep{{Label: w.File, State: catalog.StateBroken, Detail: "not touched"}}, true, a.linked(err)
	}
	file := settings.ChangeStep{Label: w.File, State: catalog.StateOK, Detail: "removed, and the empty file Machines made"}
	if !w.MadeFolder {
		return []settings.ChangeStep{file}, false, nil
	}
	err = f.RemoveSelf()
	switch {
	case errors.Is(err, local.ErrNotEmpty):
		return []settings.ChangeStep{file}, false, nil
	case err != nil:
		return []settings.ChangeStep{file}, false, a.linked(err)
	}
	return []settings.ChangeStep{file, {Label: path.Dir(w.File), State: catalog.StateOK, Detail: "removed, the empty folder Machines made"}}, false, nil
}

// invariant: a ~/.ssh an apply made, and left before its record names it,
// is taken away while it is still empty, so no folder stays that off does
// not know station made.
func (a *Applier) unmakeFolder(base, ssh string) {
	f, err := a.home.Open(base, ssh)
	if err != nil {
		return
	}
	defer f.Close()
	_ = f.RemoveSelf()
}

// value: the folder of the system that holds the home of the
// authorized_keys the system names file, and the ~/.ssh under it.
func (a *Applier) beneath(file string) (base, ssh string) {
	home := path.Dir(path.Dir(file))
	return a.at(path.Dir(home)), path.Join(path.Base(home), ".ssh")
}

const keysFile = "authorized_keys"

// value: the ~/.ssh of each authorized_keys the system names, held, or
// why it is not: a folder not there answers fs.ErrNotExist.
type holding struct {
	folder *local.Folder
	err    error
	// value: the uid of the home's owner, the only owner of the folder and
	// the file a run writes in.
	owner int
}

// invariant: every ~/.ssh a run writes in is held from its read to its
// write, the lock the VM's guest takes too, each taken in the order of its
// home owner's uid; one another run holds for 10 seconds is left as it is,
// its account named, and every other is still handled: a user who holds
// their own ~/.ssh keeps no other account's keys in place.
func (a *Applier) holdAll(ctx context.Context, files []string) (held map[string]holding, release func(), err error) {
	uid := map[string]int{}
	for _, file := range files {
		base, ssh := a.beneath(file)
		e, missing := a.home.Lstat(base, path.Dir(ssh))
		uid[file] = e.UID
		if missing != nil {
			uid[file] = -1
		}
	}
	order := slices.Clone(files)
	slices.SortStableFunc(order, func(x, y string) int { return uid[x] - uid[y] })
	held = map[string]holding{}
	release = func() {
		for _, h := range held {
			if h.folder != nil {
				h.folder.Close()
			}
		}
	}
	for _, file := range order {
		base, ssh, err := a.reach(file)
		if err != nil {
			held[file] = holding{err: err, owner: uid[file]}
			continue
		}
		f, busy, why := a.home.Hold(ctx, base, ssh, wait)
		switch {
		case busy:
			held[file] = holding{err: &unreached{fmt.Errorf("%s is held by another run writing its keys, not touched: %s's keys stay as they are, run it again once it ends", path.Dir(file), a.whose(file))}, owner: uid[file]}
			continue
		case errors.Is(why, context.Canceled), errors.Is(why, context.DeadlineExceeded):
			release()
			return nil, nil, why
		}
		held[file] = holding{folder: f, err: a.linked(why), owner: uid[file]}
	}
	return held, release, nil
}

// value: the authorized_keys f holds, and what the descriptor it was read
// through is.
func (a *Applier) textIn(f *local.Folder) (text string, e local.Entry, there bool, err error) {
	data, e, err := f.ReadFile(keysFile)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", local.Entry{}, false, nil
	case err != nil:
		return "", local.Entry{}, false, a.linked(err)
	}
	return string(data), e, true, nil
}

// invariant: a run writes only in a ~/.ssh, and an authorized_keys there,
// that the home's owner (uid) owns, each read through the descriptor it
// was opened with; anything else is refused with nothing touched.
func (a *Applier) owned(f *local.Folder, file string, e local.Entry, there bool, uid int) error {
	folder, err := f.Stat()
	if err != nil {
		return err
	}
	for _, p := range []struct {
		at string
		e  local.Entry
		is bool
	}{{path.Dir(file), folder, true}, {file, e, there}} {
		if why := notTheHomes(p.at, p.e, uid); p.is && why != "" {
			return errors.New(strings.Replace(why, ":", ", not touched:", 1))
		}
	}
	return nil
}

// value: why what is at, e, is not the home's owner's (uid), empty when it
// is: the one rule apply refuses by and drift reads by.
func notTheHomes(at string, e local.Entry, uid int) string {
	if e.UID == uid {
		return ""
	}
	return fmt.Sprintf("%s is uid %d's, not the home's (uid %d): Machines writes keys only in a folder and a file the home's owner owns, give it back with %s", at, e.UID, uid, catalog.AsRoot(fmt.Sprintf("chown %d %s", uid, at)))
}

// value: the folder of the system that holds the home of file, and the
// ~/.ssh under it, once every folder above the home is root's alone: every
// read and write of a ~/.ssh reaches it through here.
func (a *Applier) reach(file string) (base, ssh string, err error) {
	err = a.above(path.Dir(path.Dir(file)))
	if err != nil {
		return "", "", err
	}
	base, ssh = a.beneath(file)
	return base, ssh, nil
}

// value: the authorized_keys the system names file, read through its
// ~/.ssh opened once, with no lock: a read never waits on a writer.
func (a *Applier) text(file string) (text string, there bool, err error) {
	base, ssh, err := a.reach(file)
	if err != nil {
		return "", false, err
	}
	f, err := a.home.Open(base, ssh)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", false, nil
	case err != nil:
		return "", false, a.linked(err)
	}
	defer f.Close()
	text, _, there, err = a.textIn(f)
	return text, there, err
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
		r.File, err = a.found(k.Block)
		if err != nil {
			return settings.Observation{State: catalog.StateAbsent}, err
		}
	} else {
		r = rec.Files[len(rec.Files)-1]
	}
	text, _, err := a.text(r.File)
	switch {
	case errors.Is(err, fs.ErrPermission):
		return settings.Observation{State: catalog.StateUnknown, Detail: "needs admin to read " + r.File, Blind: true}, nil
	case err != nil:
		return settings.Observation{}, err
	}
	o := settings.Observation{Where: r.File}
	have, state := settings.PackageBlock(text, k.Block)
	switch {
	case state == settings.BlockAbsent:
		o.State = catalog.StateAbsent
		return o, nil
	case state == settings.BlockHere && r.Was != "" && have == r.Was:
		o.State, o.Detail, o.Loose = catalog.StateDegraded, "differs from Machines': station settings apply "+it.Name, true
		o.Diff = settings.UnifiedDiff(r.Body, have, "Machines' "+r.File, r.File)
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
	file := path.Join(who.home, ".ssh", keysFile)
	ssh := path.Join(path.Base(who.home), ".ssh")
	release, err := a.turn(ctx)
	if err != nil {
		return settings.Change{}, err
	}
	defer release()
	rec, err := a.read()
	if err != nil {
		return settings.Change{}, err
	}
	_, err = a.home.Lstat(base, ssh)
	madeFolder := errors.Is(err, fs.ErrNotExist)
	err = a.folder(base, ssh, who)
	if err != nil {
		return settings.Change{}, err
	}
	recorded := false
	if madeFolder {
		defer func() {
			if !recorded {
				a.unmakeFolder(base, ssh)
			}
		}()
	}
	files := []string{file}
	for _, w := range rec.Files {
		if w.File != file {
			files = append(files, w.File)
		}
	}
	held, letGo, err := a.holdAll(ctx, files)
	if err != nil {
		return settings.Change{}, err
	}
	defer letGo()
	if held[file].err != nil {
		return settings.Change{}, held[file].err
	}
	f := held[file].folder
	text, e, there, err := a.textIn(f)
	if err != nil {
		return settings.Change{}, err
	}
	err = a.owned(f, file, e, there, who.UID)
	if err != nil {
		return settings.Change{}, err
	}
	have, state := settings.PackageBlock(text, k.Block)
	if state == settings.BlockDamaged {
		return settings.Change{}, fmt.Errorf("%s holds a damaged %s block, a start with no end or two blocks, not touched: delete %s of %s, then apply again", file, k.Block, strings.Join(settings.DamagedLines(text, k.Block), " and "), file)
	}
	updated, changed := settings.UpsertPackageBlock(text, k.Block, body)
	// invariant: the record names the file before its block is written, so
	// a run cut between the two leaves off a block it knows, never one it
	// does not: a recorded block that never landed reads gone, and the block
	// of station's it would have replaced stays station's until it lands.
	// What station made, the file or its folder, stays recorded as made
	// across applies.
	now := written{File: file, Body: body, MadeFile: !there, MadeFolder: madeFolder}
	var olds []written
	for _, w := range rec.Files {
		if w.File != file {
			olds = append(olds, w)
			continue
		}
		now.MadeFile = now.MadeFile || w.MadeFile
		now.MadeFolder = now.MadeFolder || w.MadeFolder
		if state == settings.BlockHere && w.ours(have) && have != body {
			now.Was = have
		}
	}
	err = a.save(record{Files: append(slices.Clone(olds), now)})
	if err != nil {
		return settings.Change{}, err
	}
	recorded = true
	if changed || !there {
		err = f.Replace(keysFile, local.Seen{Data: []byte(text), There: there}, []byte(updated), 0o600, who.Owner)
		if err != nil {
			return settings.Change{}, a.linked(err)
		}
	}
	err = a.strict(f)
	if err != nil {
		return settings.Change{}, err
	}
	detail := "written"
	if !changed && there {
		detail = "already there"
	}
	steps := []settings.ChangeStep{{Label: file, State: catalog.StateOK, Detail: detail}}
	var kept []written
	var stays []error
	for _, w := range olds {
		taken, keep, why := a.takeOut(w, held[w.File], k.Block)
		steps = append(steps, taken...)
		if keep {
			kept = append(kept, w)
		}
		var mine *edited
		if !errors.As(why, &mine) {
			stays = append(stays, why)
		}
	}
	now.Was = ""
	err = a.save(record{Files: append(kept, now)})
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
		return a.unrecorded(ctx, k.Block, configured(it))
	}
	files := make([]string, 0, len(rec.Files))
	for _, w := range rec.Files {
		files = append(files, w.File)
	}
	held, letGo, err := a.holdAll(ctx, files)
	if err != nil {
		return settings.Change{}, err
	}
	defer letGo()
	var steps []settings.ChangeStep
	var kept []written
	var stays []error
	for _, w := range rec.Files {
		taken, keep, why := a.takeOut(w, held[w.File], k.Block)
		if taken[0].Detail != "already gone" {
			steps = append(steps, taken...)
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

type loosened struct {
	// value: why the keys are ignored, at what path, and the mode and owner
	// Machines sets there against what is there now; why empty when strict.
	why, at, want, have string
}

func (a *Applier) loose(file, name string) (loosened, error) {
	// invariant: the keys count only while the sign-in server reads them:
	// neither the folder nor the file writable by group or others; and
	// both are the home owner's, the rule apply refuses by. Anything else is
	// drift, an owner with the owner's own chown, a loose mode with apply as
	// its fix.
	base, ssh, err := a.reach(file)
	if err != nil {
		return loosened{}, err
	}
	owner, err := a.home.Lstat(base, path.Dir(ssh))
	if err != nil {
		return loosened{}, a.linked(err)
	}
	f, err := a.home.Open(base, ssh)
	if err != nil {
		return loosened{}, a.linked(err)
	}
	defer f.Close()
	folder, err := f.Stat()
	if err != nil {
		return loosened{}, err
	}
	keys, err := f.Lstat(keysFile)
	if err != nil {
		return loosened{}, a.linked(err)
	}
	for _, p := range []struct {
		at string
		e  local.Entry
	}{{path.Dir(file), folder}, {file, keys}} {
		l := loosened{at: p.at}
		perm := p.e.Mode.Perm()
		switch why := notTheHomes(p.at, p.e, owner.UID); {
		case why != "":
			l.want, l.have = fmt.Sprintf("uid %d\n", owner.UID), fmt.Sprintf("uid %d\n", p.e.UID)
			l.why = why
			return l, nil
		case perm&loose == 0:
			continue
		}
		l.want, l.have = octal(perm&^loose)+"\n", octal(perm)+"\n"
		l.why = l.at + " is " + octal(perm) + ", writable by others, so its keys are ignored: station settings apply " + name + " sets it to " + octal(perm&^loose)
		return l, nil
	}
	return loosened{}, nil
}

// value: the bits of a mode that let group or others write, which make
// the sign-in server ignore the keys.
const loose fs.FileMode = 0o022

// invariant: a folder or file others can write loses that write alone; a
// strict one keeps its own mode, and no owner is ever changed: station
// gives the user only what it makes, and never takes a file from whoever
// owns it.
func (a *Applier) strict(f *local.Folder) error {
	e, err := f.Stat()
	if err != nil {
		return err
	}
	if e.Mode.Perm()&loose != 0 {
		err = f.Chmod(e.Mode.Perm() &^ loose)
		if err != nil {
			return err
		}
	}
	e, err = f.Lstat(keysFile)
	if err != nil {
		return a.linked(err)
	}
	if e.Mode.Perm()&loose == 0 {
		return nil
	}
	return a.linked(f.ChmodFile(keysFile, e.Mode.Perm()&^loose))
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

// value: with no record, the file Observe reads, through the scan off
// takes the block out by: the first that holds a block of id or cannot be
// read but for a permission, else the first a permission keeps from this
// run, else root's.
func (a *Applier) found(id string) (string, error) {
	files, err := a.homes()
	if err != nil {
		return "", err
	}
	denied := ""
	for _, file := range files {
		switch a.sight(file, id) {
		case stop:
			return file, nil
		case shut:
			if denied == "" {
				denied = file
			}
		case pass:
		}
	}
	if denied != "" {
		return denied, nil
	}
	return files[0], nil
}

// value: what the no-record read makes of one file: it holds a block of
// station's or cannot be read but for a permission (stop), a permission
// keeps it from this run (shut), or nothing of station's is there (pass).
type seen int

const (
	pass seen = iota
	stop
	shut
)

func (a *Applier) sight(file, id string) seen {
	text, _, err := a.text(file)
	switch {
	case errors.Is(err, fs.ErrPermission):
		return shut
	case err != nil:
		return stop
	}
	if _, state := settings.PackageBlock(text, id); state != settings.BlockAbsent {
		return stop
	}
	return pass
}

// invariant: with no record, the block's markers are the record: off looks
// in the authorized_keys of every account with a home, root's among them,
// and a block of station's there leaves it, its owner's lines as they
// were, only while every line in it is a key Machines knows; a damaged
// one, or one holding any other line, is kept, so off never reads done
// while a block of station's is left and never deletes a key of the
// owner's.
func (a *Applier) unrecorded(ctx context.Context, id string, known map[string]bool) (settings.Change, error) {
	files, err := a.homes()
	if err != nil {
		return settings.Change{}, err
	}
	held, letGo, err := a.holdAll(ctx, files)
	if err != nil {
		return settings.Change{}, err
	}
	defer letGo()
	var steps []settings.ChangeStep
	var stays []error
	for _, file := range files {
		step, err := a.unrecordedIn(file, held[file], id, known)
		if step != nil {
			steps = append(steps, *step)
		}
		stays = append(stays, err)
	}
	return settings.Change{Steps: steps}, errors.Join(stays...)
}

func (a *Applier) unrecordedIn(file string, h holding, id string, known map[string]bool) (*settings.ChangeStep, error) {
	switch {
	case errors.Is(h.err, fs.ErrNotExist):
		return nil, nil
	case h.err != nil:
		return nil, cannot(h.err)
	}
	text, e, there, err := a.textIn(h.folder)
	if err == nil {
		err = a.owned(h.folder, file, e, there, h.owner)
	}
	if err != nil {
		return nil, cannot(err)
	}
	body, state := settings.PackageBlock(text, id)
	switch state {
	case settings.BlockAbsent:
		return nil, nil
	case settings.BlockDamaged:
		return nil, &edited{fmt.Errorf("%s holds a damaged %s block Machines has no record of, not touched: delete %s of %s, then the rest of the block", file, id, strings.Join(settings.DamagedLines(text, id), " and "), file)}
	case settings.BlockHere:
	}
	first, last := settings.BlockLines(text, id)
	var strange []string
	for i, line := range strings.Split(strings.TrimSuffix(body, "\n"), "\n") {
		if strings.TrimSpace(line) != "" && !known[identity(line)] {
			strange = append(strange, "line "+strconv.Itoa(first+1+i))
		}
	}
	if len(strange) > 0 {
		return nil, &edited{fmt.Errorf("%s holds a %s block Machines has no record of, kept: %s of %s is no key Machines knows, so the block's keys still sign in; keep what is yours out of it, then delete lines %d to %d of %s", file, id, strings.Join(strange, " and "), file, first, last, file)}
	}
	updated, _ := settings.RemovePackageBlock(text, id)
	err = h.folder.Replace(keysFile, local.Seen{Data: []byte(text), There: true}, []byte(updated), 0o600, local.Owner{})
	if err != nil {
		return nil, a.linked(err)
	}
	return &settings.ChangeStep{Label: file, State: catalog.StateOK, Detail: "removed, found with no record"}, nil
}

// value: the authorized_keys of every account of /etc/passwd with a home
// of its own, root's always, each once, whatever its shell or uid: off
// reaches every file a block can be in, though apply refuses some of them.
// No /etc/passwd is root alone; one that cannot be read refuses the run.
func (a *Applier) homes() ([]string, error) {
	files := []string{"/root/.ssh/" + keysFile}
	data, err := a.files.ReadFile(a.at("/etc/passwd"))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return files, nil
	case err != nil:
		return nil, fmt.Errorf("/etc/passwd cannot be read, nothing touched: Machines finds every account's keys through it (%w)", err)
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		f := strings.Split(line, ":")
		if len(f) < 7 || !path.IsAbs(f[5]) || path.Clean(f[5]) == "/" {
			continue
		}
		file := path.Join(path.Clean(f[5]), ".ssh", keysFile)
		if !slices.Contains(files, file) {
			files = append(files, file)
		}
	}
	return files, nil
}

// value: the keys the setting is configured with, its key input's
// default, each by its type and blob: the only lines the no-record scan
// takes as Machines'.
func configured(it catalog.Setting) map[string]bool {
	out := map[string]bool{}
	for _, in := range it.Inputs {
		if in.Name != InputKey {
			continue
		}
		body, err := Keys(in.Default)
		if err != nil {
			continue
		}
		for line := range strings.SplitSeq(strings.TrimSuffix(body, "\n"), "\n") {
			out[identity(line)] = true
		}
	}
	return out
}

func identity(line string) string {
	f := strings.Fields(line)
	if len(f) < 2 {
		return line
	}
	return f[0] + " " + f[1]
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
	err := a.above(who.home)
	if err != nil {
		return "", err
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
	case e.Mode.Perm()&0o022 != 0:
		return "", fmt.Errorf("%s is writable by others (%s), not touched: sshd ignores keys under it, take the write away with %s", who.home, octal(e.Mode), catalog.AsRoot("chmod go-w "+who.home))
	}
	return base, nil
}

// invariant: every folder from / down to the one that holds home, / itself
// aside, is root's (in a world, the run's) and writable by no one else:
// whoever else could change one could swap the home under Machines.
func (a *Applier) above(home string) error {
	system := 0
	if a.o.Run != nil {
		system = a.o.Run.UID
	}
	at := "/"
	for name := range strings.SplitSeq(strings.Trim(path.Dir(home), "/"), "/") {
		if name == "" {
			continue
		}
		e, err := a.home.Lstat(a.at(at), name)
		at = path.Join(at, name)
		switch {
		case err != nil:
			return a.linked(err)
		case e.Mode&fs.ModeSymlink != 0:
			return fmt.Errorf("%s is a link, not touched: Machines follows no link to a home", at)
		case !e.Mode.IsDir():
			return fmt.Errorf("%s is not a folder, not touched", at)
		case e.UID != system:
			return fmt.Errorf("%s is not root's (its owner is uid %d), not touched: Machines writes keys only under folders root alone can change", at, e.UID)
		case e.Mode.Perm()&0o022 != 0:
			return fmt.Errorf("%s is writable by others (%s), not touched: Machines writes keys only under folders root alone can change", at, octal(e.Mode))
		}
	}
	return nil
}

// invariant: a key goes only to an account that signs in: one whose shell
// is nologin or false never does, and a key there would still open its
// tunnels; a system account, below the first person's uid, is refused
// too; root is always taken.
func signsIn(who account) error {
	if who.name == "root" {
		return nil
	}
	switch path.Base(who.shell) {
	case "nologin", "false":
		return fmt.Errorf("%s does not sign in (its shell is %s), not touched: Machines writes keys only for an account that signs in", who.name, who.shell)
	}
	if who.uid < firstPerson {
		return fmt.Errorf("%s is a system account (uid %d), not touched: Machines writes keys only for root or a person's account, uid %d and above", who.name, who.uid, firstPerson)
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

// value: err, a link or a hard link met on the way said in the owner's
// words, by the path the system names it.
func (a *Applier) linked(err error) error {
	var pe *fs.PathError
	if !errors.As(err, &pe) {
		return err
	}
	at := path.Join("/", strings.TrimPrefix(pe.Path, path.Join("/", a.o.Root)))
	switch {
	case errors.Is(err, local.ErrLink):
		return fmt.Errorf("%s is a link, not touched: Machines follows no link in a home", at)
	case errors.Is(err, local.ErrHardLink):
		return fmt.Errorf("%s has another name, a hard link, not touched: Machines writes keys only in a file of one name", at)
	case errors.Is(err, local.ErrTooLarge):
		return fmt.Errorf("%s is larger than 1 MiB, not touched: Machines reads an authorized_keys of 1 MiB at most", at)
	case errors.Is(err, local.ErrChanged):
		return fmt.Errorf("%s changed since Machines read it, not touched: run it again to judge it as it is now", at)
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
			return a.as(account{name: user, home: path.Clean(f[5]), shell: f[6], uid: uid, Owner: local.Owner{UID: uid, GID: gid}}), nil
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
