package dotfile

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type Files interface {
	// value: the file opened to be read, so a read stops at its bound.
	Open(path string) (fs.File, error)
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	// value: the file replaced whole, at once, with its own mode and
	// owner; one not there yet is made with perm.
	Replace(path string, data []byte, perm fs.FileMode) error
	// value: Replace, only while the file still holds what was seen
	// (local.ErrChanged), and never by an overwrite in place.
	Swap(path string, seen local.Seen, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Chmod(path string, perm fs.FileMode) error
	Remove(path string) error
	Stat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Readlink(path string) (string, error)
	// invariant: an exclusive lock on path, made when missing, held until
	// release; a second holder waits for it until ctx ends.
	Hold(ctx context.Context, path string) (release func(), err error)
}

type Runner interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Options struct {
	Home string
	// value: station's own copies of the files, the embedded dotfiles.
	Ours fs.FS
	// value: the record of what each fragment wrote, beside the ledger.
	Record string
	// value: station's own folders on this run, absolute, where its XDG
	// variables put them; no setting writes in one.
	Own []string
}

type Applier struct {
	files Files
	run   Runner
	o     Options
}

var _ contract.Applier = (*Applier)(nil)

func New(files Files, run Runner, o Options) *Applier { return &Applier{files: files, run: run, o: o} }

func (*Applier) Kind() catalog.EffectKind { return catalog.EffectDotfile }

func (a *Applier) of(it catalog.Setting) (catalog.Dotfiles, error) {
	d, ok := it.Effect.(catalog.Dotfiles)
	if !ok || len(d.Files) == 0 {
		return catalog.Dotfiles{}, errors.New(it.Name + " is not files Machines writes")
	}
	// invariant: decode confines every file, and so does each call here
	// again, so a setting built past decode still writes nothing out of
	// the home or into station's own, where this run's XDG variables put
	// it too.
	for _, c := range d.Files {
		if problem := catalog.Confined(c.To, d.Mode); problem != "" {
			return catalog.Dotfiles{}, errors.New(it.Name + ": " + problem)
		}
		for _, own := range a.o.Own {
			if within(expand(a.o.Home, c.To), own) {
				return catalog.Dotfiles{}, fmt.Errorf("%s: file %s is in %s, station's own: no setting writes there", it.Name, c.To, own)
			}
		}
	}
	return d, nil
}

// value: whether p is root or in it, without case, as a macOS volume
// opens names.
func within(p, root string) bool {
	return root != "" && (strings.EqualFold(p, root) || len(p) > len(root) && strings.EqualFold(p[:len(root)+1], root+"/"))
}

// value: what one file of a setting holds of it now: there at all, exactly
// as Machines writes it, what its fingerprint covers, and the diff apply
// would make.

type seen struct {
	there, same bool
	// value: the file and pointer of each value of the owner's the
	// fragment sets otherwise, kept.
	kept     []string
	material []byte
	diff     string
}

func (a *Applier) Observe(_ context.Context, it catalog.Setting) (settings.Observation, error) {
	// invariant: the fingerprint covers each file of the setting that is
	// there, in its order: an edit or a removal of one station wrote moves
	// it, and a file the setting gains, not there yet, does not, so a
	// setting otherwise as station last wrote it reads as station's to write.
	// A fragment's covers only what it writes there: the owner's other keys
	// and lines never move it.
	d, err := a.of(it)
	if err != nil {
		return settings.Observation{}, err
	}
	r := record{}
	if d.Fragment() {
		if r, err = a.read(); err != nil {
			return settings.Observation{}, err
		}
	}
	o := settings.Observation{Where: d.Files[0].To}
	sum := sha256.New()
	present, differ := 0, false
	var diffs strings.Builder
	var kept []string
	for _, c := range d.Files {
		s, err := a.see(it.Name, d.Mode, c, r)
		if err != nil {
			return settings.Observation{}, err
		}
		if s.there {
			present++
			sum.Write(s.material)
		}
		differ = differ || !s.same
		diffs.WriteString(s.diff)
		kept = append(kept, s.kept...)
	}
	o.Digest, o.Diff, o.Loose = "sha256:"+hex.EncodeToString(sum.Sum(nil)), diffs.String(), d.Fragment()
	switch {
	case present == len(d.Files) && differ:
		o.State, o.Detail = catalog.StateDegraded, "differs from Machines': station settings apply "+it.Name
	case len(kept) > 0:
		o.State, o.Detail = catalog.StateDegraded, "kept: "+strings.Join(kept, ", ")+" still yours, not this setting's value, so it does not hold"
		o.Kept = o.Detail
	case present == len(d.Files):
		o.State, o.Detail = catalog.StateOK, d.Files[0].To
	case present == 0:
		o.State = catalog.StateAbsent
	default:
		o.State, o.Detail = catalog.StateDegraded, "some files are missing"
	}
	return o, nil
}

func (a *Applier) see(setting string, mode catalog.WriteMode, c catalog.Copy, r record) (seen, error) {
	switch mode {
	case catalog.WriteMerge:
		w, _ := r.of(setting, c.To)
		return a.seeMerge(c, w.Values)
	case catalog.WriteBlock:
		return a.seeBlock(setting, c)
	case catalog.WriteFile:
	}
	want, err := fs.ReadFile(a.o.Ours, c.From)
	if err != nil {
		return seen{}, err
	}
	have, err := a.files.ReadFile(expand(a.o.Home, c.To))
	s := seen{same: bytes.Equal(want, have)}
	switch {
	case err == nil:
		s.there, s.material = true, append([]byte(c.To+"\x00"), have...)
	case !errors.Is(err, fs.ErrNotExist):
		return seen{}, err
	}
	if !s.same {
		s.diff = settings.UnifiedDiff(string(want), string(have), "Machines' "+c.To, c.To)
	}
	return s, nil
}

func (a *Applier) Apply(ctx context.Context, it catalog.Setting, _ map[string]string) (settings.Change, error) {
	d, err := a.of(it)
	if err != nil {
		return settings.Change{}, err
	}
	var change settings.Change
	if d.Fragment() {
		err = a.locked(ctx, func(r *record) error {
			for _, c := range d.Files {
				steps, failed := a.swapped(c.To, func() ([]settings.ChangeStep, error) { return a.write(r, it.Name, d.Mode, c) })
				change.Steps = append(change.Steps, steps...)
				if failed != nil {
					return failed
				}
			}
			return nil
		})
	} else {
		err = a.whole(d, &change)
	}
	if err != nil {
		return change, err
	}
	if len(d.Reload) > 0 {
		change.Steps = append(change.Steps, a.reload(ctx, d.Reload)...)
	}
	return change, nil
}

func (a *Applier) whole(d catalog.Dotfiles, change *settings.Change) error {
	for _, c := range d.Files {
		data, err := fs.ReadFile(a.o.Ours, c.From)
		if err != nil {
			return err
		}
		mode := c.Mode
		if mode == 0 {
			mode = 0o600
		}
		full := expand(a.o.Home, c.To)
		if err := a.files.MkdirAll(path.Dir(full), 0o750); err != nil {
			return err
		}
		if err := a.files.WriteFile(full, data, mode); err != nil {
			return err
		}
		if err := a.files.Chmod(full, mode); err != nil {
			return err
		}
		change.Steps = append(change.Steps, settings.ChangeStep{Label: c.To, State: catalog.StateOK, Detail: "written"})
	}
	return nil
}

func (a *Applier) write(r *record, setting string, mode catalog.WriteMode, c catalog.Copy) ([]settings.ChangeStep, error) {
	if mode == catalog.WriteMerge {
		return a.applyMerge(r, setting, c)
	}
	return a.applyBlock(r, setting, c)
}

func (a *Applier) Undo(ctx context.Context, it catalog.Setting) (settings.Change, error) {
	// invariant: Undo removes a file only while it is still exactly station's,
	// and from a file of the owner's only the fragment station wrote and he
	// left as it is: off never destroys a change of the owner's.
	d, err := a.of(it)
	if err != nil {
		return settings.Change{}, err
	}
	var change settings.Change
	if d.Fragment() {
		err = a.locked(ctx, func(r *record) error {
			for _, c := range d.Files {
				steps, failed := a.swapped(c.To, func() ([]settings.ChangeStep, error) { return a.takeBack(r, it.Name, d.Mode, c) })
				change.Steps = append(change.Steps, steps...)
				if failed != nil {
					return failed
				}
			}
			return nil
		})
		return change, err
	}
	for _, c := range d.Files {
		want, err := fs.ReadFile(a.o.Ours, c.From)
		if err != nil {
			return change, err
		}
		full := expand(a.o.Home, c.To)
		have, err := a.files.ReadFile(full)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			return change, err
		case !bytes.Equal(want, have):
			change.Steps = append(change.Steps, settings.ChangeStep{Label: c.To, State: catalog.StateOK, Detail: "yours, kept"})
			continue
		}
		if err := a.files.Remove(full); err != nil {
			return change, err
		}
		change.Steps = append(change.Steps, settings.ChangeStep{Label: c.To, State: catalog.StateOK, Detail: "removed"})
	}
	return change, nil
}

func (a *Applier) takeBack(r *record, setting string, mode catalog.WriteMode, c catalog.Copy) ([]settings.ChangeStep, error) {
	if mode == catalog.WriteMerge {
		return a.undoMerge(r, setting, c)
	}
	return a.undoBlock(r, setting, c)
}

// value: how many times a fragment is judged again over a file another
// writer changed between its read and its write, before it is refused.
const swaps = 3

// invariant: a step writes the file only while it holds what the step
// read (Swap); one changed meanwhile by a writer outside the lock (an
// editor, the app itself) is read and judged again, at most swaps times,
// then refused, the file left as that writer made it. The record moves
// only once the file is written, so a step judged again starts from it
// as it was.
func (a *Applier) swapped(to string, step func() ([]settings.ChangeStep, error)) ([]settings.ChangeStep, error) {
	for range swaps {
		steps, err := step()
		if !errors.Is(err, local.ErrChanged) {
			return steps, err
		}
	}
	return nil, fmt.Errorf("%s changed under Machines %d times between its read and its write: nothing written; apply again once what writes it is done", to, swaps)
}

// value: the file as a step read it, which its write must still find.
func seenAs(data []byte, there bool) local.Seen { return local.Seen{Data: data, There: there} }

// value: how many links a path is followed through before it is refused,
// the kernel's own bound on a path.
const linkHops = 32

// value: the marker, after a link's own parts, that what the link names
// must be there.
const named = "\x00"

func (a *Applier) real(p string) (string, error) {
	// invariant: the real path p names: each part of it read from the root
	// down, a link replaced by what it names (`..` in it climbs from the
	// real folder, never the lexical one), at most linkHops links in all;
	// a link that names nothing is refused, and a part not there yet, past
	// every link, is taken as named.
	todo := strings.Split(p, "/")
	at, hops := "/", 0
	for len(todo) > 0 {
		part := todo[0]
		todo = todo[1:]
		switch part {
		case "", ".":
			continue
		case "..":
			at = path.Dir(at)
			continue
		case named:
			if _, err := a.files.Stat(at); errors.Is(err, fs.ErrNotExist) {
				return "", fmt.Errorf("%s: a dangling link: what it names, %s, is not there, and Machines makes nothing through it", p, at)
			}
			continue
		}
		next := path.Join(at, part)
		to, err := a.files.Readlink(next)
		if err != nil {
			at = next
			continue
		}
		hops++
		if hops > linkHops {
			return "", fmt.Errorf("%s: more than %d links", p, linkHops)
		}
		if path.IsAbs(to) {
			at = "/"
		}
		todo = append(append(strings.Split(to, "/"), named), todo...)
	}
	return at, nil
}

// value: the real path of a setting's file, and its bytes, nil and false
// when it is not there yet.
func (a *Applier) target(to string, mode catalog.WriteMode) (at string, data []byte, there bool, err error) {
	// invariant: a fragment lands in the file its links really name, and
	// only inside the real home and out of what Confined keeps it from and
	// station's own folders of this run, each compared by its own real
	// path: a link out of the home, or to a kept file or folder, or a kept
	// folder that is itself a link to the one named, is refused, never
	// followed.
	home, err := a.real(a.o.Home)
	if err != nil {
		return "", nil, false, err
	}
	if at, err = a.real(expand(a.o.Home, to)); err != nil {
		return "", nil, false, err
	}
	rest, inside := strings.CutPrefix(at, home+"/")
	switch {
	case !inside:
		return "", nil, false, fmt.Errorf("%s leads through its links to %s, outside your home: Machines writes nothing there", to, at)
	case catalog.Confined("~/"+rest, mode) != "":
		return "", nil, false, fmt.Errorf("%s leads through its links to ~/%s: %s", to, rest, catalog.Confined("~/"+rest, mode))
	}
	for _, k := range append(catalog.Kept(mode), a.o.Own...) {
		// invariant: a kept path whose links name nothing keeps its name.
		really, unreal := a.real(expand(a.o.Home, k))
		if unreal != nil {
			really = expand(a.o.Home, k)
		}
		if within(at, really) {
			return "", nil, false, fmt.Errorf("%s is really %s, in %s: Machines writes no fragment there", to, at, k)
		}
	}
	info, err := a.files.Stat(at)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return at, nil, false, nil
	case err != nil:
		return "", nil, false, err
	case !info.Mode().IsRegular():
		return "", nil, false, fmt.Errorf("%s is no regular file: Machines writes no fragment into it", to)
	case info.Size() > maxTarget:
		return "", nil, false, fmt.Errorf("%s is past 1 MiB: Machines writes no fragment into a file that large", to)
	}
	data, err = atMost(a.files.Open, at, maxTarget)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return at, nil, false, nil
	case err == nil && len(data) > maxTarget:
		return "", nil, false, fmt.Errorf("%s is past 1 MiB: Machines writes no fragment into a file that large", to)
	}
	return at, data, err == nil, err
}

// invariant: a file measured before its read is read through a reader
// that stops one byte past its bound, so one grown since is refused
// without being read whole.
func atMost(open func(string) (fs.File, error), name string, bound int) ([]byte, error) {
	f, err := open(name)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(f, int64(bound)+1))
	return data, errors.Join(err, f.Close())
}

// invariant: a file a fragment makes is the owner's from then on: made
// 0600 with the folders it lacks, which are recorded with it, so off takes
// them back only once they are empty.
func (a *Applier) land(r *record, w *written, at string, data []byte, seen local.Seen) error {
	home, err := a.real(a.o.Home)
	if err != nil {
		return err
	}
	var made []string
	for dir := path.Dir(at); dir != "/" && dir != "." && dir != home; dir = path.Dir(dir) {
		if _, err := a.files.Stat(dir); err == nil {
			break
		}
		made = append(made, dir)
	}
	next := *w
	next.Made = w.Made || !seen.There
	next.Folders = slices.Clone(w.Folders)
	for _, dir := range made {
		if !slices.Contains(next.Folders, dir) {
			next.Folders = append(next.Folders, dir)
		}
	}
	// invariant: the record is measured with all the landing adds to it,
	// the file and the folders made, before anything is made.
	if err := a.fits(r, next); err != nil {
		return err
	}
	if err := a.files.MkdirAll(path.Dir(at), 0o750); err != nil {
		return err
	}
	if err := a.files.Swap(at, seen, data, 0o600); err != nil {
		// invariant: a swap that fails once the file has landed (its
		// folder's sync) leaves it written: it is recorded, so off takes it
		// back.
		if !errors.Is(err, local.ErrChanged) && a.holds(at, data) {
			*w = next
			r.put(next)
			return fmt.Errorf("%s is written, and recorded so off takes it back, but its write did not end: %w", at, err)
		}
		// invariant: a try that does not land takes back the folders it
		// made, deepest first, while they are empty, so the next try makes
		// and records them again.
		for _, dir := range made {
			_ = a.files.Remove(dir)
		}
		return err
	}
	*w = next
	return nil
}

// value: whether the file at holds exactly data.
func (a *Applier) holds(at string, data []byte) bool {
	now, err := atMost(a.files.Open, at, maxTarget)
	return err == nil && bytes.Equal(now, data)
}

// value: whether the file a setting names is itself no link, so the record
// may say Machines made it: Machines makes no file through a link.
func (a *Applier) direct(to string) bool {
	_, err := a.files.Readlink(expand(a.o.Home, to))
	return err != nil
}

// value: a file Machines made, and the folders it made for it, removed
// once nothing of the owner's is in them.
func (a *Applier) unmake(w written, at string, seen []byte) error {
	// invariant: the record says which folders Machines made, and only one
	// that holds the file, inside the real home and not the home itself, is
	// taken at its word: any other the record names is left.
	home, err := a.real(a.o.Home)
	if err != nil {
		return err
	}
	// invariant: the file is read again just before it is removed, and one
	// that no longer holds what the step read is left (local.ErrChanged), so
	// swapped judges it again: off never removes another writer's change.
	now, err := atMost(a.files.Open, at, maxTarget)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return err
	case !bytes.Equal(now, seen):
		return local.ErrChanged
	default:
		if err := a.files.Remove(at); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	for _, dir := range w.Folders {
		if !strings.HasPrefix(at, dir+"/") || !strings.HasPrefix(dir, home+"/") {
			continue
		}
		if entries, err := a.files.ReadDir(dir); err == nil && len(entries) == 0 {
			if err := a.files.Remove(dir); err != nil {
				return err
			}
		}
	}
	return nil
}

const reloadBudget = 5 * time.Second

// outside: a reload that fails is no failure: tmux with no server running
// exits non-zero, and the new file is read at the next start.
func (a *Applier) reload(ctx context.Context, argv []string) []settings.ChangeStep {
	args := make([]string, len(argv)-1)
	for i, arg := range argv[1:] {
		args[i] = expand(a.o.Home, arg)
	}
	out, err := a.run.Output(ctx, local.Cmd{Name: argv[0], Args: args, Timeout: reloadBudget})
	if err != nil || out.ExitCode != 0 {
		return nil
	}
	return []settings.ChangeStep{{Label: argv[0], State: catalog.StateOK, Detail: "reloaded"}}
}

func expand(home, p string) string {
	if p == "~" {
		return home
	}
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		return path.Join(home, rest)
	}
	return p
}
